// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

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
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name":      "example-service",
					"namespace": "default",
				},
				"spec": map[string]any{
					"type": "ClusterIP",
					"ports": []any{map[string]any{
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
	It("should get named port from an unstructured service object", func() {
		var portNumber1 int32 = 80
		var portNumber2 int32 = 443
		// Mock an unstructured object with ports
		unstructuredObj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name":      "example-service",
					"namespace": "default",
					"annotations": map[string]string{
						"greenhouse.sap/expose": "https",
					},
				},
				"spec": map[string]any{
					"type": "ClusterIP",
					"ports": []any{
						map[string]any{
							"name":     "http",
							"port":     portNumber1,
							"protocol": "TCP",
						},
						map[string]any{
							"name":     "https",
							"port":     portNumber2,
							"protocol": "TCP",
						},
					},
				},
			},
		}
		port, err := getPortForExposedService(unstructuredObj)
		Ω(err).
			ShouldNot(HaveOccurred(), "there should be no error getting ports from an unstructured service object")
		Ω(port).
			ShouldNot(BeNil(), "the port should not be nil")
		Ω(port).
			ShouldNot(HaveValue(BeEquivalentTo(portNumber1)), "the port should not be 80")
		Ω(port.Port).
			Should(Equal(portNumber2), "the port should be 443")
	})
})
