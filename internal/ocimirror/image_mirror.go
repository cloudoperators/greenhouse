// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// manifestFetcherFunc is a function type matching the crane.Manifest signature.
type manifestFetcherFunc func(ref string, opts ...crane.Option) ([]byte, error)

// ImageMirror handles image replacement and OCI artifact replication for configured registry mirrors.
type ImageMirror struct {
	config          *RegistryMirrorConfig
	auth            authn.Authenticator
	manifestFetcher manifestFetcherFunc
}

// ImageTransform represents an upstream-to-mirror image reference rewrite.
type ImageTransform struct {
	Original string
	Mirrored string
}

// NewImageMirror creates an ImageMirror with credentials from the configured Secret.
func NewImageMirror(ctx context.Context, k8sClient client.Client, config *RegistryMirrorConfig, namespace string) (*ImageMirror, error) {
	auth := authn.Anonymous
	if config.SecretName != "" {
		a, err := getAuthFromSecret(ctx, k8sClient, config.SecretName, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read registry mirror credentials from secret %s/%s: %w", namespace, config.SecretName, err)
		}
		auth = a
	}

	return &ImageMirror{
		config:          config,
		auth:            auth,
		manifestFetcher: crane.Manifest,
	}, nil
}

// NewImageMirrorForTest creates an ImageMirror for testing with explicit auth and manifest fetcher.
func NewImageMirrorForTest(config *RegistryMirrorConfig, auth authn.Authenticator, fetcher func(ref string, opts ...crane.Option) ([]byte, error)) *ImageMirror {
	return &ImageMirror{
		config:          config,
		auth:            auth,
		manifestFetcher: fetcher,
	}
}

// EnsureChartReplicated warms the pull-through cache for a chart on a mirror domain.
func (m *ImageMirror) EnsureChartReplicated(ctx context.Context, ociRef string) (replicatedRef string, manifest []byte, err error) {
	replicationRef, returnRef := m.buildReplicationRef(ociRef)
	if replicationRef == "" {
		return "", nil, nil
	}
	manifest, err = m.triggerReplication(ctx, replicationRef)
	return returnRef, manifest, err
}

// EnsureImageReplicated warms the pull-through cache for a container image, including upstream refs.
func (m *ImageMirror) EnsureImageReplicated(ctx context.Context, ociRef string) (replicatedRef string, manifest []byte, err error) {
	if mirroredRef := m.buildMirroredOCIRef(ociRef); mirroredRef != "" {
		manifest, err = m.triggerReplication(ctx, mirroredRef)
		return mirroredRef, manifest, err
	}
	return m.EnsureChartReplicated(ctx, ociRef)
}

// buildReplicationRef resolves an OCI ref on a mirror domain to the primaryMirror ref for replication.
func (m *ImageMirror) buildReplicationRef(ociRef string) (replicationRef, returnRef string) {
	registry, repo, tagOrDigest := SplitOCIRef(ociRef)

	if registry == m.config.PrimaryMirror {
		return ociRef, ociRef
	}

	for _, mirror := range m.config.RegistryMirrors {
		if mirror.BaseDomain == registry {
			primaryRef := fmt.Sprintf("%s/%s", m.config.PrimaryMirror, repo) + tagOrDigest
			return primaryRef, ociRef
		}
	}

	return "", ""
}

// BuildImageTransformations extracts image refs from the given rendered manifest
// sets and returns the upstream-to-mirror rewrites for them, skipping refs that
// already point at a mirror and deduplicating across sets.
func (m *ImageMirror) BuildImageTransformations(manifestSets ...string) []ImageTransform {
	seen := make(map[string]struct{})
	var transforms []ImageTransform
	for _, manifests := range manifestSets {
		for _, imageRef := range ExtractUniqueOCIRefs(manifests) {
			resolved := m.config.ResolveOCIRef(imageRef)
			if resolved == nil {
				continue
			}

			original := fmt.Sprintf("%s/%s", resolved.Registry, resolved.Repository)
			if _, ok := seen[original]; ok {
				continue
			}
			seen[original] = struct{}{}

			mirrored := fmt.Sprintf("%s/%s/%s", resolved.Mirror.BaseDomain, resolved.Mirror.SubPath, resolved.Repository)
			transforms = append(transforms, ImageTransform{
				Original: original,
				Mirrored: mirrored,
			})
		}
	}

	return transforms
}

// ReplicateOCIArtifacts triggers replication for OCI artifacts found in the
// given manifest sets, skipping already-replicated refs and deduplicating
// across sets.
func (m *ImageMirror) ReplicateOCIArtifacts(ctx context.Context, alreadyReplicated []string, manifestSets ...string) ([]string, error) {
	seen := make(map[string]struct{})
	var imageRefs []string
	for _, manifests := range manifestSets {
		for _, ref := range ExtractUniqueOCIRefs(manifests) {
			if _, ok := seen[ref]; ok {
				continue
			}
			seen[ref] = struct{}{}
			imageRefs = append(imageRefs, ref)
		}
	}
	if len(imageRefs) == 0 {
		return nil, nil
	}

	alreadySet := make(map[string]struct{}, len(alreadyReplicated))
	for _, img := range alreadyReplicated {
		alreadySet[img] = struct{}{}
	}

	var replicated []string
	var replicationErrors []error
	for _, imageRef := range imageRefs {
		if _, ok := alreadySet[imageRef]; ok {
			replicated = append(replicated, imageRef)
			continue
		}

		replicatedRef, _, err := m.EnsureImageReplicated(ctx, imageRef)
		if err != nil {
			replicationErrors = append(replicationErrors, fmt.Errorf("%s: %w", imageRef, err))
			continue
		}
		if replicatedRef == "" {
			continue
		}

		replicated = append(replicated, imageRef)
	}

	sort.Strings(replicated)

	return replicated, utilerrors.NewAggregate(replicationErrors)
}

// buildMirroredOCIRef resolves imageRef against the mirror config and returns the mirrored reference.
func (m *ImageMirror) buildMirroredOCIRef(imageRef string) string {
	resolved := m.config.ResolveOCIRef(imageRef)
	if resolved == nil {
		return ""
	}

	mirroredRef := fmt.Sprintf("%s/%s/%s", m.config.PrimaryMirror, resolved.Mirror.SubPath, resolved.Repository)
	if resolved.TagOrDigest != "" {
		mirroredRef += resolved.TagOrDigest
	}

	return mirroredRef
}

// triggerReplication fetches the manifest for the given ref to warm the pull-through cache.
func (m *ImageMirror) triggerReplication(ctx context.Context, ref string, extraOpts ...crane.Option) ([]byte, error) {
	log.FromContext(ctx).V(1).Info("triggering replication", "ref", ref)
	opts := append([]crane.Option{
		crane.WithAuth(m.auth),
		crane.WithPlatform(&v1.Platform{OS: "linux", Architecture: "amd64"}),
	}, extraOpts...)
	manifest, err := m.manifestFetcher(ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest for %s: %w", ref, err)
	}

	return manifest, nil
}

func getAuthFromSecret(ctx context.Context, k8sClient client.Client, secretName, namespace string) (authn.Authenticator, error) {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	username := string(secret.Data["username"])
	password := string(secret.Data["password"])

	if username == "" || password == "" {
		return nil, fmt.Errorf("secret %s/%s missing username or password", namespace, secretName)
	}

	return &authn.Basic{
		Username: username,
		Password: password,
	}, nil
}
