// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/common"
)

var _ = Describe("createRegistryMirrorPostRenderer", func() {
	var mirrorConfig *common.RegistryMirrorConfig

	BeforeEach(func() {
		mirrorConfig = &common.RegistryMirrorConfig{
			RegistryMirrors: map[string]common.RegistryMirror{
				"ghcr.io": {
					BaseDomain: "mirror.example.com",
					SubPath:    "ghcr-mirror",
				},
				"docker.io": {
					BaseDomain: "mirror.example.com",
					SubPath:    "dockerhub-mirror",
				},
			},
		}
	})

	It("should return nil when config is nil", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(nil, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should return nil when no mirrors configured", func() {
		emptyConfig := &common.RegistryMirrorConfig{
			RegistryMirrors: map[string]common.RegistryMirror{},
		}
		manifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(emptyConfig, manifest)
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
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should return nil when no matching mirrors", func() {
		manifest := `image: registry.k8s.io/pause:3.9`
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
		Expect(postRenderer).To(BeNil())
	})

	It("should create transformation preserving full image path", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse:main`
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
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
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
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
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images).To(HaveLen(1))
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/cloudoperators/greenhouse"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/cloudoperators/greenhouse"))
	})

	It("should preserve nested repository paths", func() {
		manifest := `image: ghcr.io/org/team/project/app:v1.0`
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
		Expect(postRenderer).NotTo(BeNil())
		Expect(postRenderer.Kustomize.Images[0].Name).To(Equal("ghcr.io/org/team/project/app"))
		Expect(postRenderer.Kustomize.Images[0].NewName).To(Equal("mirror.example.com/ghcr-mirror/org/team/project/app"))
	})

	It("should handle images with digest", func() {
		manifest := `image: ghcr.io/cloudoperators/greenhouse@sha256:abc123`
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
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
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
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

	It("should skip images without registry prefix", func() {
		manifest := `image: nginx:latest`
		postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, manifest)
		Expect(postRenderer).To(BeNil())
	})
})
