// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"errors"
	"testing"

	greenhousecluster "github.com/cloudoperators/greenhouse/internal/controller/cluster"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
	admission "github.com/cloudoperators/greenhouse/internal/webhook"
)

func TestHelmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HelmControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("plugin", (&PluginReconciler{KubeRuntimeOpts: clientutil.RuntimeOptions{QPS: 5, Burst: 10}}).SetupWithManager)
	test.RegisterController("pluginPreset", (&PluginPresetReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)
	test.RegisterWebhook("pluginDefinitionWebhook", admission.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", admission.SetupPluginWebhookWithManager)
	test.RegisterWebhook("clusterWebhook", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", admission.SetupPluginPresetWebhookWithManager)
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
	readyCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
	g.Expect(readyCondition).ToNot(BeNil(), "Ready condition should not be nil")
	clusterAccessReadyCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
	g.Expect(clusterAccessReadyCondition).ToNot(BeNil())
	g.Expect(clusterAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "ClusterAccessReady condition should be true")
	helmReconcileFailedCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
	g.Expect(helmReconcileFailedCondition).ToNot(BeNil())
	g.Expect(helmReconcileFailedCondition.Status).To(Equal(metav1.ConditionFalse), "HelmReconcileFailed condition should be false")
	// helmChartTestSucceededCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmChartTestSucceededCondition)
	// g.Expect(helmChartTestSucceededCondition).ToNot(BeNil())
	// g.Expect(helmChartTestSucceededCondition.Status).To(Equal(metav1.ConditionTrue), "HelmChartTestSucceeded condition should be true")
}

// HelmReconcilerTest performs tests in Serial mode to avoid conflicts with the k8s resources
var _ = Describe("HelmControllerTest", Serial, func() {

	const (
		PluginDefinitionName           = "mytestplugin"
		PluginDefinitionVersion        = "1.0.0"
		PluginDefinitionVersionUpdated = "1.1.0"
		PluginDefinitionChartName      = "myTestpluginChart"
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

		testPluginDefinition *greenhousev1alpha1.PluginDefinition
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

		testPluginDefinition = &greenhousev1alpha1.PluginDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginDefinition",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: PluginDefinitionName,
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Description: "Testplugin",
				Version:     PluginDefinitionVersion,
				HelmChart: &greenhousev1alpha1.HelmChartReference{
					Name:       HelmChart,
					Repository: HelmRepo,
					Version:    PluginDefinitionChartVersion,
				},
				Options: []greenhousev1alpha1.PluginOption{
					{
						Name:        PluginOptionRequired,
						Description: "This is my required test plugin option",
						Required:    true,
						Type:        greenhousev1alpha1.PluginOptionTypeString,
					},
					{
						Name:        PluginOptionOptional,
						Description: "This is my optional test plugin option",
						Required:    false,
						Type:        greenhousev1alpha1.PluginOptionTypeString,
					},
					{
						Name:        PluginOptionDefault,
						Description: "This is my default test plugin option",
						Required:    false,
						Default:     asAPIextensionJSON(PluginOptionDefaultValue),
						Type:        greenhousev1alpha1.PluginOptionTypeString,
					},
				},
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, testPluginDefinition)).Should(Succeed())
		actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		Eventually(func() bool {
			err := test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
			if err != nil {
				return false
			}
			return actPluginDefinition.Spec.Version == PluginDefinitionVersion
		}).Should(BeTrue())

		testPlugin = &greenhousev1alpha1.Plugin{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Plugin",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PluginName,
				Namespace: Namespace,
			},
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: PluginDefinitionName,
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  PluginOptionRequired,
						Value: asAPIextensionJSON(PluginRequiredOptionValue),
					},
				},
			},
		}
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

		// remember original chart loader, which is overwritten in some tests
		tempChartLoader = helm.ChartLoader
	})

	AfterEach(func() {
		err := client.IgnoreNotFound(test.K8sClient.Delete(test.Ctx, testPlugin))
		Expect(err).ToNot(HaveOccurred(), "error deleting plugin")
		actPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() bool {
			return apierrors.IsNotFound(test.K8sClient.Get(test.Ctx, pluginID, actPlugin))
		}).Should(BeTrue())

		err = test.K8sClient.Delete(test.Ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "error deleting pluginDefinition")
		actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		Eventually(func() bool {
			return apierrors.IsNotFound(test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition))
		}).Should(BeTrue())

		// revert to original chart loader
		helm.ChartLoader = tempChartLoader
	})

	When("a pluginDefinition and its chart were updated", func() {
		It("should reconcile the Plugin to a newer PluginDefinition version", func() {

			testPluginDefinition.Spec.HelmChart.Name = HelmChartUpdated
			testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
			Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())

			actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
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
			testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
			Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
			actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
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
			helm.ChartLoader = helm.ChartLoaderFunc(func(name string) (*chart.Chart, error) {
				values := map[string]any{
					"imageTag": UpdatedImageVersion,
				}
				chart, err := loader.Load(name)
				if err != nil {
					return nil, err
				}
				chart.Values = values
				return chart, nil
			})
			By("increasing the pluginDefinition version")
			testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
			Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
			actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
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
			helm.ChartLoader = helm.ChartLoaderFunc(func(name string) (*chart.Chart, error) {
				chart, err := loader.Load(name)
				if err != nil {
					return nil, err
				}
				chart.Metadata.KubeVersion = "<=1.20.0-0"
				return chart, nil
			})

			By("increasing the pluginDefinition version")
			testPluginDefinition.Spec.Version = PluginDefinitionVersionUpdated
			Expect(test.K8sClient.Update(test.Ctx, testPluginDefinition)).Should(Succeed())
			actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
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
				g.Expect(actPlugin.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition).IsTrue()).To(BeFalse(), "ReadyCondition should not be true (but unknown)")
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
		actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, pluginDefinitionID, actPluginDefinition)
		}).Should(Succeed())
		Expect(actPluginDefinition.Spec.Options).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(PluginOptionDefault), "Default": Equal(asAPIextensionJSON(PluginOptionDefaultValue))})))

		actPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, pluginID, actPlugin)
		}).Should(Succeed())

		Expect(actPlugin.Spec.OptionValues).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(PluginOptionDefault), "Value": Equal(asAPIextensionJSON(PluginOptionDefaultValue))})))
	})

	It("should successfully create a Plugin with every type of OptionValue", func() {
		const pluginWithEveryOption = "mytestpluginwitheveryoption"
		var (
			complexPluginDefinition *greenhousev1alpha1.PluginDefinition
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
			complexPluginDefinition = &greenhousev1alpha1.PluginDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PluginDefinition",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginWithEveryOption,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Description: "Test PluginDefinition with all possible Option types",
					Version:     PluginDefinitionVersion,
					HelmChart: &greenhousev1alpha1.HelmChartReference{
						Name:       HelmChartWithAllOptions,
						Repository: HelmRepo,
						Version:    PluginDefinitionChartVersion,
					},
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:        PluginOptionDefault,
							Description: "This is my default test plugin option",
							Required:    false,
							Default:     asAPIextensionJSON(PluginOptionDefaultValue),
							Type:        greenhousev1alpha1.PluginOptionTypeString,
						},
						{
							Name:        PluginOptionBool,
							Description: "This is my default test plugin option with a bool value",
							Required:    false,
							Default:     asAPIextensionJSON(PluginOptionBoolDefault),
							Type:        greenhousev1alpha1.PluginOptionTypeBool,
						},
						{
							Name:        PluginOptionInt,
							Description: "This is my default test plugin option with a int value",
							Required:    false,
							Default:     asAPIextensionJSON(PluginOptionIntDefault),
							Type:        greenhousev1alpha1.PluginOptionTypeInt,
						},
						{
							Name:        PluginOptionList,
							Description: "This is my default test plugin option with a list value",
							Required:    false,
							Default:     asAPIextensionJSON(PluginOptionListDefault),
							Type:        greenhousev1alpha1.PluginOptionTypeList,
						},
						{
							Name:        PluginOptionMap,
							Description: "This is my default test plugin option with a map value",
							Required:    false,
							Default:     asAPIextensionJSON(PluginOptionMapDefault),
							Type:        greenhousev1alpha1.PluginOptionTypeMap,
						},
					},
				},
			}

			Expect(test.K8sClient.Create(test.Ctx, complexPluginDefinition)).Should(Succeed())
			complexPluginDefinitionID := types.NamespacedName{Name: complexPluginDefinition.Name, Namespace: ""}
			actComplexPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, complexPluginDefinitionID, actComplexPluginDefinition)
				if err != nil {
					return false
				}
				return actComplexPluginDefinition.Spec.Version == PluginDefinitionVersion
			}).Should(BeTrue())
		})

		By("creating a Plugin with every type of OptionValue", func() {
			complexPlugin = &greenhousev1alpha1.Plugin{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Plugin",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      pluginName,
					Namespace: Namespace,
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginWithEveryOption,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  PluginOptionDefault,
							Value: asAPIextensionJSON(stringVal),
						},
						{
							Name:  PluginOptionBool,
							Value: asAPIextensionJSON(boolVal),
						},
						{
							Name:  PluginOptionInt,
							Value: asAPIextensionJSON(intVal),
						},
						{
							Name:  PluginOptionList,
							Value: asAPIextensionJSON(listVal),
						},
						{
							Name:  PluginOptionMap,
							Value: asAPIextensionJSON(mapVal),
						},
					},
				},
			}

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
			release, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, clientutil.NewRestClientGetterFromRestConfig(test.Cfg, complexPlugin.Namespace), complexPlugin)
			Expect(err).ToNot(HaveOccurred(), "error getting release")
			Expect(release.Config).To(HaveKeyWithValue(PluginOptionDefault, stringVal), "string value not set correctly")
			Expect(release.Config).To(HaveKeyWithValue(PluginOptionBool, boolVal), "bool value not set correctly")
			Expect(release.Config).To(HaveKeyWithValue(PluginOptionInt, float64(intVal)), "int value not set correctly")
			Expect(release.Config).To(HaveKeyWithValue(PluginOptionList, listVal), "list value not set correctly")
			Expect(release.Config).To(HaveKeyWithValue(PluginOptionMap, mapVal), "map value not set correctly")
		})
	})

	DescribeTable("creating of Plugins with wrong OptionValues", func(option string, value any) {
		plugin := &greenhousev1alpha1.Plugin{
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "testPlugin",
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  option,
						Value: asAPIextensionJSON(value),
					},
				},
			},
		}
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
	// 	actPluginDefinition := &greenhousev1alpha1.PluginDefinition{}
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
	var uiPluginDefinition *greenhousev1alpha1.PluginDefinition
	var uiPlugin *greenhousev1alpha1.Plugin
	BeforeEach(func() {
		uiPluginDefinition = &greenhousev1alpha1.PluginDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginDefinition",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "myuiplugin",
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Description: "Testplugin with UI only",
				Version:     "1.0.0",
				UIApplication: &greenhousev1alpha1.UIApplicationReference{
					Name:    "myapp",
					Version: "1.0.0",
					URL:     "http://myapp.com",
				},
			},
		}
		uiPlugin = &greenhousev1alpha1.Plugin{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Plugin",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "uiplugin",
				Namespace: "default",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "myuiplugin",
			},
		}

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
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)).ToNot(BeNil())
			g.Expect(uiPlugin.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition).Status).To(Equal(metav1.ConditionTrue))
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

// asAPIextensionJSON marshals v into a JSON and returns an apiextensionsv1.JSON object
func asAPIextensionJSON(v any) *apiextensionsv1.JSON {
	bs, err := json.Marshal(v)
	Expect(err).ToNot(HaveOccurred(), "error marshalling value")
	return &apiextensionsv1.JSON{Raw: bs}
}
