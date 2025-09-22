// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"errors"
	"testing"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousecluster "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	greenhouseDef "github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestHelmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HelmControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("plugin", (&PluginReconciler{KubeRuntimeOpts: clientutil.RuntimeOptions{QPS: 5, Burst: 10}}).SetupWithManager)
	test.RegisterController("pluginPreset", (&PluginPresetReconciler{}).SetupWithManager)
	test.RegisterController("pluginDefinition", (&greenhouseDef.PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("clusterPluginDefinition", (&greenhouseDef.ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)
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

// checkReadyConditionComponentsUnderTest asserts that components of plugin's ReadyCondition are ready,
// except for WorkloadReady condition, which is not a subject under test.
// This is done because the cumulative Ready condition in tests will be false due to workload not being ready.
func checkReadyConditionComponentsUnderTest(g Gomega, plugin *greenhousev1alpha1.Plugin) {
	GinkgoHelper()
	readyCondition := plugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	g.Expect(readyCondition).ToNot(BeNil(), "Ready condition should not be nil")
	clusterAccessReadyCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
	g.Expect(clusterAccessReadyCondition).ToNot(BeNil())
	g.Expect(clusterAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "ClusterAccessReady condition should be true")
	helmReconcileFailedCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
	g.Expect(helmReconcileFailedCondition).ToNot(BeNil())
	g.Expect(helmReconcileFailedCondition.Status).To(Equal(metav1.ConditionFalse), "HelmReconcileFailed condition should be false")
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

		Namespace               = "greenhouse"
		ReleaseName             = "myplugin-release"
		HelmRepo                = "dummy"
		HelmChart               = "./../../test/fixtures/myChart"
		HelmChartUpdated        = "./../../test/fixtures/myChartV2"
		HelmChartWithAllOptions = "./../../test/fixtures/chartWithEveryOption"

		PodName             = "alpine"
		UpdatedImageString  = "alpine:3.18"
		UpdatedImageVersion = "3.18"
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
		tempChartLoader      helm.ChartLoaderFunc
	)

	BeforeEach(func() {
		Expect(client.IgnoreAlreadyExists(test.K8sClient.Create(test.Ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: Namespace,
			},
		}))).To(Succeed(), "there must be no error creating the test namespace")

		// remember original chart loader, which is overwritten in some tests
		tempChartLoader = helm.ChartLoader

		testPluginDefinition = test.NewClusterPluginDefinition(test.Ctx, PluginDefinitionName,
			test.WithVersion(PluginDefinitionVersion),
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:       HelmChart,
				Repository: HelmRepo,
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

		testTeam = test.NewTeam(test.Ctx, "suite-test-team", Namespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		Expect(test.K8sClient.Create(test.Ctx, testTeam)).Should(Succeed(), "there should be no error creating the Team")

		testPlugin = test.NewPlugin(test.Ctx, PluginName, Namespace,
			test.WithPluginDefinition(PluginDefinitionName),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithReleaseName(ReleaseName),
			test.WithPluginOptionValue(PluginOptionRequired, test.MustReturnJSONFor(PluginRequiredOptionValue)))

		Expect(test.K8sClient.Create(test.Ctx, testPlugin)).Should(Succeed())

		actPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
			if err != nil {
				return false
			}
			By("checking ReadyCondition selected components")
			checkReadyConditionComponentsUnderTest(g, actPlugin)
			g.Expect(actPlugin.Status.HelmReleaseStatus.Status).To(Equal("deployed"))
			return true
		}).Should(BeTrue())
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
				testPluginDefinition.Spec.HelmChart.Name = HelmChartUpdated
				testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
				g.Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
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
					Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				g.Expect(actPlugin.Status.Version).To(Equal(PluginDefinitionVersionUpdated))
				checkReadyConditionComponentsUnderTest(g, actPlugin)
				g.Expect(actPlugin.Status.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition).IsTrue()).To(BeTrue(), "StatusReconcileCompleteCondition should be true")
				return true
			}).Should(BeTrue())

			By("verifying newer Chart was deployed successfully")
			actPod := &corev1.Pod{}
			podID := types.NamespacedName{Name: PodName, Namespace: Namespace}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, podID, actPod)
				if err != nil {
					Expect(err).ToNot(HaveOccurred(), "error getting pod")
					return false
				}
				return actPod.Spec.Containers[0].Image == UpdatedImageString
			}).Should(BeTrue())
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
					Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				g.Expect(actPlugin.Status.Version).To(Equal(PluginDefinitionVersionUpdated))
				checkReadyConditionComponentsUnderTest(g, actPlugin)
				g.Expect(actPlugin.Status.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition).IsTrue()).To(BeTrue(), "StatusReconcileCompleteCondition should be true")
				return true
			}).Should(BeTrue())
		})
	})

	When("the pluginDefinition version was increased but the chart was changed without increasing the version", func() {
		It("should verify the Plugin was reconciled", func() {
			By("injecting different helm values for the same chart version")
			helm.ChartLoader = func(name string) (*chart.Chart, error) {
				values := map[string]any{
					"imageTag": UpdatedImageVersion,
				}
				chart, err := loader.Load(name)
				if err != nil {
					return nil, err
				}
				chart.Values = values
				return chart, nil
			}
			By("increasing the pluginDefinition version")
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, testPluginDefinition)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginDefinition")
				testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
				g.Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
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
					Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				g.Expect(actPlugin.Status.Version).To(Equal(PluginDefinitionVersionUpdated))
				checkReadyConditionComponentsUnderTest(g, actPlugin)
				g.Expect(actPlugin.Status.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition).IsTrue()).To(BeTrue(), "StatusReconcileCompleteCondition should be true")
				return true
			}).Should(BeTrue())
			By("verifying the Pod Image was actually updated")
			actPod := &corev1.Pod{}
			podID := types.NamespacedName{Name: PodName, Namespace: Namespace}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, podID, actPod)
				if err != nil {
					Expect(err).ToNot(HaveOccurred(), "error getting pod")
					return false
				}
				return actPod.Spec.Containers[0].Image == UpdatedImageString
			}).Should(BeTrue())
		})
	})

	When("the pluginDefinition has a chart depending on an older version of kubernetes", func() {
		It("should not reconcile the Plugin", func() {
			By("injecting an old kubernetes version for the chart")
			helm.ChartLoader = func(name string) (*chart.Chart, error) {
				chart, err := loader.Load(name)
				if err != nil {
					return nil, err
				}
				chart.Metadata.KubeVersion = "<=1.20.0-0"
				return chart, nil
			}

			By("increasing the pluginDefinition version")
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, testPluginDefinition)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the pluginDefinition")
				testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
				g.Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
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

			By("verifying the Plugin was not reconciled")
			actPlugin := &greenhousev1alpha1.Plugin{}
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
				if err != nil {
					g.Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				helmReconcileFailedCondition := actPlugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
				g.Expect(helmReconcileFailedCondition).ToNot(BeNil(), "HelmReconcileFailedCondition not found")
				g.Expect(helmReconcileFailedCondition.IsTrue()).To(BeTrue(), "HelmReconcileFailedCondition is not true")
				g.Expect(actPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition).IsTrue()).To(BeFalse(), "ReadyCondition should not be true (but unknown)")
				g.Expect(helmReconcileFailedCondition.Message).To(ContainSubstring("Helm template failed: chart requires kubeVersion: <=1.20.0-0"), "HelmReconcileFailedCondition message does not reflect kubernetes version error")
				return true
			}).Should(BeTrue())
		})
	})
	When("the plugin was deleted", func() {
		It("should delete the helm release", func() {
			By("deleting the plugin")
			Expect(test.K8sClient.Delete(test.Ctx, testPlugin)).Should(Succeed(), "errored deleting plugin")
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, pluginID, testPlugin)
				return err != nil
			}).Should(BeFalse())
			Eventually(func() bool {
				_, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, clientutil.NewRestClientGetterFromRestConfig(test.Cfg, testPlugin.Spec.ReleaseNamespace, clientutil.WithPersistentConfig()), testPlugin)
				if err != nil {
					return errors.Is(err, driver.ErrReleaseNotFound)
				}
				return false
			}).Should(BeTrue())
		})
	})

	It("should correctly get a default value from the pluginDefinition spec", func() {
		actPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
		}).Should(Succeed())
		Expect(actPluginDefinition.Spec.Options).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(PluginOptionDefault), "Default": Equal(test.MustReturnJSONFor(PluginOptionDefaultValue))})))

		actPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
		}).Should(Succeed())

		Expect(actPlugin.Spec.OptionValues).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(PluginOptionDefault), "Value": Equal(test.MustReturnJSONFor(PluginOptionDefaultValue))})))
	})

	It("should successfully create a Plugin with every type of OptionValue", func() {
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
					Name:       HelmChartWithAllOptions,
					Repository: HelmRepo,
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
		})

		By("creating a Plugin with every type of OptionValue", func() {
			complexPlugin = test.NewPlugin(test.Ctx, pluginName, Namespace,
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
				test.WithPluginDefinition(pluginWithEveryOption),
				test.WithReleaseName(ReleaseName),
				test.WithPluginOptionValue(PluginOptionDefault, test.MustReturnJSONFor(stringVal)),
				test.WithPluginOptionValue(PluginOptionBool, test.MustReturnJSONFor(boolVal)),
				test.WithPluginOptionValue(PluginOptionInt, test.MustReturnJSONFor(intVal)),
				test.WithPluginOptionValue(PluginOptionList, test.MustReturnJSONFor(listVal)),
				test.WithPluginOptionValue(PluginOptionMap, test.MustReturnJSONFor(mapVal)),
			)

			Expect(test.K8sClient.Create(test.Ctx, complexPlugin)).Should(Succeed())
			actComplexPlugin := &greenhousev1alpha1.Plugin{}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, complexPluginID, actComplexPlugin)
				if err != nil {
					return false
				}
				return complexPluginDefinition.Spec.Version == PluginDefinitionVersion
			}).Should(BeTrue())
		})

		By("successfully reconciling the Plugin", func() {
			actPlugin := &greenhousev1alpha1.Plugin{}
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, complexPluginID, actPlugin)
				if err != nil {
					Expect(err).ToNot(HaveOccurred(), "error getting plugin")
					return false
				}
				checkReadyConditionComponentsUnderTest(g, actPlugin)
				g.Expect(actPlugin.Status.Version).To(Equal(PluginDefinitionVersion))
				return true
			}).Should(BeTrue())
		})

		By("checking the Helm Release has the expected values set", func() {
			Eventually(func(g Gomega) {
				release, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, clientutil.NewRestClientGetterFromRestConfig(test.Cfg, complexPlugin.Namespace), complexPlugin)
				g.Expect(err).ToNot(HaveOccurred(), "error getting release")
				g.Expect(release.Config).To(HaveKeyWithValue(PluginOptionDefault, stringVal), "string value not set correctly")
				g.Expect(release.Config).To(HaveKeyWithValue(PluginOptionBool, boolVal), "bool value not set correctly")
				g.Expect(release.Config).To(HaveKeyWithValue(PluginOptionInt, float64(intVal)), "int value not set correctly")
				g.Expect(release.Config).To(HaveKeyWithValue(PluginOptionList, listVal), "list value not set correctly")
				g.Expect(release.Config).To(HaveKeyWithValue(PluginOptionMap, mapVal), "map value not set correctly")
			}).Should(Succeed(), "Helm Release should have the updated plugin option values")
		})
		test.EventuallyDeleted(test.Ctx, test.K8sClient, complexPlugin)
	})

	DescribeTable("creating of Plugins with wrong OptionValues", func(option string, value any) {
		plugin := test.NewPlugin(test.Ctx, "testPlugin", Namespace,
			test.WithPluginDefinition("testPlugin"),
			test.WithReleaseName(ReleaseName),
			test.WithPluginOptionValue(option, test.MustReturnJSONFor(value)))
		Expect(test.K8sClient.Create(test.Ctx, plugin)).Should(Not(Succeed()), "creating a plugin with wrong types should not be successful")
	},
		Entry("string with wrong type", PluginOptionRequired, 1),
		Entry("bool with wrong type", PluginOptionBool, "true"),
		Entry("int with wrong type", PluginOptionInt, "1"),
		Entry("list with wrong type", PluginOptionList, "myListValue1"),
		Entry("map with wrong type", PluginOptionMap, "myMapValue1"),
	)

	// TODO: Revisit after https://github.com/cloudoperators/greenhouse/issues/489 is resolved

	// It("should correctly set PluginFoundCondition if corresponding pluginDefinition was not found", func() {
	// 	By("deleting the pluginDefinition")
	// 	Expect(test.K8sClient.Delete(test.Ctx, testPlugin)).Should(Succeed())
	// 	actPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
	// 	Eventually(func() bool {
	// 		err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
	// 		return apierrors.IsNotFound(err)
	// 	}).Should(BeTrue())

	// 	By("verifying the Plugin was not reconciled")
	// 	actPlugin := &greenhousev1alpha1.Plugin{}
	// 	Eventually(func(g Gomega) bool {
	// 		err := test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
	// 		if err != nil {
	// 			Expect(err).ToNot(HaveOccurred(), "error getting plugin")
	// 			return false
	// 		}
	// 		g.Expect(actPlugin.Status.State).To(Equal(greenhousev1alpha1.PluginStatusError))
	// 		g.Expect(actPlugin.Status.GetConditionByType(greenhousev1alpha1.PluginFoundCondition).IsFalse()).To(BeTrue(), "PluginFoundCondition should be false")
	// 		return true
	// 	}).Should(BeTrue())
	// })

})

var _ = When("the pluginDefinition is UI only", func() {
	var uiPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	var uiPlugin *greenhousev1alpha1.Plugin
	BeforeEach(func() {
		uiPluginDefinition = test.NewClusterPluginDefinition(
			test.Ctx,
			"myuiplugin",
			test.WithVersion("1.0.0"),
			test.WithoutHelmChart(),
			test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
				Name:    "myapp",
				Version: "1.0.0",
				URL:     "http://myapp.com",
			}))
		uiPlugin = test.NewPlugin(test.Ctx, "uiplugin", "default",
			test.WithPluginDefinition("myuiplugin"),
			test.WithReleaseName("myuiplugin-release"))

		Expect(test.K8sClient.Create(test.Ctx, uiPluginDefinition)).Should(Succeed())
		Expect(test.K8sClient.Create(test.Ctx, uiPlugin)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(test.K8sClient.Delete(test.Ctx, uiPlugin)).Should(Succeed())
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Delete(test.Ctx, uiPluginDefinition)).Should(Succeed())
		}).Should(Succeed(), "error deleting uiPluginDefinition")
	})

	It("should skip the helm reconciliation without errors", func() {
		pluginID := types.NamespacedName{Name: "uiplugin", Namespace: "default"}
		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, pluginID, uiPlugin)
			if err != nil {
				return false
			}
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)).ToNot(BeNil())
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition).Status).To(Equal(metav1.ConditionTrue))
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).Status).To(Equal(metav1.ConditionFalse))
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).Message).To(Equal("PluginDefinition is not backed by HelmChart"))
			return true
		}).Should(BeTrue())
	})
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
