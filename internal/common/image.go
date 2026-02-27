// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"regexp"
	"slices"
	"strings"
)

const (
	dockerHubRegistry         = "docker.io"
	dockerHubLibraryNamespace = "library"
)

var (
	imageFieldPattern = regexp.MustCompile(`(?m)^[\s-]*image:\s+["']?([^\s"']+)["']?`)
	imageRefPattern   = regexp.MustCompile(`^(?:([a-zA-Z0-9][a-zA-Z0-9.-]*(?:\:[0-9]+)?)/)?([a-zA-Z0-9._/-]+)(?:[:@](.+))?$`)
)

// ExtractUniqueImages extracts and deduplicates all image references from YAML manifests.
func ExtractUniqueImages(manifests string) []string {
	seen := make(map[string]struct{})

	for _, match := range imageFieldPattern.FindAllStringSubmatch(manifests, -1) {
		if len(match) > 1 {
			seen[match[1]] = struct{}{}
		}
	}

	images := make([]string, 0, len(seen))
	for img := range seen {
		images = append(images, img)
	}
	slices.Sort(images)
	return images
}

// SplitImageRef splits an image reference into (registry, repository, tagOrDigest).
// tagOrDigest includes the separator (":tag" or "@sha256:..."), empty if absent.
// Defaults to docker.io when no registry is specified.
func SplitImageRef(imageRef string) (registry, repository, tagOrDigest string) {
	m := imageRefPattern.FindStringSubmatch(imageRef)
	if len(m) < 3 {
		return dockerHubRegistry, dockerHubLibraryOrRepo(imageRef), ""
	}

	registry = m[1]
	repository = m[2]
	if len(m) > 3 && m[3] != "" {
		sep := ":"
		if strings.HasPrefix(m[3], "sha256:") || strings.HasPrefix(m[3], "sha512:") {
			sep = "@"
		}
		tagOrDigest = sep + m[3]
	}

	// First segment without "." or ":" is a namespace, not a registry (e.g. "myorg/nginx").
	if registry != "" && !strings.ContainsAny(registry, ".:") {
		repository = registry + "/" + repository
		registry = ""
	}

	if registry == "" {
		registry = dockerHubRegistry
		repository = dockerHubLibraryOrRepo(repository)
	}

	return registry, repository, tagOrDigest
}
func dockerHubLibraryOrRepo(repo string) string {
	if strings.Contains(repo, "/") {
		return repo
	}
	return dockerHubLibraryNamespace + "/" + repo
}
