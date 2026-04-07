// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/ocimirror"
)

func newTestMirror(config *ocimirror.RegistryMirrorConfig) *ocimirror.ImageMirror {
	return ocimirror.NewImageMirrorForTest(config, authn.Anonymous, func(ref string, opts ...crane.Option) ([]byte, error) {
		return []byte("{}"), nil
	})
}

var _ = Describe("createRegistryMirrorPostRenderer", func() {
	var mirror *ocimirror.ImageMirror

	BeforeEach(func() {
		mirror = newTestMirror(&ocimirror.RegistryMirrorConfig{
			RegistryMirrors: map[string]ocimirror.RegistryMirror{
				"ghcr.io": {
					BaseDomain: "mirror.example.com",
					SubPath:    "ghcr-mirror",
				},
				"docker.io": {
					BaseDomain: "mirror.example.com",
					SubPath:    "dockerhub-mirror",
				},
			},
		})
	})

	It("should return nil when mirror is nil", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(nil, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should return nil when no images in manifests", func() {
		manifest := `
		apiVersion: v1
		kind: ConfigMap
		metadata:
		name: test
		data:
		key: value
		`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should return nil when no matching mirrors", func() {
		manifest := `image: registry.k8s.io/pause:3.9`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should create transformation preserving full image path", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(1))
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/cloudoperators/greenhouse"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/cloudoperators/greenhouse"))
	})

	It("should handle multiple images with different registries", func() {
		manifest := `
		containers:
		- image: ghcr.io/cloudoperators/greenhouse:main
		- image: docker.io/library/nginx:latest
		- image: registry.k8s.io/pause:3.9
		`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(2))

		names := []string{postRenderer.Kustomize.Images[0].Name, postRenderer.Kustomize.Images[1].Name}
		Expect(names).To(ConsistOf(
			"ghcr.io/cloudoperators/greenhouse",
			"docker.io/library/nginx",
		))
	})

	It("should deduplicate identical images", func() {
		manifest := `
		containers:
		- image: ghcr.io/cloudoperators/greenhouse:main
		- image: ghcr.io/cloudoperators/greenhouse:main
		- image: ghcr.io/cloudoperators/greenhouse:main
		`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(1))
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/cloudoperators/greenhouse"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/cloudoperators/greenhouse"))
	})

	It("should preserve nested repository paths", func() {
		manifest := `image: ghcr.io/org/team/project/app:v1.0`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/org/team/project/app"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/org/team/project/app"))
	})

	It("should handle images with digest", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/cloudoperators/greenhouse"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/cloudoperators/greenhouse"))
	})

	It("should handle images with quotes", func() {
		manifest := `
		containers:
		- image: "ghcr.io/cloudoperators/greenhouse:main"
		- image: 'docker.io/library/nginx:latest'
		`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())

		names := []string{postRenderer.Kustomize.Images[0].Name, postRenderer.Kustomize.Images[1].Name}
		Expect(names).To(ConsistOf(
			"ghcr.io/cloudoperators/greenhouse",
			"docker.io/library/nginx",
		))
		newNames := []string{postRenderer.Kustomize.Images[0].NewName, postRenderer.Kustomize.Images[1].NewName}
		Expect(newNames).To(ConsistOf(
			"mirror.example.com/ghcr-mirror/cloudoperators/greenhouse",
			"mirror.example.com/dockerhub-mirror/library/nginx",
		))
	})

	It("should handle Docker Hub default registry for images without registry prefix", func() {
		manifest := `
		containers:
		- image: nginx:latest
		- image: myorg/myapp:v1.0
		`
		postRenderer := createRegistryMirrorPostRenderer(mirror, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(2))

		names := []string{postRenderer.Kustomize.Images[0].Name, postRenderer.Kustomize.Images[1].Name}
		Expect(names).To(ConsistOf(
			"docker.io/library/nginx",
			"docker.io/myorg/myapp",
		))

		newNames := []string{postRenderer.Kustomize.Images[0].NewName, postRenderer.Kustomize.Images[1].NewName}
		Expect(newNames).To(ConsistOf(
			"mirror.example.com/dockerhub-mirror/library/nginx",
			"mirror.example.com/dockerhub-mirror/myorg/myapp",
		))
	})

	It("should include image refs from helm hook manifests", func() {
		mainManifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		hookManifest := `image: docker.io/library/busybox:1.36`
		postRenderer := createRegistryMirrorPostRenderer(mirror, mainManifest, hookManifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(2))

		names := []string{postRenderer.Kustomize.Images[0].Name, postRenderer.Kustomize.Images[1].Name}
		Expect(names).To(ConsistOf(
			"ghcr.io/cloudoperators/greenhouse",
			"docker.io/library/busybox",
		))
	})

	It("should deduplicate image refs across manifest and hooks", func() {
		mainManifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		hookManifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(mirror, mainManifest, hookManifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(1))
	})

	It("should pick up image refs even when only present in a hook", func() {
		hookManifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(mirror, "", hookManifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(1))
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/cloudoperators/greenhouse"))
	})
})

var _ = Describe("ensureImageReplication", func() {
	var (
		plugin *greenhousev1alpha1.Plugin
		mirror *ocimirror.ImageMirror
	)

	BeforeEach(func() {
		plugin = &greenhousev1alpha1.Plugin{}
		plugin.Name = "test-plugin"
		plugin.Namespace = "test-ns"
	})

	It("should replicate images and update status", func() {
		fetchedRefs := make([]string, 0)
		mirror = ocimirror.NewImageMirrorForTest(&ocimirror.RegistryMirrorConfig{
			RegistryMirrors: map[string]ocimirror.RegistryMirror{
				"ghcr.io": {BaseDomain: "mirror.example.com", SubPath: "ghcr-mirror"},
			},
		}, authn.Anonymous, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchedRefs = append(fetchedRefs, ref)
			return []byte("{}"), nil
		})

		manifests := "image: ghcr.io/cloudoperators/greenhouse:main"
		err := ensureImageReplication(context.Background(), mirror, plugin, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(plugin.Status.ImageReplication).To(ContainElement("ghcr.io/cloudoperators/greenhouse:main"))
		Expect(fetchedRefs).To(HaveLen(1))
	})

	It("should skip already replicated images", func() {
		fetchCount := 0
		mirror = ocimirror.NewImageMirrorForTest(&ocimirror.RegistryMirrorConfig{
			RegistryMirrors: map[string]ocimirror.RegistryMirror{
				"ghcr.io": {BaseDomain: "mirror.example.com", SubPath: "ghcr-mirror"},
			},
		}, authn.Anonymous, func(ref string, opts ...crane.Option) ([]byte, error) {
			fetchCount++
			return []byte("{}"), nil
		})

		plugin.Status.ImageReplication = []string{"ghcr.io/cloudoperators/greenhouse:main"}
		manifests := "image: ghcr.io/cloudoperators/greenhouse:main"
		err := ensureImageReplication(context.Background(), mirror, plugin, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetchCount).To(Equal(0))
	})

	It("should return error and set condition on replication failure", func() {
		mirror = ocimirror.NewImageMirrorForTest(&ocimirror.RegistryMirrorConfig{
			RegistryMirrors: map[string]ocimirror.RegistryMirror{
				"ghcr.io": {BaseDomain: "mirror.example.com", SubPath: "ghcr-mirror"},
			},
		}, authn.Anonymous, func(ref string, opts ...crane.Option) ([]byte, error) {
			return nil, errors.New("connection refused")
		})

		plugin.Status.ImageReplication = []string{"ghcr.io/cloudoperators/previous:v1"}

		manifests := "image: ghcr.io/cloudoperators/greenhouse:main"
		err := ensureImageReplication(context.Background(), mirror, plugin, manifests)
		Expect(err).To(HaveOccurred())

		Expect(plugin.Status.ImageReplication).To(Equal([]string{"ghcr.io/cloudoperators/previous:v1"}))

		cond := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReleaseCreatedCondition)
		Expect(cond).NotTo(BeNil())
		Expect(cond.IsFalse()).To(BeTrue())
		Expect(string(cond.Reason)).To(Equal(string(greenhousev1alpha1.ImageReplicationFailedReason)))
	})

	It("should set NotConfigured when no images match any mirror", func() {
		mirror = ocimirror.NewImageMirrorForTest(&ocimirror.RegistryMirrorConfig{
			RegistryMirrors: map[string]ocimirror.RegistryMirror{
				"ghcr.io": {BaseDomain: "mirror.example.com", SubPath: "ghcr-mirror"},
			},
		}, authn.Anonymous, func(ref string, opts ...crane.Option) ([]byte, error) {
			return []byte("{}"), nil
		})

		manifests := "image: registry.k8s.io/pause:3.9"
		err := ensureImageReplication(context.Background(), mirror, plugin, manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(plugin.Status.ImageReplication).To(BeEmpty())
	})
})
