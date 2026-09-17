// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test/fixture"
)

var _ = Describe("OCI Reference Extraction", func() {
	Describe("ExtractUniqueOCIRefs", func() {
		It("should extract images from workloads", func() {
			images := ExtractUniqueOCIRefs(fixture.PodManifest("ghcr.io/cloudoperators/greenhouse:main", "docker.io/library/nginx:latest"))
			Expect(images).To(ConsistOf(
				"docker.io/library/nginx:latest",
				"ghcr.io/cloudoperators/greenhouse:main",
			))
		})

		It("should deduplicate images", func() {
			images := ExtractUniqueOCIRefs(fixture.PodManifest("ghcr.io/cloudoperators/greenhouse:main", "ghcr.io/cloudoperators/greenhouse:main"))
			Expect(images).To(Equal([]string{"ghcr.io/cloudoperators/greenhouse:main"}))
		})

		It("should return empty for manifests without images", func() {
			manifests := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`
			images := ExtractUniqueOCIRefs(manifests)
			Expect(images).To(BeEmpty())
		})

		It("should handle quoted images", func() {
			manifests := `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: greenhouse
    image: "ghcr.io/cloudoperators/greenhouse:main"
  - name: nginx
    image: 'docker.io/library/nginx:latest'
`
			images := ExtractUniqueOCIRefs(manifests)
			Expect(images).To(HaveLen(2))
		})

		It("should ignore image fields of custom resources", func() {
			manifests := `
---
# Source: gpu-operator/templates/clusterpolicy.yaml
apiVersion: nvidia.com/v1
kind: ClusterPolicy
metadata:
  name: cluster-policy
spec:
  toolkit:
    repository: nvcr.io/nvidia/k8s
    image: container-toolkit
    version: v1.20.0
---
# Source: monitoring/templates/prometheus.yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
spec:
  image: quay.io/prometheus/prometheus:v3.0.0
---
# Source: gpu-operator/templates/operator.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gpu-operator
spec:
  template:
    spec:
      containers:
      - name: gpu-operator
        image: nvcr.io/nvidia/gpu-operator:v26.7.0
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(Equal([]string{"nvcr.io/nvidia/gpu-operator:v26.7.0"}))
		})

		It("should ignore custom resources that embed a pod spec", func() {
			manifests := `
apiVersion: opensearch.opster.io/v1
kind: OpenSearchCluster
metadata:
  name: logs
spec:
  nodePools:
  - component: masters
    podTemplate:
      spec:
        containers:
        - name: sidecar
          image: ghcr.io/cloudoperators/greenhouse:main
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(BeEmpty())
		})

		It("should extract images from initContainers of nested pod templates", func() {
			manifests := `
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  jobTemplate:
    spec:
      template:
        spec:
          initContainers:
          - name: init
            image: docker.io/library/busybox:1.36
          containers:
          - name: cleanup
            image: ghcr.io/cloudoperators/greenhouse:main
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(ConsistOf(
				"docker.io/library/busybox:1.36",
				"ghcr.io/cloudoperators/greenhouse:main",
			))
		})

		It("should ignore containers without an image", func() {
			manifests := `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  initContainers:
  - name: init
    image: ""
  containers:
  - name: main
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(BeEmpty())
		})
	})

	Describe("SplitOCIRef", func() {
		It("should split fully qualified image ref", func() {
			reg, repo, tagOrDigest := SplitOCIRef("ghcr.io/cloudoperators/greenhouse:main")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal(":main"))
		})

		It("should default to docker.io for unqualified images", func() {
			reg, repo, tagOrDigest := SplitOCIRef("nginx:latest")
			Expect(reg).To(Equal("docker.io"))
			Expect(repo).To(Equal("library/nginx"))
			Expect(tagOrDigest).To(Equal(":latest"))
		})

		It("should handle org/image format as docker.io", func() {
			reg, repo, tagOrDigest := SplitOCIRef("myorg/myapp:v1.0")
			Expect(reg).To(Equal("docker.io"))
			Expect(repo).To(Equal("myorg/myapp"))
			Expect(tagOrDigest).To(Equal(":v1.0"))
		})

		It("should handle digest references", func() {
			reg, repo, tagOrDigest := SplitOCIRef("ghcr.io/cloudoperators/greenhouse@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal("@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
		})

		It("should handle nested paths", func() {
			reg, repo, tagOrDigest := SplitOCIRef("ghcr.io/org/team/project/app:v1.0")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("org/team/project/app"))
			Expect(tagOrDigest).To(Equal(":v1.0"))
		})

		It("should default to :latest when tag is absent", func() {
			reg, repo, tagOrDigest := SplitOCIRef("ghcr.io/cloudoperators/greenhouse")
			Expect(reg).To(Equal("ghcr.io"))
			Expect(repo).To(Equal("cloudoperators/greenhouse"))
			Expect(tagOrDigest).To(Equal(":latest"))
		})
	})

})
