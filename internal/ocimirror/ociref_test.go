// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package ocimirror

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCI Reference Extraction", func() {
	Describe("ExtractUniqueOCIRefs", func() {
		It("should extract images from manifests", func() {
			manifests := `
containers:
- image: ghcr.io/cloudoperators/greenhouse:main
- image: docker.io/library/nginx:latest
`
			images := ExtractUniqueOCIRefs(manifests)
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
			images := ExtractUniqueOCIRefs(manifests)
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
			images := ExtractUniqueOCIRefs(manifests)
			Expect(images).To(BeEmpty())
		})

		It("should handle quoted images", func() {
			manifests := `
containers:
- image: "ghcr.io/cloudoperators/greenhouse:main"
- image: 'docker.io/library/nginx:latest'
`
			images := ExtractUniqueOCIRefs(manifests)
			Expect(images).To(HaveLen(2))
		})

		It("should ignore `image:` used as a CRD field name with nested children", func() {
			// Regression: OpenSearch index templates declare `image` as a field, not a container ref.
			manifests := `
mappings:
  properties:
    container:
      properties:
        image:
          properties:
            name:
              type: keyword
            tag:
              type: keyword
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(BeEmpty())
		})

		It("should ignore `image` fields outside of containers", func() {
			manifests := `
---
# Source: gpu-operator/templates/clusterpolicy.yaml
apiVersion: nvidia.com/v1
kind: ClusterPolicy
metadata:
  name: cluster-policy
spec:
  toolkit:
    repository: keppel.global.cloud.sap/ccloud-nvcr-io-mirror/nvidia/k8s
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

		It("should extract images from documents with only initContainers", func() {
			manifests := `
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
spec:
  initContainers:
  - name: init-config-reloader
    image: quay.io/prometheus-operator/prometheus-config-reloader:v0.80.0
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(Equal([]string{"quay.io/prometheus-operator/prometheus-config-reloader:v0.80.0"}))
		})

		It("should ignore containers without an image", func() {
			manifests := `
apiVersion: monitoring.coreos.com/v1
kind: ThanosRuler
metadata:
  name: thanos-ruler
spec:
  containers:
  - name: thanos-ruler
    volumeMounts:
    - mountPath: /etc/thanos/secrets
      name: alertmanager-sso-cert
  initContainers:
  - name: init
    image: ""
`
			Expect(ExtractUniqueOCIRefs(manifests)).To(BeEmpty())
		})

		It("should ignore CustomResourceDefinitions", func() {
			manifests := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: runners.example.com
spec:
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            default:
              containers:
              - name: runner
                image: docker.io/library/busybox:1.36
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
