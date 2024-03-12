// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/helm"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("helm package test", func() {
	helmChart := &chart.Chart{
		Values: map[string]interface{}{
			"key1": "helmValue1",
			"key2": "helmValue2",
		},
	}

	When("getting the values for the Helm chart of a pluginConfig", func() {
		It("should correctly get regular values and overwrite helm values", func() {
			pluginConfig.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{*optionValue}
			helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, pluginConfig, true)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error getting the values")
			Expect(helmValues).ShouldNot(BeNil(),
				"the values should not be empty")

			Expect(helmValues).To(ContainElement("pluginValue1"))
			Expect(helmValues).To(ContainElement("helmValue2"))
		})

		It("should correctly get a value stored in a secret", func() {
			pluginConfig.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{*secretOptionValue}
			Expect(test.K8sClient.Create(test.Ctx, pluginSecret, &client.CreateOptions{})).
				Should(Succeed(), "creating an secret should be successful")

			helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, pluginConfig, true)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error getting the values")
			Expect(helmValues).ShouldNot(BeNil(),
				"the values should not be empty")

			Expect(helmValues).To(ContainElement("pluginSecretValue1"))
		})

		It("should correctly merge default values from the plugin spec and greenhouse values with plugin config", func() {
			pluginConfig.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{*optionValue}
			Expect(test.K8sClient.Create(test.Ctx, testPluginWithHelmChart)).
				Should(Succeed(), "creating a plugin should be successful")
			Expect(test.K8sClient.Create(test.Ctx, team)).
				Should(Succeed(), "creating a team should be successful")
			pluginOptionValues, err := helm.GetPluginOptionValuesForPluginConfig(test.Ctx, test.K8sClient, pluginConfig)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the plugin option values")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousesapv1alpha1.PluginOptionValue{Name: "key1", Value: test.MustReturnJSONFor("pluginValue1"), ValueFrom: nil}), "the plugin option values should contain default from plugin spec")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousesapv1alpha1.PluginOptionValue{Name: "greenhouse.teamNames", Value: test.MustReturnJSONFor([]string{"test-team-1"}), ValueFrom: nil}), "the plugin option values should contain greenhouse values")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousesapv1alpha1.PluginOptionValue{Name: "greenhouse.clusterName", Value: test.MustReturnJSONFor(pluginConfig.Spec.ClusterName), ValueFrom: nil}), "the plugin option values should contain the clusterName from the pluginConfig")
			Expect(pluginOptionValues).To(
				ContainElement(greenhousesapv1alpha1.PluginOptionValue{Name: "greenhouse.organizationName", Value: test.MustReturnJSONFor(pluginConfig.GetNamespace()), ValueFrom: nil}), "the plugin option values should contain the orgName from the pluginConfig namspace")
		})
	})

	When("handling a helm chart from a plugin", func() {
		It("should correctly error on missing helm chart reference", func() {
			pluginConfig.Spec.OptionValues = []greenhousesapv1alpha1.PluginOptionValue{*optionValue}
			err := helm.InstallOrUpgradeHelmChartFromPlugin(context.Background(), test.K8sClient, test.RestClientGetter, testPluginWithoutHelmChart, pluginConfig)
			Expect(err).Should(HaveOccurred(),
				"there should be an error for plugins without helm chart")

			Expect(err.Error()).To(ContainSubstring("no helm chart defined in plugin.Spec.HelmChart"), "the error should contain the correct message")
		})

		It("should correctly install a helm chart from a plugin", func() {
			err := helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChart, pluginConfig)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error for plugins with helm chart")

			cfg, err := helm.ExportNewHelmAction(test.RestClientGetter, pluginConfig.Namespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			listAction := action.NewList(cfg)
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			Expect(containsReleaseByName(releases, "test-plugin-config")).To(BeTrue(), "there should be a helm release with the name of the plugin-config")
		})

		It("should correctly uninstall a helm chart from a plugin", func() {
			releaseNotFound, err := helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, pluginConfig)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error uninstalling")
			// We expect the release from the previous test to be found
			Expect(releaseNotFound).To(BeFalse(), "the release should have been found before deleting")

			cfg, err := helm.ExportNewHelmAction(test.RestClientGetter, pluginConfig.Namespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			listAction := action.NewList(cfg)
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			Expect(containsReleaseByName(releases, pluginConfig.ObjectMeta.Name)).To(BeFalse(), "there should be no helm release with the name of the plugin-config")
		})

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

var _ = DescribeTable("getting helm values from PluginConfig", func(defaultValue any, exp any) {
	helmChart := &chart.Chart{
		Values: make(map[string]interface{}, 0),
	}

	pluginConfigWithOptionValue := &greenhousesapv1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "green",
			Name:      "house",
		},
		Spec: greenhousesapv1alpha1.PluginConfigSpec{
			Plugin: "greenhouse",
			OptionValues: []greenhousesapv1alpha1.PluginOptionValue{
				{
					Name:  "value1",
					Value: test.MustReturnJSONFor(defaultValue),
				},
			},
		},
	}

	helmValues, err := helm.ExportGetValuesForHelmChart(context.Background(), test.K8sClient, helmChart, pluginConfigWithOptionValue, true)
	Expect(err).ShouldNot(HaveOccurred(),
		"there should be no error getting the values")
	Expect(helmValues).ShouldNot(BeNil(),
		"the values should not be empty")

	val, ok := namedValueSliceValueByName(helmValues, "value1")
	Expect(ok).Should(BeTrue(), "the helm values should contain the of the PluginConfig")
	Expect(val).Should(Equal(exp), "the helm values should equal the one from the PluginConfig")
},
	Entry("should get the string default value", "1", "1"),
	Entry("should get the string default value with special chars", "1,2,3,key=test", "1,2,3,key=test"),
	Entry("should get the bool default value", true, true),
	Entry("should get the list default value", []string{"one", "two"}, []any{"one", "two"}),
	// Int decoded as float64, known helm issue https://github.com/helm/helm/issues/1707
	Entry("should get the int default value", 1, 1.0),
	Entry("should get the map default value", map[string]any{"key": "value"}, map[string]any{"key": "value"}),
)

func namedValueSliceValueByName(valuesMap map[string]interface{}, valueName string) (any, bool) {
	for k, v := range valuesMap {
		if k == valueName {
			return v, true
		}
	}
	return nil, false
}

func containsReleaseByName(releases []*release.Release, releaseName string) bool {
	for _, r := range releases {
		if r.Name == releaseName {
			return true
		}
	}
	return false
}
