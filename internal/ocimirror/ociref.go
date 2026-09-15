// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

import (
	"bufio"
	"bytes"
	"maps"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// containerFields are the fields kustomize's imagetag.LegacyFilter, used by the post-renderer, rewrites images in.
var containerFields = []string{"containers", "initContainers"}

const crdKind = "CustomResourceDefinition"

// ExtractUniqueOCIRefs extracts and deduplicates the images of containerFields from YAML manifests, skipping CRDs like LegacyFilter does.
func ExtractUniqueOCIRefs(manifests string) []string {
	seen := make(map[string]struct{})

	reader := kyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifests)))
	for {
		doc, err := reader.Read()
		if err != nil {
			break
		}
		if !hasContainerField(doc) {
			continue
		}
		var obj map[string]any
		if err := kyaml.Unmarshal(doc, &obj); err != nil || obj["kind"] == crdKind {
			continue
		}
		collectContainerImages(obj, seen)
	}

	return slices.Sorted(maps.Keys(seen))
}

// hasContainerField is a cheap check to skip parsing documents without containers.
func hasContainerField(doc []byte) bool {
	return slices.ContainsFunc(containerFields, func(field string) bool {
		return bytes.Contains(doc, []byte(field))
	})
}

func collectContainerImages(node any, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		for key, value := range v {
			if slices.Contains(containerFields, key) {
				collectImages(value, seen)
			}
			collectContainerImages(value, seen)
		}
	case []any:
		for _, item := range v {
			collectContainerImages(item, seen)
		}
	}
}

func collectImages(containers any, seen map[string]struct{}) {
	items, _ := containers.([]any)
	for _, item := range items {
		container, _ := item.(map[string]any)
		if image, _ := container["image"].(string); image != "" {
			seen[image] = struct{}{}
		}
	}
}

// SplitOCIRef breaks an OCI reference into registry, repository, and tag/digest.
func SplitOCIRef(imageRef string) (registry, repository, tagOrDigest string) {
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

	// ref.Identifier() doesn't include the separator so we prepend ":" or "@".
	switch r := ref.(type) {
	case name.Tag:
		tagOrDigest = ":" + r.TagStr()
	case name.Digest:
		tagOrDigest = "@" + r.DigestStr()
	}

	return registry, repository, tagOrDigest
}
