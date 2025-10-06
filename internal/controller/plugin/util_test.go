// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
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
						"greenhouse.sap/exposed-named-port": "https",
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

	Describe("getURLForExposedIngress", func() {
		DescribeTable("should generate correct URLs for ingress objects",
			func(spec networkingv1.IngressSpec, annotations map[string]string, expectedURL string, shouldError bool) {
				ingress := &networkingv1.Ingress{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "networking.k8s.io/v1",
						Kind:       "Ingress",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-ingress",
						Namespace:   "default",
						Annotations: annotations,
					},
					Spec: spec,
				}

				url, err := getURLForExposedIngress(ingress)

				if shouldError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(url).To(Equal(expectedURL))
				}
			},
			Entry("HTTP ingress with single host",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"http://api.example.com",
				false,
			),
			Entry("HTTPS ingress with TLS",
				networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"secure.example.com"}},
					},
					Rules: []networkingv1.IngressRule{
						{Host: "secure.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"https://secure.example.com",
				false,
			),
			Entry("HTTPS ingress with wildcard TLS",
				networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{}},
					},
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"https://api.example.com",
				false,
			),
			Entry("ingress with multiple hosts - uses first by default",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
						{Host: "admin.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"http://api.example.com",
				false,
			),
			Entry("ingress with multiple hosts - uses specified host",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
						{Host: "admin.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose:             "true",
					greenhouseapis.AnnotationKeyExposedIngressHost: "admin.example.com",
				},
				"http://admin.example.com",
				false,
			),
			Entry("HTTPS ingress with multiple hosts and TLS for specific host",
				networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"secure.example.com"}},
					},
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
						{Host: "secure.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose:             "true",
					greenhouseapis.AnnotationKeyExposedIngressHost: "secure.example.com",
				},
				"https://secure.example.com",
				false,
			),
			Entry("ingress with no rules - should error",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"",
				true,
			),
			Entry("ingress with empty host - should error",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: ""},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose: "true",
				},
				"",
				true,
			),
			Entry("ingress with specified host not found - should error",
				networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
					},
				},
				map[string]string{
					greenhouseapis.AnnotationKeyExpose:             "true",
					greenhouseapis.AnnotationKeyExposedIngressHost: "nonexistent.example.com",
				},
				"",
				true,
			),
		)
	})

	Describe("resolvePluginDependencies", func() {
		When("input is references only to plugin names", func() {
			It("should leave dependencies as they are", func() {
				input := []greenhousev1alpha1.WaitForItem{
					{
						PluginRef: greenhousev1alpha1.PluginRef{Name: "test-plugin-1"},
					},
					{
						PluginRef: greenhousev1alpha1.PluginRef{Name: "test-plugin-2"},
					},
					{
						PluginRef: greenhousev1alpha1.PluginRef{Name: "test-plugin-3"},
					},
				}
				output := resolvePluginDependencies(input, "cluster-a")
				Expect(output).To(BeComparableTo(input), "the output should not change in regards to input")
			})
		})
		When("input is mixed", func() {
			It("should transform dependencies to contain only plugin names", func() {
				input := []greenhousev1alpha1.WaitForItem{
					{
						PluginRef: greenhousev1alpha1.PluginRef{Name: "test-plugin-1"},
					},
					{
						PluginRef: greenhousev1alpha1.PluginRef{PluginPreset: "test-preset-1"},
					},
					{
						PluginRef: greenhousev1alpha1.PluginRef{PluginPreset: "test-preset-2"},
					},
				}
				output := resolvePluginDependencies(input, "cluster-a")
				Expect(output).To(
					ContainElements(
						greenhousev1alpha1.WaitForItem{
							PluginRef: greenhousev1alpha1.PluginRef{Name: "test-plugin-1", PluginPreset: ""},
						},
						greenhousev1alpha1.WaitForItem{
							PluginRef: greenhousev1alpha1.PluginRef{Name: "test-preset-1-cluster-a", PluginPreset: ""},
						},
						greenhousev1alpha1.WaitForItem{
							PluginRef: greenhousev1alpha1.PluginRef{Name: "test-preset-2-cluster-a", PluginPreset: ""},
						},
					), "the dependencies should be transformed to plugin names")
			})
		})
	})
})
