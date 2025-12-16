// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate PluginDefinition against Plugin ", func() {
	pluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "greenhouse",
			Name:      "testPlugin",
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:    "stringWithDefault",
					Type:    greenhousev1alpha1.PluginOptionTypeString,
					Default: test.MustReturnJSONFor("defaulted"),
				},
				{
					Name:     "stringRequired",
					Type:     greenhousev1alpha1.PluginOptionTypeString,
					Required: true,
				},
				{
					Name: "secretValue",
					Type: greenhousev1alpha1.PluginOptionTypeSecret,
				},
			},
		},
	}

	When("plugin is missing required OptionValues", func() {
		It("should raise an validation error", func() {
			plugin := &greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "testPlugin",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
			}
			err := validateOptions(pluginDefinition, plugin)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})

	When("plugin has OptionValues for all required Options", func() {
		It("should not return an error", func() {
			plugin := &greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "testPlugin",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "stringRequired",
							Value: test.MustReturnJSONFor("required"),
						},
					},
				},
			}
			err := validateOptions(pluginDefinition, plugin)
			Expect(err).NotTo(HaveOccurred(), "expected no error, got ", err)
		})
	})

	When("plugin has OptionValues with wrong types", func() {
		It("should raise an validation error", func() {
			plugin := &greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "testPlugin",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "stringRequired",
							Value: test.MustReturnJSONFor(true),
						},
					},
				},
			}
			err := validateOptions(pluginDefinition, plugin)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})

	When("plugin has OptionValues with type secret", func() {
		It("should raise an validation error if there is no secret reference", func() {
			plugin := &greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "testPlugin",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "secret",
							Value: test.MustReturnJSONFor(true),
						},
					},
				},
			}
			err := validateOptions(pluginDefinition, plugin)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
		It("should reference a secret", func() {
			plugin := &greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "testPlugin",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name: "secret",
							ValueFrom: &greenhousev1alpha1.PluginValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "secret",
									Key:  "key",
								},
							},
						},
					},
				},
			}
			err := validateOptions(pluginDefinition, plugin)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})
})
