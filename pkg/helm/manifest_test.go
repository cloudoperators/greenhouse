// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime/schema"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
	"github.com/cloudoperators/greenhouse/pkg/test"
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

		manifestObjectMap, err := helm.ObjectMapFromRelease(clientutil.NewRestClientGetterFromRestConfig(test.Cfg, "greenhouse", clientutil.WithPersistentConfig()), helmReleaseWithManifest, &helm.ManifestObjectFilter{
			APIVersion: "v1",
			Kind:       "Service",
			Labels: map[string]string{
				greenhouseapis.LabelKeyExposeService: "true",
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
})
