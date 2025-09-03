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
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		It("should correctly error on missing helm chart reference", func() {
			plugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{*optionValue}
			err := helm.InstallOrUpgradeHelmChartFromPlugin(context.Background(), test.K8sClient, test.RestClientGetter, testPluginWithoutHelmChart.Spec, plugin)
			Expect(err).Should(HaveOccurred(),
				"there should be an error for pluginDefinitions without helm chart")

			Expect(err.Error()).To(ContainSubstring("no helm chart defined in pluginDefinition.Spec.HelmChart"), "the error should contain the correct message")
		})

		It("should correctly install a helm chart from a pluginDefinition", func() {
			err := helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChart.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error for plugindefinitions with helm chart")

			cfg, err := helm.ExportNewHelmAction(test.RestClientGetter, plugin.Spec.ReleaseNamespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			listAction := action.NewList(cfg)
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			Expect(containsReleaseByName(releases, "test-plugin")).To(BeTrue(), "there should be a helm release with the name of the plugin")
		})

		It("should correctly uninstall a helm chart from a pluginDefinition", func() {
			releaseNotFound, err := helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error uninstalling")
			// We expect the release from the previous test to be found
			Expect(releaseNotFound).To(BeFalse(), "the release should have been found before deleting")

			cfg, err := helm.ExportNewHelmAction(test.RestClientGetter, plugin.Spec.ReleaseNamespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			listAction := action.NewList(cfg)
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			Expect(containsReleaseByName(releases, plugin.ObjectMeta.Name)).To(BeFalse(), "there should be no helm release with the name of the plugin")
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

	When("handling a helm chart with CRDs", Ordered, func() {
		It("should re-create CRDs from Helm chart when CRD is missing on upgrade", func() {
			By("installing helm chart")
			err := helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error installing helm chart")

			By("getting the Team CRD")
			var teamCRD = &apiextensionsv1.CustomResourceDefinition{}
			teamCRDName := "teams.greenhouse.fixtures"
			teamCRDKey := types.NamespacedName{Name: teamCRDName, Namespace: ""}
			err = test.K8sClient.Get(test.Ctx, teamCRDKey, teamCRD)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting Team CRD")

			By("deleting the Team CRD")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, teamCRD)

			By("upgrading helm chart")
			err = helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error upgrading helm chart")

			Eventually(func(g Gomega) {
				By("getting Team CRD")
				var teamCRD = &apiextensionsv1.CustomResourceDefinition{}
				g.Expect(test.K8sClient.Get(test.Ctx, teamCRDKey, teamCRD)).To(Succeed(), "there must be no error getting the Team CRD")
				g.Expect(teamCRD.Name).To(Equal(teamCRDName), "re-created Team CRD should have the correct name")
			}).Should(Succeed(), "should re-create CRDs from Helm chart")

			By("cleaning up test")
			_, err = helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, plugin)
			Expect(err).ToNot(HaveOccurred(), "there must be no error uninstalling helm release")
		})

		It("should not create CRDs from Helm chart when CRD is missing on templating", func() {
			By("installing helm chart")
			err := helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error installing helm chart")

			By("getting the Team CRD")
			var teamCRD = &apiextensionsv1.CustomResourceDefinition{}
			teamCRDName := "teams.greenhouse.fixtures"
			teamCRDKey := types.NamespacedName{Name: teamCRDName, Namespace: ""}
			err = test.K8sClient.Get(test.Ctx, teamCRDKey, teamCRD)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting Team CRD")

			By("deleting the Team CRD")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, teamCRD)

			By("templating the Helm Chart from the Plugin")
			_, err = helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

			By("getting the Team CRD")
			err = test.K8sClient.Get(test.Ctx, teamCRDKey, teamCRD)
			Expect(err).To(HaveOccurred(), "Team CRD should not be re-created")

			By("cleaning up test")
			_, err = helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, plugin)
			Expect(err).ToNot(HaveOccurred(), "there must be no error uninstalling helm release")
		})
	})

	When("helm install fails at initial release", func() {
		It("should rollback to the same initial version", func() {
			By("installing a helm chart from a pluginDefinition")
			err := helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error for installing helm chart from plugin")

			By("getting the deployed helm release")
			cfg, err := helm.ExportNewHelmAction(test.RestClientGetter, plugin.Spec.ReleaseNamespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			getAction := action.NewGet(cfg)
			helmRelease, err := getAction.Run(plugin.Name)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
			Expect(helmRelease.Info.Status).To(Equal(release.StatusDeployed), "helm release should be set to deployed")

			By("fake failing helm install by setting the deployed helm release to failed")
			helmRelease.SetStatus(release.StatusFailed, "release set to failed manually from a test")
			err = cfg.Releases.Update(helmRelease)
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the helm release")

			By("checking that the helm release status is set to failed")
			getAction = action.NewGet(cfg)
			helmRelease, err = getAction.Run(plugin.Name)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
			Expect(helmRelease.Info.Status).To(Equal(release.StatusFailed), "helm release should be set to failed")

			By("running install or upgrade again")
			err = helm.InstallOrUpgradeHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, testPluginWithHelmChartCRDs.Spec, plugin)
			Expect(err).ShouldNot(HaveOccurred(),
				"there should be no error for upgrading the helm chart")

			By("checking that the helm release has been fixed")
			getAction = action.NewGet(cfg)
			helmRelease, err = getAction.Run(plugin.Name)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
			Expect(helmRelease.Info.Status).To(Equal(release.StatusDeployed), "helm release should be set to deployed")

			By("cleaning up test")
			_, err = helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, plugin)
			Expect(err).ToNot(HaveOccurred(), "there must be no error uninstalling helm release")
		})
	})
})

var _ = DescribeTable("getting helm values from Plugin", func(defaultValue any, exp any) {
	helmChart := &chart.Chart{
		Values: make(map[string]any),
	}

	pluginWithOptionValue := test.NewPlugin(test.Ctx, "green", "house",
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "test-team-1"),
		test.WithPluginDefinition("greenhouse"),
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

func containsReleaseByName(releases []*release.Release, releaseName string) bool {
	for _, r := range releases {
		if r.Name == releaseName {
			return true
		}
	}
	return false
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
				ValueFrom: &greenhousev1alpha1.ValueFromSource{
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
				ValueFrom: &greenhousev1alpha1.ValueFromSource{
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
				ValueFrom: &greenhousev1alpha1.ValueFromSource{
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
