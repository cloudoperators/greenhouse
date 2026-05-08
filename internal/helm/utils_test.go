// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Utils", func() {
	Describe("MergePluginAndPluginOptionValueSlice", func() {
		Context("when a PluginOption is overridden", func() {
			It("should override the PluginOption's default value", func() {
				pluginOptions := []greenhousev1alpha1.PluginOption{
					{
						Name:    "option1",
						Default: test.MustReturnJSONFor("default-value"),
						Type:    greenhousev1alpha1.PluginOptionTypeString,
					},
				}

				pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "option1",
						Value: test.MustReturnJSONFor("override-value"),
					},
				}

				result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

				Expect(result).To(HaveLen(1))
				Expect(result[0].Name).To(Equal("option1"))
				Expect(string(result[0].Value.Raw)).To(Equal(`"override-value"`))
			})
		})
	})
	Context("when the list of PluginOptions is empty", func() {
		It("should return the PluginOptionValues unchanged", func() {
			var emptyPluginOptions []greenhousev1alpha1.PluginOption

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "option1",
					Value: test.MustReturnJSONFor("value1"),
				},
				{
					Name:  "option2",
					Value: test.MustReturnJSONFor("value2"),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(emptyPluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(2))
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"value1"`))
			Expect(result[1].Name).To(Equal("option2"))
			Expect(string(result[1].Value.Raw)).To(Equal(`"value2"`))
		})

		It("should handle nil PluginOptions", func() {
			var nilPluginOptions []greenhousev1alpha1.PluginOption = nil

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "option1",
					Value: test.MustReturnJSONFor("value1"),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(nilPluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"value1"`))
		})
	})

	Context("with complex data types", func() {
		It("should override PluginOption with a map type", func() {
			defaultMap := map[string]string{"key1": "default1", "key2": "default2"}
			overrideMap := map[string]string{"key1": "override1", "key3": "override3"}

			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "mapOption",
					Default: test.MustReturnJSONFor(defaultMap),
					Type:    greenhousev1alpha1.PluginOptionTypeMap,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "mapOption",
					Value: test.MustReturnJSONFor(overrideMap),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("mapOption"))

			var resultMap map[string]string
			err := json.Unmarshal(result[0].Value.Raw, &resultMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultMap).To(HaveLen(2))
			Expect(resultMap["key1"]).To(Equal("override1"))
			Expect(resultMap["key3"]).To(Equal("override3"))
		})

		It("should override PluginOption with an empty map", func() {
			defaultMap := map[string]string{"key1": "default1", "key2": "default2"}
			emptyMap := map[string]string{}

			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "mapOption",
					Default: test.MustReturnJSONFor(defaultMap),
					Type:    greenhousev1alpha1.PluginOptionTypeMap,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "mapOption",
					Value: test.MustReturnJSONFor(emptyMap),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("mapOption"))

			var resultMap map[string]string
			err := json.Unmarshal(result[0].Value.Raw, &resultMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultMap).To(BeEmpty())
		})

		It("should override PluginOption with a slice type", func() {
			defaultSlice := []string{"default1", "default2"}
			overrideSlice := []string{"override1", "override2", "override3"}

			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "sliceOption",
					Default: test.MustReturnJSONFor(defaultSlice),
					Type:    greenhousev1alpha1.PluginOptionTypeList,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "sliceOption",
					Value: test.MustReturnJSONFor(overrideSlice),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("sliceOption"))

			var resultSlice []string
			err := json.Unmarshal(result[0].Value.Raw, &resultSlice)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultSlice).To(HaveLen(3))
			Expect(resultSlice).To(Equal(overrideSlice))
		})

		It("should override PluginOption with an empty slice", func() {
			defaultSlice := []string{"default1", "default2"}
			emptySlice := []string{}

			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "sliceOption",
					Default: test.MustReturnJSONFor(defaultSlice),
					Type:    greenhousev1alpha1.PluginOptionTypeList,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "sliceOption",
					Value: test.MustReturnJSONFor(emptySlice),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("sliceOption"))

			var resultSlice []string
			err := json.Unmarshal(result[0].Value.Raw, &resultSlice)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultSlice).To(BeEmpty())
		})

		It("should override PluginOption with integer type", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "intOption",
					Default: test.MustReturnJSONFor(42),
					Type:    greenhousev1alpha1.PluginOptionTypeInt,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "intOption",
					Value: test.MustReturnJSONFor(99),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("intOption"))

			var resultInt int
			err := json.Unmarshal(result[0].Value.Raw, &resultInt)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultInt).To(Equal(99))
		})

		It("should override PluginOption with boolean type", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "boolOption",
					Default: test.MustReturnJSONFor(false),
					Type:    greenhousev1alpha1.PluginOptionTypeBool,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "boolOption",
					Value: test.MustReturnJSONFor(true),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("boolOption"))

			var resultBool bool
			err := json.Unmarshal(result[0].Value.Raw, &resultBool)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultBool).To(BeTrue())
		})
	})

	Context("with edge cases", func() {
		It("should handle both empty inputs", func() {
			var emptyPluginOptions []greenhousev1alpha1.PluginOption
			var emptyOptionValues []greenhousev1alpha1.PluginOptionValue

			result := helm.MergePluginAndPluginOptionValueSlice(emptyPluginOptions, emptyOptionValues)

			Expect(result).To(BeEmpty())
		})

		It("should handle only PluginOptions with defaults", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "option1",
					Default: test.MustReturnJSONFor("default1"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
				{
					Name:    "option2",
					Default: test.MustReturnJSONFor("default2"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
			}

			var emptyOptionValues []greenhousev1alpha1.PluginOptionValue

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, emptyOptionValues)

			Expect(result).To(HaveLen(2))
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"default1"`))
			Expect(result[1].Name).To(Equal("option2"))
			Expect(string(result[1].Value.Raw)).To(Equal(`"default2"`))
		})

		It("should handle PluginOptions without Default values", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name: "option1",
					Type: greenhousev1alpha1.PluginOptionTypeString,
					// No Default value
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "option1",
					Value: test.MustReturnJSONFor("value1"),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"value1"`))
		})

		It("should handle multiple overrides", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "option1",
					Default: test.MustReturnJSONFor("default1"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
				{
					Name:    "option2",
					Default: test.MustReturnJSONFor("default2"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
				{
					Name:    "option3",
					Default: test.MustReturnJSONFor("default3"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "option1",
					Value: test.MustReturnJSONFor("override1"),
				},
				{
					Name:  "option3",
					Value: test.MustReturnJSONFor("override3"),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(3))
			// Result should be sorted by Name
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"override1"`))
			Expect(result[1].Name).To(Equal("option2"))
			Expect(string(result[1].Value.Raw)).To(Equal(`"default2"`))
			Expect(result[2].Name).To(Equal("option3"))
			Expect(string(result[2].Value.Raw)).To(Equal(`"override3"`))
		})

		It("should handle new values not in PluginOptions", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "option1",
					Default: test.MustReturnJSONFor("default1"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "option1",
					Value: test.MustReturnJSONFor("override1"),
				},
				{
					Name:  "option2",
					Value: test.MustReturnJSONFor("value2"), // New option not in pluginOptions
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(2))
			// Result should be sorted by Name
			Expect(result[0].Name).To(Equal("option1"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"override1"`))
			Expect(result[1].Name).To(Equal("option2"))
			Expect(string(result[1].Value.Raw)).To(Equal(`"value2"`))
		})

		It("should ensure output is sorted by Name", func() {
			pluginOptions := []greenhousev1alpha1.PluginOption{
				{
					Name:    "c-option",
					Default: test.MustReturnJSONFor("c-default"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
				{
					Name:    "a-option",
					Default: test.MustReturnJSONFor("a-default"),
					Type:    greenhousev1alpha1.PluginOptionTypeString,
				},
			}

			pluginOptionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "d-option", // New option not in pluginOptions
					Value: test.MustReturnJSONFor("d-value"),
				},
				{
					Name:  "b-option", // New option not in pluginOptions
					Value: test.MustReturnJSONFor("b-value"),
				},
			}

			result := helm.MergePluginAndPluginOptionValueSlice(pluginOptions, pluginOptionValues)

			Expect(result).To(HaveLen(4))
			// Result should be sorted by Name
			Expect(result[0].Name).To(Equal("a-option"))
			Expect(result[1].Name).To(Equal("b-option"))
			Expect(result[2].Name).To(Equal("c-option"))
			Expect(result[3].Name).To(Equal("d-option"))
		})
	})
})
