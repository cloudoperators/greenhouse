// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/ocimirror"
)

var _ = Describe("OCI Reference Extraction", func() {
	Describe("ExtractUniqueOCIRefs", func() {
		It("should extract images from manifests", func() {
			manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
			images := ocimirror.ExtractUniqueOCIRefs(manifests)
			Expect(images).To(HaveLen(2))
			Expect(images).To(ConsistOf(
				"docker.io/library/nginx:latest",
				"ghcr.io/cloudoperators/greenhouse:main",
			))
		})

		It("should deduplicate images", func() {
			manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: ghcr.io/cloudoperators/greenhouse:main
`
			images := ocimirror.ExtractUniqueOCIRefs(manifests)
			Expect(images).To(HaveLen(1))
			Expect(images[0]).To(Equal("ghcr.io/cloudoperators/greenhouse:main"))
		})

		It("should return empty for manifests without images", func() {
			manifests := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`
			images := ocimirror.ExtractUniqueOCIRefs(manifests)
			Expect(images).To(BeEmpty())
		})

		It("should handle quoted images", func() {
			manifests := `
containers:
- image: "ghcr.io/cloudoperators/greenhouse:main"
- image: 'docker.io/library/nginx:latest'
`
			images := ocimirror.ExtractUniqueOCIRefs(manifests)
			Expect(images).To(HaveLen(2))
		})
	})

	Describe("SplitOCIRef", func() {
		It("should split fully qualified image ref", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("ghcr.io/cloudoperators/greenhouse:main")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal(":main"))
		})

		It("should default to docker.io for unqualified images", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("nginx:latest")
			Expect(reg).To(Equal("docker.io"))
			Expect(repo).To(Equal("library/nginx"))
			Expect(tagOrDigest).To(Equal(":latest"))
		})

		It("should handle org/image format as docker.io", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("myorg/myapp:v1.0")
			Expect(reg).To(Equal("docker.io"))
			Expect(repo).To(Equal("myorg/myapp"))
			Expect(tagOrDigest).To(Equal(":v1.0"))
		})

		It("should handle digest references", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("ghcr.io/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal("@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
		})

		It("should handle nested paths", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("ghcr.io/org/team/project/app:v1.0")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("org/team/project/app"))
			Expect(tagOrDigest).To(Equal(":v1.0"))
		})

		It("should default to :latest when tag is absent", func() {
			reg, repo, tagOrDigest := ocimirror.SplitOCIRef("ghcr.io/cloudoperators/greenhouse")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal(":latest"))
		})
	})

})
