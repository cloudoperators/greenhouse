// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	pluginPresetName                       = "test-pluginpreset"
	pluginPresetDefinitionName             = "preset-plugindefinition"
	pluginDefinitionWithDefaultsName       = "plugin-definition-with-defaults"
	pluginDefinitionWithRequiredOptionName = "plugin-definition-with-required-option"

	releaseNamespace = "test-namespace"

	clusterA = "cluster-a"
	clusterB = "cluster-b"

	preventDeletionAnnotation = "greenhouse.sap/prevent-deletion"
)

var (
	clusterAKubeConfig []byte
	clusterAK8sClient  client.Client
	clusterARemote     *envtest.Environment

	clusterBKubeConfig []byte
	clusterBK8sClient  client.Client
	clusterBRemote     *envtest.Environment

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
				Name:       "./../../test/fixtures/chartWithConfigMap",
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
		format.MaxLength = 0
		By("creating a test PluginDefinition")
		err := test.K8sClient.Create(test.Ctx, pluginPresetDefinition)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginDefinition")

		By("bootstrapping the remote cluster")
		_, clusterAK8sClient, clusterARemote, clusterAKubeConfig = test.StartControlPlane("6886", false, false)
		By("bootstrapping the other remote cluster")
		_, clusterBK8sClient, clusterBRemote, clusterBKubeConfig = test.StartControlPlane("6887", false, false)

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the remote cluster")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(clusterAKubeConfig, releaseNamespace, clientutil.WithPersistentConfig())
		remoteK8sClient, err := clientutil.NewK8sClientFromRestClientGetter(remoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = remoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: releaseNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating the namespace on the other remote cluster")
		otherRemoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(clusterBKubeConfig, releaseNamespace, clientutil.WithPersistentConfig())
		otherRemoteK8sClient, err := clientutil.NewK8sClientFromRestClientGetter(otherRemoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = otherRemoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: releaseNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating two test clusters")
		for clusterName, kubeCfg := range map[string][]byte{clusterA: clusterAKubeConfig, clusterB: clusterBKubeConfig} {
			err := test.K8sClient.Create(test.Ctx, cluster(clusterName))
			Expect(err).Should(Succeed(), "failed to create test cluster: "+clusterName)

			By("creating a secret with a valid kubeconfig for a remote cluster")
			secretObj := clusterSecret(clusterName)
			secretObj.Data = map[string][]byte{
				greenhouseapis.KubeConfigKey: kubeCfg,
			}
			Expect(test.K8sClient.Create(test.Ctx, secretObj)).Should(Succeed())
		}
	})

	AfterAll(func() {
		By("Stopping remote environments")
		err := clusterARemote.Stop()
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
		err = clusterBRemote.Stop()
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error stopping the other remote environment")
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

		/*		By("manually creating a Plugin with OwnerReference but cluster not matching the selector")
				pluginNotExp := plugin(clusterB, expPlugin.OwnerReferences)
				Expect(test.K8sClient.Create(test.Ctx, pluginNotExp)).Should(Succeed(), "failed to create test Plugin")

				Eventually(func(g Gomega) error {
					err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: pluginNotExp.Name, Namespace: pluginNotExp.Namespace}, pluginNotExp)
					g.Expect(err).To(HaveOccurred(), "there should be an error getting the Plugin")
					return client.IgnoreNotFound(err)
				}).Should(Succeed(), "the Plugin should be deleted")*/

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

		By("deleting the PluginPreset")
		err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Namespace: testPluginPreset.Namespace, Name: testPluginPreset.Name}, testPluginPreset)
		Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
		testPluginPreset.Annotations = map[string]string{}
		err = test.K8sClient.Update(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred())
		Expect(test.K8sClient.Delete(test.Ctx, testPluginPreset)).Should(Succeed(), "failed to delete test PluginPreset")
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
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(pluginPreset), pluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			pluginPreset.Annotations = map[string]string{}
			Expect(test.K8sClient.Update(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		}).Should(Succeed(), "failed to update PluginPreset")
		Expect(test.K8sClient.Delete(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should reconcile a PluginPreset on cluster changes", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("making clusterB match the clusterSelector")
		pluginList := &greenhousev1alpha1.PluginList{}
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "there should be a Plugin created for the Preset")

		cluster := greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterB,
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

		By("deleting clusterB to ensure the Plugin is deleted")
		test.MustDeleteCluster(test.Ctx, test.K8sClient, client.ObjectKeyFromObject(&cluster))
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.InNamespace(cluster.GetNamespace()), client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "the PluginPreset should have removed the Plugin for the deleted Cluster")

		By("removing the PluginPreset")
		err = test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to get PluginPreset")
		// Remove prevent-deletion annotation before deleting PluginPreset.
		_, err = clientutil.Patch(test.Ctx, test.K8sClient, testPluginPreset, func() error {
			delete(testPluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "failed to patch PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should delete a Plugin if the cluster no longer matches", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		err = test.K8sClient.Create(test.Ctx, cluster(clusterB))
		Expect(err).ToNot(HaveOccurred(), "failed to create test cluster: "+clusterB)
		secretObj := clusterSecret(clusterB)
		secretObj.Data = map[string][]byte{
			greenhouseapis.KubeConfigKey: clusterBKubeConfig,
		}
		Expect(test.K8sClient.Update(test.Ctx, secretObj)).Should(Succeed())

		By("making clusterB match the clusterSelector")
		pluginList := &greenhousev1alpha1.PluginList{}
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "there should be a Plugin created for the Preset")

		cluster := greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterB,
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

		By("changing clusterB labels to ensure the Plugin is deleted")
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, &cluster, func() error {
			cluster.Labels = map[string]string{}
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Cluster labels")

		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.InNamespace(cluster.GetNamespace()), client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "the PluginPreset should have removed the Plugin for the deleted Cluster")

		By("removing the PluginPreset")
		err = test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to get PluginPreset")
		// Remove prevent-deletion annotation before deleting PluginPreset.
		_, err = clientutil.Patch(test.Ctx, test.K8sClient, testPluginPreset, func() error {
			delete(testPluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "failed to patch PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should set the Status NotReady if ClusterSelector does not match", func() {
		// Create a PluginPreset with a ClusterSelector that does not match any cluster
		pluginPreset := pluginPreset("not-ready", "non-existing-cluster")
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).Should(Succeed(), "failed to create test PluginPreset")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "not-ready", Namespace: pluginPreset.Namespace}, pluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			g.Expect(pluginPreset.Status.StatusConditions.Conditions).NotTo(BeNil(), "the PluginPreset should have a StatusConditions")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ClusterListEmpty).IsTrue()).Should(BeTrue(), "PluginPreset should have the ClusterListEmptyCondition set to true")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition).IsFalse()).Should(BeTrue(), "PluginPreset should have the ReadyCondition set to false")
		}).Should(Succeed(), "the PluginPreset should be reconciled")
	})

	It("should reconcile PluginStatuses for PluginPreset", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("checking PluginStatuses in the PluginPreset")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the PluginPreset")
			g.Expect(testPluginPreset.Status.PluginStatuses).To(HaveLen(1), "there should be exactly one plugin status")
			managedPluginStatus := testPluginPreset.Status.PluginStatuses[0]
			expectedPluginName := testPluginPreset.Name + "-" + clusterA
			g.Expect(managedPluginStatus.PluginName).To(Equal(expectedPluginName), "managed plugin status should have the correct PluginName set")
			g.Expect(managedPluginStatus.ReadyCondition.IsTrue()).To(BeTrue(), "reported Ready condition for managed plugin should be set to true")
			g.Expect(testPluginPreset.Status.AvailablePlugins).To(Equal(1), "PluginPreset Status should show exactly one available plugin")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(1), "PluginPreset Status should show exactly one ready plugin")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
		}).Should(Succeed())

		By("making clusterB match the clusterSelector")
		cluster := greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterB,
				Namespace: test.TestNamespace,
			},
		}
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, &cluster, func() error {
			cluster.Labels = map[string]string{"cluster": clusterA}
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update other Cluster labels")

		By("checking that there are two Plugins")
		pluginList := &greenhousev1alpha1.PluginList{}
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).ToNot(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(2), "there should be exactly two Plugins managed by the PluginPreset")
		}).Should(Succeed(), "the PluginPreset should have noticed the ClusterLabel change")

		By("checking that the additional Plugin is reported in PluginStatuses of the PluginPreset")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the PluginPreset")

			g.Expect(testPluginPreset.Status.PluginStatuses).To(HaveLen(2), "there should be exactly two plugin statuses")
			g.Expect(slices.ContainsFunc(testPluginPreset.Status.PluginStatuses, func(status greenhousev1alpha1.ManagedPluginStatus) bool {
				return status.PluginName == testPluginPreset.Name+"-"+clusterA && status.ReadyCondition.IsTrue()
			})).To(BeTrue(), "Ready true status should be reported for the first plugin")
			g.Expect(slices.ContainsFunc(testPluginPreset.Status.PluginStatuses, func(status greenhousev1alpha1.ManagedPluginStatus) bool {
				return status.PluginName == testPluginPreset.Name+"-"+clusterB && status.ReadyCondition.IsTrue()
			})).To(BeTrue(), "Ready true status should be reported for the additional plugin")
			g.Expect(testPluginPreset.Status.AvailablePlugins).To(Equal(2), "PluginPreset Status should show exactly two available plugins")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(2), "PluginPreset Status should show exactly two ready plugins")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
		}).Should(Succeed())

		By("deleting otherTestCluster to ensure the Plugin is deleted")
		test.MustDeleteCluster(test.Ctx, test.K8sClient, client.ObjectKeyFromObject(&cluster))
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.InNamespace(cluster.GetNamespace()), client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "the PluginPreset should have removed the Plugin for the deleted Cluster")

		By("checking that the Plugin removal is reflected in the PluginStatuses of the PluginPreset")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the PluginPreset")
			g.Expect(testPluginPreset.Status.PluginStatuses).To(HaveLen(1), "there should be exactly one plugin status")
			managedPluginStatus := testPluginPreset.Status.PluginStatuses[0]
			expectedPluginName := testPluginPreset.Name + "-" + clusterA
			g.Expect(managedPluginStatus.PluginName).To(Equal(expectedPluginName), "managed plugin status should have the correct PluginName set")
			g.Expect(managedPluginStatus.ReadyCondition.IsTrue()).To(BeTrue(), "reported Ready condition for managed plugin should be set to true")
			g.Expect(testPluginPreset.Status.AvailablePlugins).To(Equal(1), "PluginPreset Status should show exactly one available plugin")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(1), "PluginPreset Status should show exactly one ready plugin")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
		}).Should(Succeed())

		By("removing the PluginPreset")
		err = test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to get PluginPreset")
		// Remove prevent-deletion annotation before deleting PluginPreset.
		_, err = clientutil.Patch(test.Ctx, test.K8sClient, testPluginPreset, func() error {
			delete(testPluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "failed to patch PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should create a Plugin with required options taken from PluginPreset overrides", func() {
		By("creating PluginDefinition with required option values")
		pluginDefinition := pluginDefinitionWithRequiredOption()
		Expect(test.K8sClient.Create(test.Ctx, pluginDefinition)).To(Succeed(), "failed to create PluginDefinition")
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginDefinition)

		By("creating a PluginPreset with overrides")
		pluginPreset := pluginPreset(pluginPresetName+"-override1", clusterA)
		pluginPreset.Spec.Plugin.PluginDefinition = pluginDefinition.Name
		pluginPreset.Spec.ClusterOptionOverrides = append(pluginPreset.Spec.ClusterOptionOverrides, greenhousev1alpha1.ClusterOptionOverride{
			ClusterName: clusterA,
			Overrides: []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "test-required-option-1",
					Value: test.MustReturnJSONFor(5),
				},
			},
		})
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed(), "failed to create PluginPreset")
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("checking that Plugin has been created with overridden required option")
		pluginObjectKey := types.NamespacedName{Name: pluginPresetName + "-override1-" + clusterA, Namespace: test.TestNamespace}
		plugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, pluginObjectKey, plugin)
		}).Should(Succeed(), "the Plugin should be created successfully")
		Expect(plugin.Spec.OptionValues).To(ContainElement(pluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0]),
			"ClusterOptionOverrides should be applied to the Plugin OptionValues")

		By("removing plugin preset")
		err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(pluginPreset), pluginPreset)
		Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
		_, err = clientutil.Patch(test.Ctx, test.K8sClient, pluginPreset, func() error {
			delete(pluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "failed to remove prevent-deletion annotation from PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should save an error when Plugin creation failed due to required options being unset", func() {
		By("creating a PluginPreset based on PluginDefinition with required option")
		pluginPreset := pluginPreset(pluginPresetName+"-missing1", clusterA)
		pluginPreset.Spec.Plugin.PluginDefinition = pluginDefinitionWithRequiredOptionName
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed(), "failed to create PluginPreset")
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("checking that Plugin creation error has been saved to PluginPreset")
		Eventually(func(g Gomega) {
			presetObjectKey := types.NamespacedName{Name: pluginPresetName + "-missing1", Namespace: test.TestNamespace}
			g.Expect(test.K8sClient.Get(test.Ctx, presetObjectKey, pluginPreset)).To(Succeed())
			pluginFailedCondition := pluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
			g.Expect(pluginFailedCondition).ToNot(BeNil())
			g.Expect(pluginFailedCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(pluginFailedCondition.Reason).To(Equal(greenhousev1alpha1.PluginReconcileFailed))
			expectedPluginName := pluginPresetName + "-missing1-" + clusterA
			g.Expect(pluginFailedCondition.Message).To(ContainSubstring(expectedPluginName + ": Required value: Option 'test-required-option-1' is required by PluginDefinition 'plugin-definition-with-required-option'"))
		}).Should(Succeed(), "Plugin creation error not found in PluginPreset")

		By("removing plugin preset")
		err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(pluginPreset), pluginPreset)
		Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
		_, err = clientutil.Patch(test.Ctx, test.K8sClient, pluginPreset, func() error {
			delete(pluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "failed to remove prevent-deletion annotation from PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})
})

var _ = Describe("Plugin Preset skip changes", Ordered, func() {
	DescribeTable("",
		func(testPlugin *greenhousev1alpha1.Plugin, testPresetPlugin *greenhousev1alpha1.PluginPreset, testPluginDefinition *greenhousev1alpha1.PluginDefinition, clusterName string, expected bool) {
			Expect(shouldSkipPlugin(testPlugin, testPresetPlugin, testPluginDefinition, clusterName)).To(BeEquivalentTo(expected))
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
			"",
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
			"",
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
			"",
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
			"",
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
			"",
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
			"",
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
			"",
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
			"",
			false,
		), Entry("should not skip when Plugin has different valueFrom then PluginDefinition",
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
							Name: "plugin_definition.test_parameter",
							ValueFrom: &greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "test-secret",
									Key:  "test-key",
								},
							},
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
			"",
			false,
		), Entry("should skip when Plugin has same valueFrom as PluginDefinition",
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
							Name: "plugin_definition.test_parameter",
							ValueFrom: &greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "test-secret",
									Key:  "test-key",
								},
							},
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
							{Name: "plugin_definition.test_parameter",
								ValueFrom: &greenhousev1alpha1.ValueFromSource{
									Secret: &greenhousev1alpha1.SecretKeyReference{
										Name: "test-secret",
										Key:  "test-key",
									},
								},
							},
						},
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
			"",
			true,
		), Entry("should not skip when Plugin has different value then plugin override",
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
								Name:  "global.greenhouse.test_parameter",
								Value: asAPIextensionJSON(2),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "global.greenhouse.test_parameter",
									Value: asAPIextensionJSON(3),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			clusterA,
			false,
		), Entry("should skip when Plugin has different value then plugin override but cluster name is different",
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
								Name:  "global.greenhouse.test_parameter",
								Value: asAPIextensionJSON(2),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "global.greenhouse.test_parameter",
									Value: asAPIextensionJSON(3),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.PluginDefinition{},
			clusterB,
			true,
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
			Name:      clusterName,
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

func pluginDefinitionWithRequiredOption() *greenhousev1alpha1.PluginDefinition {
	return &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginDefinitionWithRequiredOptionName,
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			DisplayName: "test display name",
			Version:     "1.0.0",
			Options: []greenhousev1alpha1.PluginOption{
				{
					Name:     "test-required-option-1",
					Type:     "int",
					Required: true,
				},
			},
			UIApplication: &greenhousev1alpha1.UIApplicationReference{
				Name:    "test-ui-app",
				Version: "0.0.1",
			},
		},
	}
}
