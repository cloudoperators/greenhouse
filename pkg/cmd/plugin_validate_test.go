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

package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate Plugin against PluginConfig ", func() {
	plugin := &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "greenhouse",
			Name:      "testPlugin",
		},
		Spec: greenhousev1alpha1.PluginSpec{
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

	When("pluginConfig is missing required OptionValues", func() {
		It("should raise an validation error", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin:       "testPlugin",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
			}
			err := validateOptions(plugin, pluginConfig)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})

	When("pluginConfig has OptionValues for all required Options", func() {
		It("should not return an error", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "testPlugin",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "stringRequired",
							Value: test.MustReturnJSONFor("required"),
						},
					},
				},
			}
			err := validateOptions(plugin, pluginConfig)
			Expect(err).NotTo(HaveOccurred(), "expected no error, got ", err)
		})
	})

	When("pluginConfig has OptionValues with wrong types", func() {
		It("should raise an validation error", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "testPlugin",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "stringRequired",
							Value: test.MustReturnJSONFor(true),
						},
					},
				},
			}
			err := validateOptions(plugin, pluginConfig)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})

	When("pluginConfig has OptionValues with type secret", func() {
		It("should raise an validation error if there is no secret reference", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "testPlugin",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "secret",
							Value: test.MustReturnJSONFor(true),
						},
					},
				},
			}
			err := validateOptions(plugin, pluginConfig)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
		It("should reference a secret", func() {
			pluginConfig := &greenhousev1alpha1.PluginConfig{
				Spec: greenhousev1alpha1.PluginConfigSpec{
					Plugin: "testPlugin",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name: "secret",
							ValueFrom: &greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "secret",
									Key:  "key",
								},
							},
						},
					},
				},
			}
			err := validateOptions(plugin, pluginConfig)
			Expect(err).To(HaveOccurred(), "expected an error, got nil")
		})
	})
})
