// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	pluginPresetName                 = "test-pluginpreset"
	pluginPresetDefinitionName       = "preset-plugindefinition"
	pluginDefinitionWithDefaultsName = "plugin-definition-with-defaults"

	releaseNamespace = "test-namespace"

	clusterA = "cluster-a"
	clusterB = "cluster-b"
	clusterC = "cluster-c"
)

var (
	pluginPresetRemoteKubeConfig []byte
	pluginPresetK8sClient        client.Client
	pluginPresetRemote           *envtest.Environment

	pluginPresetDefinition = &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginPresetDefinitionName,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Description: "Testplugin",
			Version:     "1.0.0",
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:       "./../../test/fixtures/myChart",
				Repository: "dummy",
				Version:    "1.0.0",
			},
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:        "myRequiredOption",
					Description: "This is my required test plugin option",
					Required:    true,
					Type:        greenhousev1alpha1.PluginOptionTypeString,
				},
			},
		},
	}
)

var _ = Describe("PluginPreset Controller Lifecycle", Ordered, func() {
	BeforeAll(func() {
		By("creating a test PluginDefinition")
		err := test.K8sClient.Create(test.Ctx, pluginPresetDefinition)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginDefinition")

		By("bootstrapping the remote cluster")
		_, pluginPresetK8sClient, pluginPresetRemote, pluginPresetRemoteKubeConfig = test.StartControlPlane("6886", false, false)

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the cluster")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(pluginPresetRemoteKubeConfig, releaseNamespace, clientutil.WithPersistentConfig())
		remoteK8sClient, err := clientutil.NewK8sClientFromRestClientGetter(remoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = remoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: releaseNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating two test clusters for the same remote environment")
		for _, clusterName := range []string{clusterA, clusterB} {
			err := test.K8sClient.Create(test.Ctx, cluster(clusterName))
			Expect(err).Should(Succeed(), "failed to create test cluster: "+clusterName)

			By("creating a secret with a valid kubeconfig for a remote cluster")
			secretObj := clusterSecret(clusterName)
			secretObj.Data = map[string][]byte{
				greenhouseapis.KubeConfigKey: pluginPresetRemoteKubeConfig,
			}
			Expect(test.K8sClient.Create(test.Ctx, secretObj)).Should(Succeed())
		}
	})

	AfterAll(func() {
		err := pluginPresetRemote.Stop()
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
	})

	It("should reconcile a PluginPreset", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		Expect(expPlugin.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginPreset, pluginPresetName), "the Plugin should be labeled as managed by the PluginPreset")

		By("modifying the Plugin and ensuring it is reconciled")
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, expPlugin, func() error {
			// add a new option value that is not specified by the PluginPreset
			opt := greenhousev1alpha1.PluginOptionValue{Name: "option1", Value: test.MustReturnJSONFor("value1")}
			expPlugin.Spec.OptionValues = append(expPlugin.Spec.OptionValues, opt)
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Plugin")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Plugin")
			g.Expect(expPlugin.Spec.OptionValues).ToNot(ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "option1", Value: test.MustReturnJSONFor("value1")}), "the Plugin should be reconciled")
		}).Should(Succeed(), "the Plugin should be reconciled")

		By("manually creating a Plugin with OwnerReference but cluster not matching the selector")
		pluginNotExp := plugin(clusterB, expPlugin.OwnerReferences)
		Expect(test.K8sClient.Create(test.Ctx, pluginNotExp)).Should(Succeed(), "failed to create test Plugin")

		Eventually(func(g Gomega) error {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: pluginNotExp.Name, Namespace: pluginNotExp.Namespace}, pluginNotExp)
			g.Expect(err).To(HaveOccurred(), "there should be an error getting the Plugin")
			return client.IgnoreNotFound(err)
		}).Should(Succeed(), "the Plugin should be deleted")

		By("removing the preset label from the Plugin")
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, expPlugin, func() error {
			delete(expPlugin.Labels, greenhouseapis.LabelKeyPluginPreset)
			return controllerutil.RemoveControllerReference(testPluginPreset, expPlugin, test.K8sClient.Scheme())
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Plugin")

		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Namespace: testPluginPreset.Namespace, Name: testPluginPreset.Name}, testPluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			return testPluginPreset.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.PluginSkippedCondition).IsTrue()
		}).Should(BeTrue(), "PluginPreset should have the SkippedCondition set to true")

		By("re-adding the preset label to the Plugin")
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, expPlugin, func() error {
			expPlugin.Labels[greenhouseapis.LabelKeyPluginPreset] = testPluginPreset.Name
			return controllerutil.SetControllerReference(testPluginPreset, expPlugin, test.K8sClient.Scheme())
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Plugin")

		By("deleting the PluginPreset to ensure all Plugins are deleted")
		Expect(test.K8sClient.Delete(test.Ctx, testPluginPreset)).Should(Succeed(), "failed to delete test PluginPreset")
		Eventually(func(g Gomega) error {
			err := test.K8sClient.Get(test.Ctx, expPluginName, pluginNotExp)
			g.Expect(err).To(HaveOccurred(), "there should be an error getting the Plugin")
			return client.IgnoreNotFound(err)
		}).Should(Succeed(), "the Plugin should be deleted")
	})

	It("should reconcile a PluginPreset with plugin definition defaults", func() {
		By("ensuring a Plugin Definition has been created")
		pluginDefinition := pluginDefinitionWithDefaults()
		Expect(test.K8sClient.Create(test.Ctx, pluginDefinition)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginDefinition)

		By("ensuring a Plugin Preset has been created")
		pluginPreset := pluginPreset(pluginPresetName+"-2", clusterA)
		pluginPreset.Spec.Plugin.PluginDefinition = pluginDefinition.Name
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-2-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		By("checking plugin options with plugin definition defaults and plugin preset values")
		Expect(expPlugin.Spec.OptionValues).To(ContainElement(pluginPreset.Spec.Plugin.OptionValues[0]))
		Expect(expPlugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
			Name:  pluginDefinition.Spec.Options[0].Name,
			Value: pluginDefinition.Spec.Options[0].Default,
		}))

		By("removing plugin preset")
		Expect(test.K8sClient.Delete(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should reconcile a PluginPreset on cluster changes", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("onboarding another cluster")
		err = test.K8sClient.Create(test.Ctx, cluster(clusterC))
		Expect(err).Should(Succeed(), "failed to create test cluster: "+clusterC)
		secretObj := clusterSecret(clusterC)
		secretObj.Data = map[string][]byte{
			greenhouseapis.KubeConfigKey: pluginPresetRemoteKubeConfig,
		}
		Expect(test.K8sClient.Create(test.Ctx, secretObj)).Should(Succeed())

		By("making clusterC match the clusterSelector")
		pluginList := &greenhousev1alpha1.PluginList{}
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "there should be a Plugin created for the Preset")

		cluster := greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterC,
				Namespace: test.TestNamespace,
			},
		}
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, &cluster, func() error {
			cluster.Labels = map[string]string{"cluster": clusterA}
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Cluster labels")
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(2), "there should be two Plugins")
		}).Should(Succeed(), "the PluginPreset should have noticed the ClusterLabel change")

		By("deleting clusterC to ensure the Plugin is deleted")
		Expect(test.K8sClient.Delete(test.Ctx, &cluster)).To(Succeed(), "failed to delete ClusterC")

		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "the PluginPreset should have removed the Plugin for the deleted Cluster")

		By("deleting the PluginPreset to ensure all Plugins are deleted")
		Expect(test.K8sClient.Delete(test.Ctx, testPluginPreset)).Should(Succeed(), "failed to delete test PluginPreset")
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(BeEmpty(), "all plugins for the Preset should be deleted")
		}).Should(Succeed(), "plugins for the pluginPreset should be deleted")
	})

	It("should set the Status NotReady if ClusterSelector does not match", func() {
		// Create a PluginPreset with a ClusterSelector that does not match any cluster
		pluginPreset := pluginPreset("not-ready", "non-existing-cluster")
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).Should(Succeed(), "failed to create test PluginPreset")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "not-ready", Namespace: pluginPreset.Namespace}, pluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			g.Expect(pluginPreset.Status.StatusConditions.Conditions).NotTo(BeNil(), "the PluginPreset should have a StatusConditions")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ClusterListEmpty).IsTrue()).Should(BeTrue(), "PluginPreset should have the ClusterListEmptyCondition set to true")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ReadyCondition).IsFalse()).Should(BeTrue(), "PluginPreset should have the ReadyCondition set to false")
		}).Should(Succeed(), "the PluginPreset should be reconciled")
	})
})

var _ = Describe("Plugin Preset skip changes", Ordered, func() {
	DescribeTable("",
		func(testPlugin *greenhousev1alpha1.Plugin, testPresetPlugin *greenhousev1alpha1.PluginPreset, testPluginDefinition *greenhousev1alpha1.PluginDefinition, expected bool) {
			Expect(shouldSkipPlugin(testPlugin, testPresetPlugin, testPluginDefinition)).To(BeEquivalentTo(expected))
		},
		Entry("should skip when plugin preset name in plugin's labels is different then defined name in plugin preset",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName + "A",
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			true,
		),
		Entry("should not skip when plugin preset contains options which is not present in plugin",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "plugin_preset.test_parameter",
								Value: asAPIextensionJSON(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			false,
		),
		Entry("should not skip when plugin preset has option with different value",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_preset.test_parameter",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "plugin_preset.test_parameter",
								Value: asAPIextensionJSON(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			false,
		),
		Entry("should not skip when one of plugin preset option has more then one value and one of them is different then option in plugin",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_preset.test_parameter_1",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "plugin_preset.test_parameter_2",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_preset.test_parameter_4",
							Value: asAPIextensionJSON(3),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "plugin_preset.test_parameter_1",
								Value: asAPIextensionJSON(1),
							},
							{
								Name:  "plugin_preset.test_parameter_2",
								Value: asAPIextensionJSON(2),
							},
							{
								Name:  "plugin_preset.test_parameter_4",
								Value: asAPIextensionJSON(4),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			false,
		),
		Entry("should skip when plugin preset has the same values like plugin",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_preset.test_parameter",
							Value: asAPIextensionJSON(3),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "plugin_preset.test_parameter",
								Value: asAPIextensionJSON(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			true,
		),
		Entry("should not skip when plugin has custom values",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_preset.test_parameter",
							Value: asAPIextensionJSON(3),
						},
						{
							Name:  "custom_parameter",
							Value: asAPIextensionJSON(123),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "plugin_preset.test_parameter",
								Value: asAPIextensionJSON(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			false,
		),
		Entry("should skip when plugin has default values from plugin definition",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_definition.test_parameter",
							Value: asAPIextensionJSON(3),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues:     []greenhousev1alpha1.PluginOptionValue{},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: asAPIextensionJSON(3),
						},
					},
				},
			},
			true,
		),
		Entry("should not skip when plugin has different values then plugin definition",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: pluginPresetDefinitionName,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "global.greenhouse.test_parameter",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "plugin_definition.test_parameter",
							Value: asAPIextensionJSON(3),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginSpec{
						PluginDefinition: pluginPresetDefinitionName,
						OptionValues:     []greenhousev1alpha1.PluginOptionValue{},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: asAPIextensionJSON(4),
						},
					},
				},
			},
			false,
		),
	)
})

var _ = Describe("overridesPluginOptionValues", Ordered, func() {
	DescribeTable("test cases", func(plugin *greenhousev1alpha1.Plugin, preset *greenhousev1alpha1.PluginPreset, expectedPlugin *greenhousev1alpha1.Plugin) {
		overridesPluginOptionValues(plugin, preset)
		Expect(plugin).To(BeEquivalentTo(expectedPlugin))
	},
		Entry("with no defined pluginPresetOverrides",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
		),
		Entry("with defined pluginPresetOverrides but for another cluster",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterB,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: asAPIextensionJSON(1),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
		),
		Entry("with defined pluginPresetOverrides for the correct cluster",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: asAPIextensionJSON(1),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
					},
				},
			},
		),
		Entry("with defined pluginPresetOverrides for the cluster and plugin with empty option values",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName:  clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: asAPIextensionJSON(1),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
					},
				},
			},
		),
		Entry("with defined pluginPresetOverrides and plugin has two options",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "option-2",
							Value: asAPIextensionJSON(1),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-2",
									Value: asAPIextensionJSON(2),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "option-2",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
		),
		Entry("with defined pluginPresetOverrides has multiple options to override",
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "option-2",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "option-3",
							Value: asAPIextensionJSON(1),
						},
					},
				},
			},
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-2",
									Value: asAPIextensionJSON(2),
								},
								{
									Name:  "option-3",
									Value: asAPIextensionJSON(2),
								},
								{
									Name:  "option-4",
									Value: asAPIextensionJSON(2),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.Plugin{
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: clusterA,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "option-1",
							Value: asAPIextensionJSON(1),
						},
						{
							Name:  "option-2",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "option-3",
							Value: asAPIextensionJSON(2),
						},
						{
							Name:  "option-4",
							Value: asAPIextensionJSON(2),
						},
					},
				},
			},
		),
	)
})

// clusterSecret returns the secret for a cluster.
func clusterSecret(clusterName string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-" + clusterName,
			Namespace: test.TestNamespace,
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
	}
}

// cluster returns a cluster object with the given name.
func cluster(name string) *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				"cluster": name,
				"foo":     "bar",
			},
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}
}

func plugin(clusterName string, ownerRefs []metav1.OwnerReference) *greenhousev1alpha1.Plugin {
	return &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginPresetName + "-" + clusterName,
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
			},
			OwnerReferences: ownerRefs,
		},
		Spec: greenhousev1alpha1.PluginSpec{
			ClusterName:      clusterB,
			PluginDefinition: pluginPresetDefinitionName,
			OptionValues: []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
			},
		},
	}
}

func pluginPreset(name, selectorValue string) *greenhousev1alpha1.PluginPreset {
	return &greenhousev1alpha1.PluginPreset{
		TypeMeta: metav1.TypeMeta{
			Kind:       greenhousev1alpha1.PluginPresetKind,
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.PluginPresetSpec{
			Plugin: greenhousev1alpha1.PluginSpec{
				PluginDefinition: pluginPresetDefinitionName,
				ReleaseNamespace: releaseNamespace,
				OptionValues: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "myRequiredOption",
						Value: test.MustReturnJSONFor("myValue"),
					},
				},
			},
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": selectorValue,
				},
			},
			ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{},
		},
	}
}

func pluginDefinitionWithDefaults() *greenhousev1alpha1.PluginDefinition {
	return &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginDefinitionWithDefaultsName,
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			DisplayName: "test display name",
			Version:     "1.0.0",
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:    "test-plugin-definition-option-1",
					Type:    "int",
					Default: &apiextensionsv1.JSON{Raw: []byte("1")},
				},
			},
			UIApplication: &greenhousev1alpha1.UIApplicationReference{
				Name:    "test-ui-app",
				Version: "0.0.1",
			},
		},
	}
}
