// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("helm package test", func() {
	helmChart := &chart.Chart{
		Values: map[string]any{
			"key1": "helmValue1",
			"key2": "helmValue2",
		},
	}

	When("getting the values for the Helm chart of a plugin", func() {
		It("should correctly get regular values and overwrite helm values", func() {
			plugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{*optionValue}
			helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error getting the values")
			Expect(helmValues).ShouldNot(BeNil(),
				"the values should not be empty")

			Expect(helmValues).To(ContainElement("pluginValue1"))
			Expect(helmValues).To(ContainElement("helmValue2"))
		})

		It("should correctly get a value stored in a secret", func() {
			plugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{*secretOptionValue}
			Expect(test.K8sClient.Create(test.Ctx, pluginSecret, &client.CreateOptions{})).
				Should(Succeed(), "creating an secret should be successful")

			helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error getting the values")
			Expect(helmValues).ShouldNot(BeNil(),
				"the values should not be empty")

			Expect(helmValues).To(ContainElement("pluginSecretValue1"))
		})

		It("should correctly merge default values from the pluginDefinition spec and greenhouse values with plugin", func() {
			plugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{*optionValue}
			Expect(test.K8sClient.Create(test.Ctx, testPluginWithHelmChart)).
				Should(Succeed(), "creating a pluginDefinition should be successful")
			Expect(test.K8sClient.Create(test.Ctx, team)).
				Should(Succeed(), "creating a team should be successful")
			pluginOptionValues, err := helm.GetPluginOptionValuesForPlugin(test.Ctx, test.K8sClient, plugin)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the pluginDefinition option values")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "key1", Value: test.MustReturnJSONFor("pluginValue1"), ValueFrom: nil}), "the plugin option values should contain default from pluginDefinition spec")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "global.greenhouse.teamNames", Value: test.MustReturnJSONFor([]string{"test-team-1"}), ValueFrom: nil}), "the plugin option values should contain greenhouse values")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor(plugin.Spec.ClusterName), ValueFrom: nil}), "the plugin option values should contain the clusterName from the plugin")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "global.greenhouse.organizationName", Value: test.MustReturnJSONFor(plugin.GetNamespace()), ValueFrom: nil}), "the plugin option values should contain the orgName from the plugin namespace")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "global.greenhouse.baseDomain", Value: test.MustReturnJSONFor(common.DNSDomain), ValueFrom: nil}), "the plugin option values should contain the baseDomain")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "global.greenhouse.ownedBy", Value: test.MustReturnJSONFor(plugin.Labels[string(greenhouseapis.LabelKeyOwnedBy)]), ValueFrom: nil}), "the plugin option values should contain the owning team")
		})
	})

	When("handling a helm chart from a pluginDefinition", func() {
		It("should configure the chartPathOptions correctly for OCI repositories", func() {
			cpo := action.ChartPathOptions{}
			chartName := helm.ExportConfigureChartPathOptions(&cpo, testPluginWithHelmChartOCI.Spec.HelmChart)

			Expect(chartName).Should(Equal(fmt.Sprintf("%s/%s", testPluginWithHelmChartOCI.Spec.HelmChart.Repository, testPluginWithHelmChartOCI.Spec.HelmChart.Name)))
			Expect(cpo.RepoURL).Should(Equal(""))
			Expect(cpo.Version).Should(Equal(testPluginWithHelmChartOCI.Spec.HelmChart.Version))
		})

		It("should not overwrite repoURL and chartName for non-oci", func() {
			cpo := action.ChartPathOptions{}
			chartName := helm.ExportConfigureChartPathOptions(&cpo, testPluginWithHelmChart.Spec.HelmChart)
			Expect(chartName).Should(Equal(testPluginWithHelmChart.Spec.HelmChart.Name))
			Expect(cpo.RepoURL).Should(Equal(testPluginWithHelmChart.Spec.HelmChart.Repository))
			Expect(cpo.Version).Should(Equal(testPluginWithHelmChart.Spec.HelmChart.Version))
		})
	})
})

var _ = DescribeTable("getting helm values from Plugin", func(defaultValue any, exp any) {
	helmChart := &chart.Chart{
		Values: make(map[string]any),
	}

	pluginWithOptionValue := test.NewPlugin(test.Ctx, "green", "house",
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "test-team-1"),
		test.WithClusterPluginDefinition("greenhouse"),
		test.WithPluginOptionValue("value1", test.MustReturnJSONFor(defaultValue)),
	)

	helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, pluginWithOptionValue)
	Expect(err).ShouldNot(HaveOccurred(),
		"there should be no error getting the values")
	Expect(helmValues).ShouldNot(BeNil(),
		"the values should not be empty")

	val, ok := namedValueSliceValueByName(helmValues, "value1")
	Expect(ok).Should(BeTrue(), "the helm values should contain the of the Plugin")
	Expect(val).Should(Equal(exp), "the helm values should equal the one from the Plugin")
},
	Entry("should get the string default value", "1", "1"),
	Entry("should get the string default value with special chars", "1,2,3,key=test", "1,2,3,key=test"),
	Entry("should get the bool default value", true, true),
	Entry("should get the list default value", []string{"one", "two"}, []any{"one", "two"}),
	// Int decoded as float64, known helm issue https://github.com/helm/helm/issues/1707
	Entry("should get the int default value", 1, 1.0),
	Entry("should get the map default value", map[string]any{"key": "value"}, map[string]any{"key": "value"}),
)

func namedValueSliceValueByName(valuesMap map[string]any, valueName string) (any, bool) {
	for k, v := range valuesMap {
		if k == valueName {
			return v, true
		}
	}
	return nil, false
}

var _ = Describe("Plugin option checksum", Ordered, func() {
	var (
		secretWithOptionValue = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-org",
				Name:      "plugin-secret",
			},
			Data: map[string][]byte{
				"secretKey": []byte("pluginSecretValue1"),
			},
		}

		optionValuesOneRequired = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "stringRequired",
				Value: test.MustReturnJSONFor("required"),
			},
		}
		optionValuesOneSecret = []greenhousev1alpha1.PluginOptionValue{
			{
				Name: "secret",
				ValueFrom: &greenhousev1alpha1.PluginValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "plugin-secret",
						Key:  "secretKey",
					},
				},
			},
		}
		optionValuesRequiredAndSecret = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "stringRequired",
				Value: test.MustReturnJSONFor("required"),
			},
			{
				Name: "secret",
				ValueFrom: &greenhousev1alpha1.PluginValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "plugin-secret",
						Key:  "secretKey",
					},
				},
			},
		}
		optionValuesSecretAndRequired = []greenhousev1alpha1.PluginOptionValue{
			{
				Name: "secret",
				ValueFrom: &greenhousev1alpha1.PluginValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "plugin-secret",
						Key:  "secretKey",
					},
				},
			},
			{
				Name:  "stringRequired",
				Value: test.MustReturnJSONFor("required"),
			},
		}
		optionValuesRequiredAndOptional = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "stringRequired",
				Value: test.MustReturnJSONFor("required"),
			},
			{
				Name:  "key1",
				Value: test.MustReturnJSONFor("pluginValue1"),
			},
		}
		optionValuesEmpty []greenhousev1alpha1.PluginOptionValue
	)

	BeforeAll(func() {
		// Add secrets for test cases.
		Expect(test.K8sClient.Create(test.Ctx, secretWithOptionValue)).To(Succeed(), "there should be no error creating a secret")
	})

	AfterAll(func() {
		Expect(test.K8sClient.Delete(test.Ctx, secretWithOptionValue)).To(Succeed(), "there should be no error deleting a secret")
	})

	var _ = DescribeTable("comparing plugin option checksums",
		func(optionValues1 []greenhousev1alpha1.PluginOptionValue, optionValues2 []greenhousev1alpha1.PluginOptionValue, expected bool) {
			plugin1 := greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hashing-plugin1",
					Namespace: "test-org",
				},
			}
			plugin2 := greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hashing-plugin2",
					Namespace: "test-org",
				},
			}
			plugin1.Spec.OptionValues = optionValues1
			plugin2.Spec.OptionValues = optionValues2
			hashedValues1, err := helm.CalculatePluginOptionChecksum(test.Ctx, test.K8sClient, &plugin1)
			Expect(err).ToNot(HaveOccurred(), "there should be no error calculating plugin option checksum")
			hashedValues2, err := helm.CalculatePluginOptionChecksum(test.Ctx, test.K8sClient, &plugin2)
			Expect(err).ToNot(HaveOccurred(), "there should be no error calculating plugin option checksum")

			comparisonResult := hashedValues1 == hashedValues2
			Expect(comparisonResult).To(Equal(expected))
		},
		Entry("the same option values should be equal", optionValuesOneRequired, optionValuesOneRequired, true),
		Entry("the same option values should be equal", optionValuesRequiredAndSecret, optionValuesRequiredAndSecret, true),
		Entry("the same option values in different order should be equal", optionValuesSecretAndRequired, optionValuesRequiredAndSecret, true),
		Entry("different option values should not be equal", optionValuesOneRequired, optionValuesOneSecret, false),
		Entry("different option values should not be equal", optionValuesEmpty, optionValuesOneRequired, false),
		Entry("different option values should not be equal", optionValuesRequiredAndOptional, optionValuesRequiredAndSecret, false),
	)
})
