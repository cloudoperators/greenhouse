// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	greenhousecluster "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	greenhouseDef "github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestHelmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HelmControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("plugin", (&PluginReconciler{
		KubeRuntimeOpts: clientutil.RuntimeOptions{QPS: 5, Burst: 10},
	}).SetupWithManager)
	test.RegisterController("pluginPreset", (&PluginPresetReconciler{}).SetupWithManager)
	test.RegisterController("pluginDefinition", (&greenhouseDef.PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("clusterPluginDefinition", (&greenhouseDef.ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)
	test.RegisterWebhook("organizationWebhook", webhookv1alpha1.SetupOrganizationWebhookWithManager)
	test.RegisterWebhook("TeamWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", webhookv1alpha1.SetupPluginWebhookWithManager)
	test.RegisterWebhook("clusterWebhook", webhookv1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", webhookv1alpha1.SetupSecretWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", webhookv1alpha1.SetupPluginPresetWebhookWithManager)
	test.TestBeforeSuite()

	// return the test.Cfg, as the in-cluster config is not available
	ctrl.GetConfig = func() (*rest.Config, error) {
		return test.Cfg, nil
	}
})

// getNestedValue retrieves a value from a nested map using a dot-separated key path.
// For example, "foo.bar.baz" will traverse map["foo"]["bar"]["baz"].
// Returns the value and true if found, nil and false otherwise.
func getNestedValue(m map[string]any, key string) (any, bool) {
	// Handle simple keys without dots
	if val, ok := m[key]; ok {
		return val, true
	}

	// Split the key by dots and traverse the nested structure
	keys := strings.Split(key, ".")

	current := any(m)
	for _, k := range keys {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[k]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// verifyPluginHelmIntegration validates the complete Plugin -> HelmRelease -> HelmChart integration.
// It verifies:
// 1. HelmRelease exists for the Plugin
// 2. HelmChart referenced by HelmRelease exists and matches PluginDefinition's HelmChart spec
// 3. PluginOptionValues are correctly passed to the HelmRelease values
func verifyPluginHelmIntegration(g Gomega, plugin *greenhousev1alpha1.Plugin, pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition) {
	GinkgoHelper()

	// Fetch the HelmRelease for this Plugin
	helmRelease := &helmv2.HelmRelease{}
	helmReleaseID := types.NamespacedName{Name: plugin.Name, Namespace: plugin.Namespace}
	err := test.K8sClient.Get(test.Ctx, helmReleaseID, helmRelease)
	g.Expect(err).ToNot(HaveOccurred(), "HelmRelease should exist for Plugin")
	g.Expect(helmRelease.Spec.ChartRef).ToNot(BeNil(), "HelmRelease should reference a HelmChart")

	// Fetch the HelmChart referenced by the HelmRelease
	helmChart := &sourcev1.HelmChart{}
	helmChartID := types.NamespacedName{
		Name:      helmRelease.Spec.ChartRef.Name,
		Namespace: helmRelease.Spec.ChartRef.Namespace,
	}
	err = test.K8sClient.Get(test.Ctx, helmChartID, helmChart)
	g.Expect(err).ToNot(HaveOccurred(), "HelmChart referenced by HelmRelease should exist")

	// Verify HelmChart matches PluginDefinition's HelmChart specification
	g.Expect(helmChart.Spec.Chart).To(Equal(pluginDefinition.Spec.HelmChart.Name), "HelmChart name should match PluginDefinition")
	g.Expect(helmChart.Spec.Version).To(Equal(pluginDefinition.Spec.HelmChart.Version), "HelmChart version should match PluginDefinition")

	// Verify PluginOptionValues are present in HelmRelease values
	if len(plugin.Spec.OptionValues) > 0 {
		g.Expect(helmRelease.Spec.Values).ToNot(BeNil(), "HelmRelease should have values when Plugin has OptionValues")

		helmReleaseValues := make(map[string]any)
		err = json.Unmarshal(helmRelease.Spec.Values.Raw, &helmReleaseValues)
		g.Expect(err).ToNot(HaveOccurred(), "HelmRelease values should be valid JSON")

		// Verify each PluginOptionValue is present in HelmRelease values
		for _, optionValue := range plugin.Spec.OptionValues {
			var expectedValue any
			err = json.Unmarshal(optionValue.Value.Raw, &expectedValue)
			g.Expect(err).ToNot(HaveOccurred(), "PluginOptionValue should be valid JSON")

			// Handle nested keys (e.g., "foo.bar.baz")
			actualValue, found := getNestedValue(helmReleaseValues, optionValue.Name)
			g.Expect(found).To(BeTrue(), "HelmRelease values should contain key %s", optionValue.Name)
			g.Expect(actualValue).To(Equal(expectedValue), "HelmRelease value for %s should match expected value", optionValue.Name)
		}
	}
}

// HelmReconcilerTest performs tests in Serial mode to avoid conflicts with the k8s resources
var _ = Describe("HelmControllerTest", Serial, func() {

	const (
		PluginDefinitionName           = "mytestplugin"
		PluginDefinitionVersion        = "1.0.0"
		PluginDefinitionVersionUpdated = "1.1.0"
		PluginDefinitionChartVersion   = "1.0.0"

		PluginOptionRequired     = "myRequiredOption"
		PluginOptionOptional     = "myOptionalOption"
		PluginOptionDefault      = "myDefaultOption"
		PluginOptionDefaultValue = "myDefaultValue"
		PluginOptionBool         = "myBoolOption"
		PluginOptionBoolDefault  = true
		PluginOptionInt          = "myIntOption"
		PluginOptionIntDefault   = 1

		PluginName                = "myplugin"
		PluginRequiredOptionValue = "required"

		Namespace                   = "greenhouse"
		ReleaseName                 = "myplugin-release"
		HelmChartName               = "dummy"
		HelmChartUpdatedName        = "dummy-updated"
		HelmChartWithAllOptionsName = "chart-with-every-option"
	)

	var (
		PluginOptionList        = "myListOption"
		PluginOptionListDefault = []any{"myListValue1", "myListValue2"}
		PluginOptionMap         = "myMapOption"
		PluginOptionMapDefault  = map[string]any{"myMapKey1": "myMapValue1", "myMapKey2": "myMapValue2"}

		testTeam             *greenhousev1alpha1.Team
		testPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
		testPlugin           *greenhousev1alpha1.Plugin
		pluginDefinitionID   = types.NamespacedName{Name: PluginDefinitionName, Namespace: ""}
		pluginID             = types.NamespacedName{Name: PluginName, Namespace: Namespace}
		helmReleaseID        = types.NamespacedName{Name: PluginName, Namespace: Namespace}
		tempChartLoader      helm.ChartLoaderFunc
	)

	BeforeEach(func() {
		Expect(client.IgnoreAlreadyExists(test.K8sClient.Create(test.Ctx, test.NewOrganization(test.Ctx, Namespace)))).To(Succeed(), "there must be no error creating the greenhouse organization for the test")

		// remember original chart loader, which is overwritten in some tests
		tempChartLoader = helm.ChartLoader

		testPluginDefinition = test.NewClusterPluginDefinition(test.Ctx, PluginDefinitionName,
			test.WithVersion(PluginDefinitionVersion),
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:       HelmChartName,
				Repository: "oci://greenhouse/helm-charts",
				Version:    PluginDefinitionChartVersion,
			}),
			test.AppendPluginOption(greenhousev1alpha1.PluginOption{
				Name:        PluginOptionRequired,
				Description: "This is my required test plugin option",
				Required:    true,
				Type:        greenhousev1alpha1.PluginOptionTypeString,
			}),
			test.AppendPluginOption(greenhousev1alpha1.PluginOption{
				Name:        PluginOptionOptional,
				Description: "This is my optional test plugin option",
				Required:    false,
				Type:        greenhousev1alpha1.PluginOptionTypeString,
			}),
			test.AppendPluginOption(greenhousev1alpha1.PluginOption{
				Name:        PluginOptionDefault,
				Description: "This is my default test plugin option",
				Required:    false,
				Default:     test.MustReturnJSONFor(PluginOptionDefaultValue),
				Type:        greenhousev1alpha1.PluginOptionTypeString,
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, testPluginDefinition)).Should(Succeed())
		actPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		Eventually(func() bool {
			err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
			if err != nil {
				return false
			}
			return actPluginDefinition.Spec.Version == PluginDefinitionVersion
		}).Should(BeTrue())

		By("mocking HelmChart Ready condition for testPluginDefinition")
		test.MockHelmChartReady(test.Ctx, test.K8sClient, testPluginDefinition, flux.HelmRepositoryDefaultNamespace)

		testTeam = test.NewTeam(test.Ctx, "suite-test-team", Namespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		Expect(test.K8sClient.Create(test.Ctx, testTeam)).Should(Succeed(), "there should be no error creating the Team")

		testPlugin = test.NewPlugin(test.Ctx, PluginName, Namespace,
			test.WithClusterPluginDefinition(PluginDefinitionName),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithReleaseName(ReleaseName),
			test.WithPluginOptionValue(PluginOptionRequired, test.MustReturnJSONFor(PluginRequiredOptionValue)))

		Expect(test.K8sClient.Create(test.Ctx, testPlugin)).Should(Succeed())

		By("verifying Plugin conditions and Helm integration")
		actPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, pluginID, actPlugin)).To(Succeed())
			g.Expect(test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)).To(Succeed())
			verifyPluginHelmIntegration(g, actPlugin, actPluginDefinition)
		}).Should(Succeed())
	})

	AfterEach(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPlugin)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginDefinition)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testTeam)
		// revert to original chart loader
		helm.ChartLoader = tempChartLoader
	})

	When("a pluginDefinition and its chart were updated", func() {
		It("should reconcile the Plugin to a newer PluginDefinition version", func() {
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, testPluginDefinition)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginDefinition")
				testPluginDefinition.Spec.HelmChart.Name = HelmChartUpdatedName
				testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
				g.Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
				test.MockHelmChartReady(test.Ctx, test.K8sClient, testPluginDefinition, flux.HelmRepositoryDefaultNamespace)
				return true
			}).Should(BeTrue())

			actPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
				if err != nil {
					g.Expect(err).ToNot(HaveOccurred(), "error getting pluginDefinition")
					return false
				}
				return actPluginDefinition.Spec.Version == PluginDefinitionVersionUpdated
			}).Should(BeTrue())

			By("verifying the Plugin was reconciled")
			actPlugin := &greenhousev1alpha1.Plugin{}
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
				g.Expect(err).ToNot(HaveOccurred(), "error getting plugin")
				verifyPluginHelmIntegration(g, actPlugin, actPluginDefinition)
			}).Should(Succeed(), "Plugin should be reconciled to updated PluginDefinition")
		})
	})

	When("the pluginDefinition version was increased", func() {
		It("should reconcile the Plugin", func() {
			By("increasing the pluginDefinition version")
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, testPluginDefinition)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginDefinition")
				testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
				g.Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
				test.MockHelmChartReady(test.Ctx, test.K8sClient, testPluginDefinition, flux.HelmRepositoryDefaultNamespace)
				return true
			}).Should(BeTrue())
			actPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
				if err != nil {
					g.Expect(err).ToNot(HaveOccurred(), "error getting pluginDefinition")
					return false
				}
				return actPluginDefinition.Spec.Version == PluginDefinitionVersionUpdated
			}).Should(BeTrue())

			By("verifying the Plugin was reconciled")
			actPlugin := &greenhousev1alpha1.Plugin{}
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
				if err != nil {
					g.Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				verifyPluginHelmIntegration(g, actPlugin, actPluginDefinition)
				return true
			}).Should(BeTrue())
		})
	})

	When("the plugin was deleted", func() {
		It("should delete the HelmRelease", func() {
			By("deleting the plugin")
			Expect(test.K8sClient.Delete(test.Ctx, testPlugin)).Should(Succeed(), "errored deleting plugin")

			By("verifying the HelmRelease is deleted")
			Eventually(func(g Gomega) {
				helmRelease := &helmv2.HelmRelease{}
				err := test.K8sClient.Get(test.Ctx, helmReleaseID, helmRelease)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "HelmRelease should be deleted")
			}).Should(Succeed())
		})
	})

	It("should successfully create a HelmRelease with every type of OptionValue", func() {
		const pluginWithEveryOption = "mytestpluginwitheveryoption"
		var (
			complexPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
			complexPlugin           *greenhousev1alpha1.Plugin
			pluginName              = "mypluginwitheveryoption"
			complexPluginID         = types.NamespacedName{Name: pluginName, Namespace: Namespace}

			stringVal = "myStringValue"
			boolVal   = true
			intVal    = 1
			listVal   = []any{"myListValue1", "myListValue2"}
			mapVal    = map[string]any{"myMapKey1": "myMapValue1", "myMapKey2": "myMapValue2"}
		)

		By("creating a pluginDefinition with every type of option", func() {
			complexPluginDefinition = test.NewClusterPluginDefinition(test.Ctx, pluginWithEveryOption,
				test.WithVersion(PluginDefinitionVersion),
				test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
					Name:       HelmChartWithAllOptionsName,
					Repository: "oci://greenhouse/helm-charts",
					Version:    PluginDefinitionChartVersion,
				}),
				test.AppendPluginOption(greenhousev1alpha1.PluginOption{
					Name:        PluginOptionDefault,
					Description: "This is my default test plugin option",
					Required:    false,
					Default:     test.MustReturnJSONFor(PluginOptionDefaultValue),
					Type:        greenhousev1alpha1.PluginOptionTypeString,
				}),
				test.AppendPluginOption(greenhousev1alpha1.PluginOption{
					Name:        PluginOptionBool,
					Description: "This is my default test plugin option with a bool value",
					Required:    false,
					Default:     test.MustReturnJSONFor(PluginOptionBoolDefault),
					Type:        greenhousev1alpha1.PluginOptionTypeBool,
				}),
				test.AppendPluginOption(greenhousev1alpha1.PluginOption{
					Name:        PluginOptionInt,
					Description: "This is my default test plugin option with an int value",
					Required:    false,
					Default:     test.MustReturnJSONFor(PluginOptionIntDefault),
					Type:        greenhousev1alpha1.PluginOptionTypeInt,
				}),
				test.AppendPluginOption(greenhousev1alpha1.PluginOption{
					Name:        PluginOptionList,
					Description: "This is my default test plugin option with a list value",
					Required:    false,
					Default:     test.MustReturnJSONFor(PluginOptionListDefault),
					Type:        greenhousev1alpha1.PluginOptionTypeList,
				}),
				test.AppendPluginOption(greenhousev1alpha1.PluginOption{
					Name:        PluginOptionMap,
					Description: "This is my default test plugin option with a map value",
					Required:    false,
					Default:     test.MustReturnJSONFor(PluginOptionMapDefault),
					Type:        greenhousev1alpha1.PluginOptionTypeMap,
				}),
			)

			Expect(test.K8sClient.Create(test.Ctx, complexPluginDefinition)).Should(Succeed())
			complexPluginDefinitionID := types.NamespacedName{Name: complexPluginDefinition.Name, Namespace: ""}
			actComplexPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, complexPluginDefinitionID, actComplexPluginDefinition)
				if err != nil {
					return false
				}
				return actComplexPluginDefinition.Spec.Version == PluginDefinitionVersion
			}).Should(BeTrue())

			By("mocking HelmChart Ready condition for complexPluginDefinition")
			test.MockHelmChartReady(test.Ctx, test.K8sClient, complexPluginDefinition, flux.HelmRepositoryDefaultNamespace)
		})

		By("creating a Plugin with every type of OptionValue", func() {
			complexPlugin = test.NewPlugin(test.Ctx, pluginName, Namespace,
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
				test.WithClusterPluginDefinition(pluginWithEveryOption),
				test.WithReleaseName(ReleaseName),
				test.WithPluginOptionValue(PluginOptionDefault, test.MustReturnJSONFor(stringVal)),
				test.WithPluginOptionValue(PluginOptionBool, test.MustReturnJSONFor(boolVal)),
				test.WithPluginOptionValue(PluginOptionInt, test.MustReturnJSONFor(intVal)),
				test.WithPluginOptionValue(PluginOptionList, test.MustReturnJSONFor(listVal)),
				test.WithPluginOptionValue(PluginOptionMap, test.MustReturnJSONFor(mapVal)),
			)

			Expect(test.K8sClient.Create(test.Ctx, complexPlugin)).Should(Succeed())
		})

		By("verifying the HelmRelease is created and configured correctly", func() {
			Eventually(func(g Gomega) {
				actPlugin := &greenhousev1alpha1.Plugin{}
				err := test.K8sClient.Get(test.Ctx, complexPluginID, actPlugin)
				g.Expect(err).ToNot(HaveOccurred(), "Plugin should exist")
				verifyPluginHelmIntegration(g, actPlugin, complexPluginDefinition)
			}).Should(Succeed())
		})

		test.EventuallyDeleted(test.Ctx, test.K8sClient, complexPlugin)
	})
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
