// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	retry "github.com/hashicorp/go-retryablehttp"
)

type artifactory struct {
	log             logr.Logger
	client          *retry.Client
	storageBasePath string
}

type IArtifactory interface {
	Save(artifactID, digest string, content []byte) error
	Get(ctx context.Context, artifactID, url, digest string) ([]byte, error)
	GetRaw(ctx context.Context, artifactID, url, digest string) ([]byte, error)
	DeleteAllExcept(artifactID, digest string) error
	Path(artifactID, digest string) string
}

// noopLogger implements retryable http.LeveledLogger using logr
type noopLogger struct{ log logr.Logger }

func (l noopLogger) Error(msg string, keysAndValues ...any) {
	l.log.Error(errors.New(msg), "http error", keysAndValues...)
}
func (l noopLogger) Info(msg string, keysAndValues ...any) {
	l.log.V(1).Info(msg, keysAndValues...)
}
func (l noopLogger) Debug(msg string, keysAndValues ...any) {
	l.log.V(2).Info(msg, keysAndValues...)
}
func (l noopLogger) Warn(msg string, keysAndValues ...any) {
	l.log.V(1).Info("WARN: "+msg, keysAndValues...)
}

// NewArtifactory creates a new Artifactory instance with retryable HTTP client
func NewArtifactory(log logr.Logger, storagePath string, retries int) IArtifactory {
	retryClient := retry.NewClient()
	retryClient.RetryMax = retries              // retry
	retryClient.RetryWaitMin = 3 * time.Second  // initial delay
	retryClient.RetryWaitMax = 15 * time.Second // max backoff delay
	retryClient.Logger = retry.LeveledLogger(noopLogger{log})

	// stop retrying if context is canceled
	retryClient.RequestLogHook = func(_ retry.Logger, req *http.Request, retry int) {
		select {
		case <-req.Context().Done():
			log.Info("context canceled, aborting retries", "url", req.URL)
		default:
		}
	}
	return &artifactory{
		log:             log,
		client:          retryClient,
		storageBasePath: storagePath,
	}
}

func (a *artifactory) Get(ctx context.Context, artifactID, srcURL, digest string) ([]byte, error) {
	// 1. Try local fetch first
	data, err := a.fetchFromFileSystem(artifactID, digest)
	if err == nil {
		a.log.V(1).Info("artifact found in local cache", "artifactID", artifactID, "digest", digest)
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to fetch artifact from filesystem: %w", err)
	}

	a.log.V(1).Info("artifact not found locally, fetching from remote source", "url", srcURL)

	// 2. Fetch from remote source (tar.gz → map[string][]byte)
	files, err := a.fetchFromSource(ctx, srcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artifact from source: %w", err)
	}

	// 3. Marshal map into []byte and return
	content, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artifact content: %w", err)
	}

	a.log.V(1).Info("artifact fetched successfully from remote", "artifactID", artifactID, "digest", digest)
	return content, nil
}

func (a *artifactory) Save(artifactID, digest string, content []byte) error {
	return a.saveToFileSystem(artifactID, digest, content)
}

// GetRaw fetches the artifact bytes verbatim — no gzip/tar extraction.
// Use this when the caller wants the original archive on disk (e.g. a Helm chart .tgz
// to hand to the chart loader). Local cache is checked first.
func (a *artifactory) GetRaw(ctx context.Context, artifactID, srcURL, digest string) ([]byte, error) {
	data, err := a.fetchFromFileSystem(artifactID, digest)
	if err == nil {
		a.log.V(1).Info("artifact found in local cache", "artifactID", artifactID, "digest", digest)
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to fetch artifact from filesystem: %w", err)
	}

	a.log.V(1).Info("artifact not found locally, fetching raw bytes from remote source", "url", srcURL)
	body, err := a.download(ctx, srcURL)
	if err != nil {
		return nil, err
	}
	a.log.V(1).Info("artifact fetched successfully from remote", "artifactID", artifactID, "digest", digest, "bytes", len(body))
	return body, nil
}

func (a *artifactory) DeleteAllExcept(artifactID, digest string) error {
	return a.deleteAllExceptFromFileSystem(artifactID, digest)
}

func (a *artifactory) Path(artifactID, digest string) string {
	return filepath.Join(a.storageBasePath, artifactID, digest)
}

func (a *artifactory) saveToFileSystem(artifactID, digest string, content []byte) error {
	if artifactID == "" {
		return errors.New("artifactID must not be empty")
	}
	if digest == "" {
		return errors.New("digest must not be empty")
	}
	if len(content) == 0 {
		return errors.New("content must not be empty")
	}

	filePath := filepath.Join(a.storageBasePath, artifactID, digest)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Direct write
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write artifact file: %w", err)
	}

	a.log.V(1).Info("artifact saved to disk",
		"path", filePath,
		"digest", digest,
		"artifactID", artifactID,
	)
	return nil
}

func (a *artifactory) deleteAllExceptFromFileSystem(artifactID, keepDigest string) error {
	dir := filepath.Join(a.storageBasePath, artifactID)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			a.log.V(1).Info("no artifact directory found to clean up", "path", dir)
			return nil
		}
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if name == keepDigest {
			continue
		}

		target := filepath.Join(dir, name)
		if err := os.Remove(target); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to delete artifact file %s: %w", target, err)
		}

		a.log.V(1).Info("deleted old artifact digest", "digest", name)
	}

	return nil
}

func (a *artifactory) fetchFromFileSystem(artifactID, digest string) ([]byte, error) {
	if digest == "" {
		return nil, errors.New("digest must not be empty")
	}

	filePath := filepath.Join(a.storageBasePath, artifactID, digest)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read artifact file: %w", err)
	}

	return data, nil
}

func (a *artifactory) fetchFromSource(ctx context.Context, srcURL string) (map[string][]byte, error) {
	body, err := a.download(ctx, srcURL)
	if err != nil {
		return nil, err
	}
	return extractTarGz(body)
}

// download fetches the artifact body from srcURL using the retry client. The returned
// bytes are the raw response body (unchanged). ARTIFACT_DOMAIN rewriting is applied.
func (a *artifactory) download(ctx context.Context, srcURL string) ([]byte, error) {
	srcURL = replaceArtifactDomain(srcURL)
	a.log.V(1).Info("fetching artifact", "srcUrl", srcURL)
	req, err := retry.NewRequestWithContext(ctx, http.MethodGet, srcURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

// extractTarGz unpacks a gzipped tar archive into a map keyed by entry name.
func extractTarGz(body []byte) (map[string][]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	files := make(map[string][]byte)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeReg:
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %q from tar: %w", header.Name, err)
			}
			files[header.Name] = data
		case tar.TypeDir:
			continue
		}
	}
	return files, nil
}

// replaceArtifactDomain rewrites the host portion of the given artifact URL
// if ARTIFACT_DOMAIN is set. It preserves scheme and path.
//
// Examples:
//
//	ARTIFACT_DOMAIN=localhost:5050
//	  → http://localhost:5050/externalartifact/foo/bar.tar.gz
//	ARTIFACT_DOMAIN=http://127.0.0.1:5050
//	  → http://127.0.0.1:5050/externalartifact/foo/bar.tar.gz
func replaceArtifactDomain(artifactURL string) string {
	override, ok := os.LookupEnv("ARTIFACT_DOMAIN")
	if !ok || override == "" {
		// not set, return original URL
		return artifactURL
	}

	parsed, err := url.Parse(artifactURL)
	if err != nil || parsed.Host == "" {
		// fallback to original if parsing fails
		return artifactURL
	}

	// Parse override domain (to check if it has a scheme)
	overrideURL, err := url.Parse(override)
	if err == nil && overrideURL.Host != "" {
		// override has a scheme (e.g., http://localhost:5050)
		parsed.Scheme = overrideURL.Scheme
		parsed.Host = overrideURL.Host
	} else {
		// override is just host[:port] (e.g., localhost:5050)
		parsed.Host = override
		// Keep original scheme
	}

	return parsed.String()
}
