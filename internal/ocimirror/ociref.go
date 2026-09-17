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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// ExtractUniqueOCIRefs extracts and deduplicates the container images of the standard workloads in the given manifests.
// Images of custom resources are ignored, even when the resource embeds a pod spec.
func ExtractUniqueOCIRefs(manifests string) []string {
	seen := make(map[string]struct{})

	decoder := clientgoscheme.Codecs.UniversalDeserializer()
	reader := kyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifests)))
	for {
		doc, err := reader.Read()
		if err != nil {
			break
		}
		// Fast path, decoding is the expensive part and every pod spec has containers.
		if !bytes.Contains(doc, []byte("containers")) {
			continue
		}
		obj, _, err := decoder.Decode(doc, nil, nil)
		if err != nil {
			continue
		}
		spec := podSpecOf(obj)
		if spec == nil {
			continue
		}
		for _, container := range slices.Concat(spec.InitContainers, spec.Containers) {
			if container.Image != "" {
				seen[container.Image] = struct{}{}
			}
		}
	}

	return slices.Sorted(maps.Keys(seen))
}

// podSpecOf returns the pod spec of the standard Kubernetes workloads and nil for anything else.
func podSpecOf(obj runtime.Object) *corev1.PodSpec {
	switch o := obj.(type) {
	case *corev1.Pod:
		return &o.Spec
	case *appsv1.Deployment:
		return &o.Spec.Template.Spec
	case *appsv1.DaemonSet:
		return &o.Spec.Template.Spec
	case *appsv1.StatefulSet:
		return &o.Spec.Template.Spec
	case *appsv1.ReplicaSet:
		return &o.Spec.Template.Spec
	case *corev1.ReplicationController:
		if o.Spec.Template != nil {
			return &o.Spec.Template.Spec
		}
	case *batchv1.Job:
		return &o.Spec.Template.Spec
	case *batchv1.CronJob:
		return &o.Spec.JobTemplate.Spec.Template.Spec
	}
	return nil
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
