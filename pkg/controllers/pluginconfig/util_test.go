// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
