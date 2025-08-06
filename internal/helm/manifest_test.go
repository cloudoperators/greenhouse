// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime/schema"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("validate the manifest functions", Ordered, func() {
	It("should get the objects from a release manifest", func() {
		helmReleaseWithManifest := &release.Release{
			Manifest: `
---
# Source: some file0.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: greenhouse
---
# Source: some file1.yaml
apiVersion: v1
kind: Service
metadata:
  name: exposed-service
  namespace: greenhouse
  annotations:
    greenhouse.sap/expose: "true"
spec:
  selector:
    app: some-app
  type: ClusterIP
  ports:
  - name: http
    port: 80
---
`,
		}

		manifestObjectMap, err := helm.ObjectMapFromRelease(clientutil.NewRestClientGetterFromRestConfig(test.Cfg, "greenhouse", clientutil.WithPersistentConfig()), helmReleaseWithManifest, &helm.ManifestObjectFilter{
			APIVersion: "v1",
			Kind:       "Service",
			Annotations: map[string]string{
				greenhouseapis.AnnotationKeyExpose: "true",
			},
		})
		Ω(err).
			ShouldNot(HaveOccurred(), "there should be no error getting the objects from the helm release")
		Ω(manifestObjectMap).
			ShouldNot(BeNil(), "the manifest object list should not be nil")
		Ω(manifestObjectMap).
			Should(HaveLen(1), "there should be one object in the manifest object list according to the filter")
		key := helm.ObjectKey{GVK: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}, Namespace: "greenhouse", Name: "exposed-service"}
		Ω(manifestObjectMap[key].Name).
			Should(Equal("exposed-service"), "the name of the object should be exposed-service")
		Ω(manifestObjectMap[key].Object).
			ShouldNot(BeNil(), "the object should not be nil")
	})

	It("should not throw any error for missing CRDs", func() {
		helmReleaseWithManifest := &release.Release{
			Manifest: `
---
# Source: some file0.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: greenhouse
---
# Source: some file1.yaml
apiVersion: v3
kind: ImaginaryCRD
metadata:
  name: exposed-service
  namespace: greenhouse
  labels:
    greenhouse.sap/expose: "true"
spec:
  selector:
    app: some-app
  type: ClusterIP
  ports:
  - name: http
    port: 80
---
`,
		}

		manifestObjectMap, err := helm.ObjectMapFromRelease(clientutil.NewRestClientGetterFromRestConfig(test.Cfg, "greenhouse", clientutil.WithPersistentConfig()), helmReleaseWithManifest, nil)
		Ω(err).
			ShouldNot(HaveOccurred(), "there should be no error getting the objects from the helm release")
		Ω(manifestObjectMap).
			ShouldNot(BeNil(), "the manifest object list should not be nil")
		Ω(manifestObjectMap).
			Should(HaveLen(1), "there should be one object in the manifest object list ignoring the missing CRD")
	})
})
