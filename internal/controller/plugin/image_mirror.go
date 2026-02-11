// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"

	"github.com/cloudoperators/greenhouse/internal/common"
)

const (
	// dockerHubRegistry is the default registry for container images.
	dockerHubRegistry = "docker.io"
	// dockerHubLibraryNamespace is the namespace for official Docker Hub images.
	dockerHubLibraryNamespace = "library"
)

var (
	// imageFieldPattern extracts image references from YAML manifests
	imageFieldPattern = regexp.MustCompile(`(?m)^[\s-]*image:\s+["']?([^\s"']+)["']?`)

	// imageRefPattern parses image references: [registry/]repository[:tag|@digest]
	imageRefPattern = regexp.MustCompile(`^(?:([a-zA-Z0-9][a-zA-Z0-9.-]*(?:\:[0-9]+)?)/)?([a-zA-Z0-9._/-]+)(?:[:@](.+))?$`)
)

// CreateRegistryMirrorPostRenderer creates a Kustomize PostRenderer for mirroring container images.
// It extracts all image references from rendered manifests and creates transformations for images
// whose registries are configured in the mirror config.
func createRegistryMirrorPostRenderer(mirrorConfig *common.RegistryMirrorConfig, renderedManifests string) *helmv2.PostRenderer {
	if mirrorConfig == nil || len(mirrorConfig.RegistryMirrors) == 0 {
		return nil
	}

	images := buildImageTransformations(renderedManifests, mirrorConfig)
	if len(images) == 0 {
		return nil
	}

	return &helmv2.PostRenderer{
		Kustomize: &helmv2.Kustomize{
			Images: images,
		},
	}
}

// buildImageTransformations extracts images from manifests and creates mirror transformations.
func buildImageTransformations(manifests string, config *common.RegistryMirrorConfig) []kustomize.Image {
	imageRefs := extractUniqueImages(manifests)

	var transformations []kustomize.Image
	for _, imageRef := range imageRefs {
		registry, repo := splitImageRef(imageRef)

		mirror, hasMirror := config.RegistryMirrors[registry]
		if !hasMirror {
			continue
		}

		// Build base image names without tag/digest.
		// Kustomize will automatically preserve the tag/digest from the manifest.
		baseName := fmt.Sprintf("%s/%s", registry, repo)
		newName := fmt.Sprintf("%s/%s/%s", mirror.BaseDomain, mirror.SubPath, repo)

		transformations = append(transformations, kustomize.Image{
			Name:    baseName,
			NewName: newName,
		})
	}

	return transformations
}

// extractUniqueImages extracts and deduplicates all image references from YAML manifests.
func extractUniqueImages(manifests string) []string {
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

// splitImageRef returns (registry, repository) with tag/digest stripped.
// If registry isn't specified, docker.io is assumed.
// Spec for reference: https://kubernetes.io/docs/concepts/containers/images/
func splitImageRef(imageRef string) (registry, repository string) {
	m := imageRefPattern.FindStringSubmatch(imageRef)
	if len(m) < 3 {
		// Couldn't parse. Fall back to docker.io + "library".
		return dockerHubRegistry, dockerHubLibraryOrRepo(imageRef)
	}

	registry = m[1]
	repository = m[2]

	// If the first segment isn't a domain or host:port, it's a namespace, not a registry.
	// e.g. "myorg/nginx" -> registry="", repository="myorg/nginx"
	if registry != "" && !strings.ContainsAny(registry, ".:") {
		repository = registry + "/" + repository
		registry = ""
	}

	// Still no registry. Use docker.io and add "library/" for single-segment names.
	if registry == "" {
		registry = dockerHubRegistry
		repository = dockerHubLibraryOrRepo(repository)
	}

	return registry, repository
}

// dockerHubLibraryOrRepo normalizes Docker Hub names.
// "nginx" -> "library/nginx"; "user/image" stays as is.
func dockerHubLibraryOrRepo(repo string) string {
	if strings.Contains(repo, "/") {
		return repo
	}
	return dockerHubLibraryNamespace + "/" + repo
}
