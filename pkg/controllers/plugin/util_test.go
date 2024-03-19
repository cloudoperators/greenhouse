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

package pluginconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("validate utility functions", Ordered, func() {
	It("should get the ports from an unstructured service object", func() {
		var portNumber int32 = 80
		// Mock an unstructured object with ports
		unstructuredObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name":      "example-service",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"type": "ClusterIP",
					"ports": []interface{}{map[string]interface{}{
						"port":     portNumber,
						"protocol": "TCP",
					}},
				},
			},
		}
		port, err := getPortForExposedService(unstructuredObj)
		Ω(err).
			ShouldNot(HaveOccurred(), "there should be no error getting ports from an unstructured service object")
		Ω(port).
			ShouldNot(BeNil(), "the port should not be nil")
		Ω(port.Port).
			Should(Equal(portNumber), "the port should be 80")
	})
})
