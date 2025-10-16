// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("validateCompatibility", func() {
	var opts *PluginTemplatePresetOptions

	Context("with mismatched plugin definition names", func() {
		BeforeEach(func() {
			opts = &PluginTemplatePresetOptions{
				pluginDefinition: &greenhousev1alpha1.ClusterPluginDefinition{
					TypeMeta:   metav1.TypeMeta{Kind: ClusterPluginDefinitionKind},
					ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
				},
				pluginPreset: &greenhousev1alpha1.PluginPreset{
					TypeMeta: metav1.TypeMeta{Kind: PluginPresetKind},
					Spec: greenhousev1alpha1.PluginPresetSpec{
						Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
							PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
								Name: "apache",
								Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
							},
						},
					},
				},
			}
		})

		It("should return an error", func() {
			err := opts.validateCompatibility()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("references (Cluster-)PluginDefinition 'apache' but provided file defines 'nginx'"))
		})
	})

	Context("with missing helm chart reference", func() {
		BeforeEach(func() {
			opts = &PluginTemplatePresetOptions{
				pluginDefinition: &greenhousev1alpha1.ClusterPluginDefinition{
					TypeMeta:   metav1.TypeMeta{Kind: ClusterPluginDefinitionKind},
					ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
					Spec: greenhousev1alpha1.PluginDefinitionSpec{
						HelmChart: nil,
					},
				},
				pluginPreset: &greenhousev1alpha1.PluginPreset{
					TypeMeta: metav1.TypeMeta{Kind: PluginPresetKind},
					Spec: greenhousev1alpha1.PluginPresetSpec{
						Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
							PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
								Name: "nginx",
								Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
							},
						},
					},
				},
			}
		})

		It("should return an error about missing helm chart", func() {
			err := opts.validateCompatibility()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must have a HelmChart reference"))
			Expect(err.Error()).To(ContainSubstring("nginx"))
		})
	})

	Context("with valid configuration", func() {
		BeforeEach(func() {
			opts = &PluginTemplatePresetOptions{
				pluginDefinition: &greenhousev1alpha1.ClusterPluginDefinition{
					TypeMeta:   metav1.TypeMeta{Kind: ClusterPluginDefinitionKind},
					ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
					Spec: greenhousev1alpha1.PluginDefinitionSpec{
						HelmChart: &greenhousev1alpha1.HelmChartReference{
							Name:       "nginx",
							Repository: "https://charts.bitnami.com/bitnami",
							Version:    "15.4.4",
						},
					},
				},
				pluginPreset: &greenhousev1alpha1.PluginPreset{
					TypeMeta: metav1.TypeMeta{Kind: PluginPresetKind},
					Spec: greenhousev1alpha1.PluginPresetSpec{
						Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
							PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
								Name: "nginx",
								Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
							},
						},
					},
				},
			}
		})

		It("should not return an error", func() {
			err := opts.validateCompatibility()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("prepareValues", func() {
	var (
		opts         *PluginTemplatePresetOptions
		pluginDef    *greenhousev1alpha1.ClusterPluginDefinition
		pluginPreset *greenhousev1alpha1.PluginPreset
	)

	BeforeEach(func() {
		opts = &PluginTemplatePresetOptions{
			clusterName: "test-cluster",
		}

		pluginDef = &greenhousev1alpha1.ClusterPluginDefinition{
			TypeMeta:   metav1.TypeMeta{Kind: ClusterPluginDefinitionKind},
			ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:    "replicas",
						Default: &apiextensionsv1.JSON{Raw: []byte("2")},
					},
				},
			},
		}

		pluginPreset = &greenhousev1alpha1.PluginPreset{
			TypeMeta: metav1.TypeMeta{Kind: PluginPresetKind},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-preset",
				Namespace: "test-org",
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
					PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
						Name: "nginx",
						Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
					},
				},
			},
		}

		opts.pluginDefinition = pluginDef
		opts.pluginPreset = pluginPreset
	})

	Context("with PluginDefinition defaults only", func() {
		It("should include default values", func() {
			err := opts.prepareValues()
			Expect(err).NotTo(HaveOccurred())

			var replicasValue *greenhousev1alpha1.PluginOptionValue
			for _, v := range opts.values {
				if v.Name == "replicas" {
					replicasValue = &v
					break
				}
			}

			Expect(replicasValue).NotTo(BeNil())
			Expect(string(replicasValue.Value.Raw)).To(Equal("2"))
		})

		It("should include greenhouse values", func() {
			err := opts.prepareValues()
			Expect(err).NotTo(HaveOccurred())

			var clusterNameValue *greenhousev1alpha1.PluginOptionValue
			for _, v := range opts.values {
				if v.Name == "global.greenhouse.clusterName" {
					clusterNameValue = &v
					break
				}
			}

			Expect(clusterNameValue).NotTo(BeNil())
			Expect(string(clusterNameValue.Value.Raw)).To(Equal(`"test-cluster"`))
		})
	})

	Context("with PluginPreset overrides", func() {
		BeforeEach(func() {
			pluginPreset.Spec.Plugin.OptionValues = []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "replicas",
					Value: &apiextensionsv1.JSON{Raw: []byte("3")},
				},
			}
		})

		It("should override default values", func() {
			err := opts.prepareValues()
			Expect(err).NotTo(HaveOccurred())

			var replicasValue *greenhousev1alpha1.PluginOptionValue
			for _, v := range opts.values {
				if v.Name == "replicas" {
					replicasValue = &v
					break
				}
			}

			Expect(replicasValue).NotTo(BeNil())
			Expect(string(replicasValue.Value.Raw)).To(Equal("3"))
		})
	})

	Context("with cluster-specific overrides", func() {
		BeforeEach(func() {
			pluginPreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
				{
					ClusterName: "test-cluster",
					Overrides: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "replicas",
							Value: &apiextensionsv1.JSON{Raw: []byte("5")},
						},
					},
				},
			}
		})

		It("should use cluster-specific values", func() {
			err := opts.prepareValues()
			Expect(err).NotTo(HaveOccurred())

			var replicasValue *greenhousev1alpha1.PluginOptionValue
			for _, v := range opts.values {
				if v.Name == "replicas" {
					replicasValue = &v
					break
				}
			}

			Expect(replicasValue).NotTo(BeNil())
			Expect(string(replicasValue.Value.Raw)).To(Equal("5"))
		})
	})
})

var _ = Describe("processSecretsToLiterals", func() {
	var opts *PluginTemplatePresetOptions

	BeforeEach(func() {
		opts = &PluginTemplatePresetOptions{}
	})

	Context("with secret references", func() {
		It("should convert to literal format", func() {
			input := []greenhousev1alpha1.PluginOptionValue{
				{
					Name: "password",
					ValueFrom: &greenhousev1alpha1.ValueFromSource{
						Secret: &greenhousev1alpha1.SecretKeyReference{
							Name: "my-secret",
							Key:  "password",
						},
					},
				},
			}

			result, err := opts.processSecretsToLiterals(input)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("password"))
			Expect(string(result[0].Value.Raw)).To(Equal(`"my-secret/password"`))
			Expect(result[0].ValueFrom).To(BeNil())
		})
	})
})
