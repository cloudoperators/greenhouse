// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package imagemirror

import (
	"regexp"
	"slices"

	"github.com/google/go-containerregistry/pkg/name"
)

var imageFieldPattern = regexp.MustCompile(`(?m)^[\s-]*image:\s+["']?([^\s"']+)["']?`)

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

// SplitImageRef breaks an image ref into registry, repository, and tag/digest.
func SplitImageRef(imageRef string) (registry, repository, tagOrDigest string) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "docker.io", imageRef, ""
	}

	// name.ParseReference normalizes Docker Hub to "index.docker.io",
	// but our mirror config uses "docker.io" as the map key.
	registry = ref.Context().RegistryStr()
	if registry == name.DefaultRegistry {
		registry = "docker.io"
	}

	repository = ref.Context().RepositoryStr()

	// ref.Identifier() doesnt include the separator so we prepend ":" or "@".
	switch r := ref.(type) {
	case name.Tag:
		tagOrDigest = ":" + r.TagStr()
	case name.Digest:
		tagOrDigest = "@" + r.DigestStr()
	}

	return registry, repository, tagOrDigest
}
