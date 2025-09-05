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
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	pluginPresetName                       = "test-pluginpreset"
	pluginPresetDefinitionName             = "preset-plugindefinition"
	pluginDefinitionWithDefaultsName       = "plugin-definition-with-defaults"
	pluginDefinitionWithRequiredOptionName = "plugin-definition-with-required-option"

	releaseName      = "test-release"
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

	testTeam               = test.NewTeam(test.Ctx, "test-pluginpreset-team", test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	pluginPresetDefinition = test.NewClusterPluginDefinition(test.Ctx, pluginPresetDefinitionName, test.WithHelmChart(
		&greenhousev1alpha1.HelmChartReference{
			Name:       "./../../test/fixtures/chartWithConfigMap",
			Repository: "dummy",
			Version:    "1.0.0",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:        "myRequiredOption",
			Description: "This is my required test plugin option",
			Required:    true,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}))
)

var _ = Describe("PluginPreset Controller Lifecycle", Ordered, func() {
	var defaultPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
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

		By("creating the test Team")
		Expect(test.K8sClient.Create(test.Ctx, testTeam)).To(Succeed(), "there should be no error creating a test Team")

		By("creating two test clusters")
		for clusterName, kubeCfg := range map[string][]byte{clusterA: clusterAKubeConfig, clusterB: clusterBKubeConfig} {
			err := test.K8sClient.Create(test.Ctx, cluster(clusterName, testTeam.Name))
			Expect(err).Should(Succeed(), "failed to create test cluster: "+clusterName)

			By("creating a secret with a valid kubeconfig for a remote cluster")
			secretObj := clusterSecret(clusterName, testTeam.Name)
			secretObj.Data = map[string][]byte{
				greenhouseapis.KubeConfigKey: kubeCfg,
			}
			Expect(test.K8sClient.Create(test.Ctx, secretObj)).Should(Succeed())
		}

		By("creating PluginDefinition with default options")
		defaultPluginDefinition = test.NewClusterPluginDefinition(test.Ctx, pluginDefinitionWithDefaultsName, test.AppendPluginOption(
			greenhousev1alpha1.PluginOption{
				Name:    "test-plugin-definition-option-1",
				Type:    "int",
				Default: &apiextensionsv1.JSON{Raw: []byte("1")}},
		),
			test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
				Name:    "test-ui-app",
				Version: "0.0.1",
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, defaultPluginDefinition)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, defaultPluginDefinition)
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testTeam)

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
		testPluginPreset := pluginPreset(pluginPresetName, clusterA, testTeam.Name)
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
			expLabels := expPlugin.GetLabels()
			expLabels["foo"] = "bar"
			expPlugin.SetLabels(expLabels)
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Plugin")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Plugin")
			g.Expect(expPlugin.Spec.OptionValues).ToNot(ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "option1", Value: test.MustReturnJSONFor("value1")}), "the Plugin should be reconciled")
			g.Expect(expPlugin.Labels).To(HaveKeyWithValue("foo", "bar"), "the Plugin should keep manual label changes")
		}).Should(Succeed(), "the Plugin should be reconciled")

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
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should reconcile a PluginPreset with plugin definition defaults", func() {

		By("ensuring a Plugin Preset has been created")
		pluginPreset := pluginPreset(pluginPresetName+"-2", clusterA, testTeam.Name)
		pluginPreset.Spec.Plugin.PluginDefinition = pluginDefinitionWithDefaultsName
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
			Name:  defaultPluginDefinition.Spec.Options[0].Name,
			Value: defaultPluginDefinition.Spec.Options[0].Default,
		}))

		By("removing plugin preset")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(pluginPreset), pluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			pluginPreset.Annotations = map[string]string{}
			Expect(test.K8sClient.Update(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		}).Should(Succeed(), "failed to update PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should reconcile a PluginPreset on cluster changes", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA, testTeam.Name)
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
		testPluginPreset := pluginPreset(pluginPresetName, clusterA, testTeam.Name)
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		err = test.K8sClient.Create(test.Ctx, cluster(clusterB, testTeam.Name))
		Expect(err).ToNot(HaveOccurred(), "failed to create test cluster: "+clusterB)
		secretObj := clusterSecret(clusterB, testTeam.Name)
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
		pluginPreset := pluginPreset("not-ready", "non-existing-cluster", testTeam.Name)
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).Should(Succeed(), "failed to create test PluginPreset")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "not-ready", Namespace: pluginPreset.Namespace}, pluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting PluginPreset")
			g.Expect(pluginPreset.Status.StatusConditions.Conditions).NotTo(BeNil(), "the PluginPreset should have a StatusConditions")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ClusterListEmpty).IsTrue()).Should(BeTrue(), "PluginPreset should have the ClusterListEmptyCondition set to true")
			g.Expect(pluginPreset.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition).IsFalse()).Should(BeTrue(), "PluginPreset should have the ReadyCondition set to false")
			g.Expect(pluginPreset.Status.GetConditionByType(greenhousev1alpha1.AllPluginsReadyCondition).IsFalse()).Should(BeTrue(), "AllPluginsReadyCondition should be set to false")
		}).Should(Succeed(), "the PluginPreset should be reconciled")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should reconcile PluginStatuses for PluginPreset", func() {
		By("creating a PluginPreset")
		testPluginPreset := pluginPreset(pluginPresetName, clusterA, testTeam.Name)
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
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(1), "PluginPreset Status should show exactly one plugin in total")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(1), "PluginPreset Status should show exactly one ready plugin")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
			g.Expect(testPluginPreset.Status.GetConditionByType(greenhousev1alpha1.AllPluginsReadyCondition).IsTrue()).To(BeTrue(), "AllPluginsReadyCondition should be true when all plugins are ready")
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
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(2), "PluginPreset Status should show exactly two plugins in total")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(2), "PluginPreset Status should show exactly two ready plugins")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
			g.Expect(testPluginPreset.Status.GetConditionByType(greenhousev1alpha1.AllPluginsReadyCondition).IsTrue()).To(BeTrue(), "AllPluginsReadyCondition should be true when all plugins are ready")
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
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(1), "PluginPreset Status should show exactly one plugin in total")
			g.Expect(testPluginPreset.Status.ReadyPlugins).To(Equal(1), "PluginPreset Status should show exactly one ready plugin")
			g.Expect(testPluginPreset.Status.FailedPlugins).To(Equal(0), "PluginPreset Status should show exactly zero failed plugins")
			g.Expect(testPluginPreset.Status.GetConditionByType(greenhousev1alpha1.AllPluginsReadyCondition).IsTrue()).To(BeTrue(), "AllPluginsReadyCondition should be true when all plugins are ready")
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
		pluginDefinition := test.NewClusterPluginDefinition(test.Ctx, pluginDefinitionWithRequiredOptionName,
			test.AppendPluginOption(
				greenhousev1alpha1.PluginOption{
					Name:     "test-required-option-1",
					Type:     "int",
					Required: true}),
			test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
				Name:    "test-ui-app",
				Version: "0.0.1"}),
		)
		Expect(test.K8sClient.Create(test.Ctx, pluginDefinition)).To(Succeed(), "failed to create PluginDefinition")
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginDefinition)

		By("creating a PluginPreset with overrides")
		pluginPreset := pluginPreset(pluginPresetName+"-override1", clusterA, testTeam.Name)
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
		pluginPreset := pluginPreset(pluginPresetName+"-missing1", clusterA, testTeam.Name)
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

	It("should successfully propagate labels from PluginPreset to Plugin", func() {
		By("ensuring a Plugin Preset has been created")
		pluginPreset := pluginPreset(pluginPresetName+"-label-propagation", clusterA, testTeam.Name)
		pluginPreset.Spec.Plugin.PluginDefinition = pluginDefinitionWithDefaultsName
		pluginPreset.SetAnnotations(map[string]string{
			lifecycle.PropagateLabelsAnnotation: "support_group, region",
		})
		pluginPreset.SetLabels(map[string]string{
			"support_group": "foo",
			"region":        "bar",
			"cluster_type":  "operations",
		})
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-label-propagation-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		By("checking Plugin has propagated labels from PluginPreset")
		Expect(expPlugin.Labels).To(HaveKey("support_group"), "the plugin should have the support_group propagated label")
		Expect(expPlugin.Labels).To(HaveKey("region"), "the plugin should have the region propagated label")

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
})

var _ = Describe("Plugin Preset skip changes", Ordered, func() {
	DescribeTable("",
		func(testPlugin *greenhousev1alpha1.Plugin, testPresetPlugin *greenhousev1alpha1.PluginPreset, testPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition, clusterName string, expected bool) {
			Expect(shouldSkipPlugin(testPlugin, testPresetPlugin, testPluginDefinition, clusterName)).To(BeEquivalentTo(expected))
		},
		Entry("should skip when plugin preset name in plugin's labels is different then defined name in plugin preset",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName+"A"),
			),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetName,
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			true,
		),
		Entry("should not skip when plugin preset contains options which is not present in plugin",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
			),
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
								Value: test.MustReturnJSONFor(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			false,
		),
		Entry("should not skip when plugin preset has option with different value",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter",
					test.MustReturnJSONFor(2), nil),
			),
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
								Value: test.MustReturnJSONFor(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			false,
		),
		Entry("should not skip when one of plugin preset option has more then one value and one of them is different then option in plugin",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter_1",
					test.MustReturnJSONFor(1), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter_2",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter_4",
					test.MustReturnJSONFor(3), nil),
			),
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
								Value: test.MustReturnJSONFor(1),
							},
							{
								Name:  "plugin_preset.test_parameter_2",
								Value: test.MustReturnJSONFor(2),
							},
							{
								Name:  "plugin_preset.test_parameter_4",
								Value: test.MustReturnJSONFor(4),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			false,
		),
		Entry("should skip when plugin preset has the same values like plugin",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter",
					test.MustReturnJSONFor(3), nil),
			),
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
								Value: test.MustReturnJSONFor(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			true,
		),
		Entry("should not skip when plugin has custom values",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_preset.test_parameter",
					test.MustReturnJSONFor(3), nil),
				test.WithPluginOptionValue("custom_parameter",
					test.MustReturnJSONFor(123), nil),
			),
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
								Value: test.MustReturnJSONFor(3),
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			"",
			false,
		),
		Entry("should skip when plugin has default values from plugin definition",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_definition.test_parameter",
					test.MustReturnJSONFor(3), nil),
			),
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
			&greenhousev1alpha1.ClusterPluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: test.MustReturnJSONFor(3),
						},
					},
				},
			},
			"",
			true,
		),
		Entry("should not skip when plugin has different values then plugin definition",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_definition.test_parameter",
					test.MustReturnJSONFor(3), nil),
			),
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
			&greenhousev1alpha1.ClusterPluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: test.MustReturnJSONFor(4),
						},
					},
				},
			},
			"",
			false,
		), Entry("should not skip when Plugin has different valueFrom then PluginDefinition",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_definition.test_parameter",
					nil, &greenhousev1alpha1.ValueFromSource{
						Secret: &greenhousev1alpha1.SecretKeyReference{
							Name: "test-secret",
							Key:  "test-key",
						},
					}),
			),
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
			&greenhousev1alpha1.ClusterPluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: test.MustReturnJSONFor(4),
						},
					},
				},
			},
			"",
			false,
		), Entry("should skip when Plugin has same valueFrom as PluginPreset",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("plugin_definition.test_parameter",
					nil, &greenhousev1alpha1.ValueFromSource{
						Secret: &greenhousev1alpha1.SecretKeyReference{
							Name: "test-secret",
							Key:  "test-key",
						},
					}),
			),
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
			&greenhousev1alpha1.ClusterPluginDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: pluginPresetDefinitionName,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Options: []greenhousev1alpha1.PluginOption{
						{
							Name:    "plugin_definition.test_parameter",
							Default: test.MustReturnJSONFor(4),
						},
					},
				},
			},
			"",
			true,
		), Entry("should not skip when Plugin has different value then plugin override",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
			),
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
								Value: test.MustReturnJSONFor(2),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "global.greenhouse.test_parameter",
									Value: test.MustReturnJSONFor(3),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
			clusterA,
			false,
		), Entry("should skip when Plugin has different value then plugin override but cluster name is different",
			test.NewPlugin(test.Ctx, "", "",
				test.WithPresetLabelValue(pluginPresetName),
				test.WithPluginDefinition(pluginPresetDefinitionName),
				test.WithPluginOptionValue("global.greenhouse.test_parameter",
					test.MustReturnJSONFor(2), nil),
			),
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
								Value: test.MustReturnJSONFor(2),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "global.greenhouse.test_parameter",
									Value: test.MustReturnJSONFor(3),
								},
							},
						},
					},
				},
			},
			&greenhousev1alpha1.ClusterPluginDefinition{},
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
			test.NewPlugin(test.Ctx, "", "", test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(2), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{},
			},
			test.NewPlugin(test.Ctx, "", "", test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(2), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
		),
		Entry("with defined pluginPresetOverrides but for another cluster",
			test.NewPlugin(test.Ctx, "", clusterA, test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
				test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(2), nil)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterB,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: test.MustReturnJSONFor(1),
								},
							},
						},
					},
				},
			},
			test.NewPlugin(test.Ctx, "", clusterA, test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
				test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(2), nil)),
		),
		Entry("with defined pluginPresetOverrides for the correct cluster",
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(2), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: test.MustReturnJSONFor(1),
								},
							},
						},
					},
				},
			},
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
		),
		Entry("with defined pluginPresetOverrides for the cluster and plugin with empty option values",
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-1",
									Value: test.MustReturnJSONFor(1),
								},
							},
						},
					},
				},
			},
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
		),
		Entry("with defined pluginPresetOverrides and plugin has two options",
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil), test.WithPluginOptionValue("option-2", test.MustReturnJSONFor(1), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-2",
									Value: test.MustReturnJSONFor(2),
								},
							},
						},
					},
				},
			},
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil), test.WithPluginOptionValue("option-2", test.MustReturnJSONFor(2), nil), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
		),
		Entry("with defined pluginPresetOverrides has multiple options to override",
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA),
				test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil),
				test.WithPluginOptionValue("option-2", test.MustReturnJSONFor(1), nil),
				test.WithPluginOptionValue("option-3", test.MustReturnJSONFor(1), nil),
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name)),
			&greenhousev1alpha1.PluginPreset{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginOptionValue{
								{
									Name:  "option-2",
									Value: test.MustReturnJSONFor(2),
								},
								{
									Name:  "option-3",
									Value: test.MustReturnJSONFor(2),
								},
								{
									Name:  "option-4",
									Value: test.MustReturnJSONFor(2),
								},
							},
						},
					},
				},
			},
			test.NewPlugin(test.Ctx, "", clusterA, test.WithCluster(clusterA), test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
				test.WithPluginOptionValue("option-1", test.MustReturnJSONFor(1), nil),
				test.WithPluginOptionValue("option-2", test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("option-3", test.MustReturnJSONFor(2), nil),
				test.WithPluginOptionValue("option-4", test.MustReturnJSONFor(2), nil)),
		),
	)
})

var _ = Describe("getReleaseName", func() {
	It("returns plugin.Spec.ReleaseName if set", func() {
		plugin := &greenhousev1alpha1.Plugin{Spec: greenhousev1alpha1.PluginSpec{ReleaseName: "explicit-release"}}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginSpec{ReleaseName: "preset-release"}}}
		Expect(getReleaseName(plugin, preset)).To(Equal("explicit-release"))
	})

	It("returns plugin.Name if HelmReleaseStatus is set and ReleaseName is empty", func() {
		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: "plugin-name"},
			Spec:       greenhousev1alpha1.PluginSpec{ReleaseName: ""},
			Status:     greenhousev1alpha1.PluginStatus{HelmReleaseStatus: &greenhousev1alpha1.HelmReleaseStatus{}},
		}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginSpec{ReleaseName: "preset-release"}}}
		Expect(getReleaseName(plugin, preset)).To(Equal("plugin-name"))
	})

	It("returns preset.Spec.Plugin.ReleaseName if plugin.Spec.ReleaseName is empty and no HelmReleaseStatus", func() {
		plugin := &greenhousev1alpha1.Plugin{Spec: greenhousev1alpha1.PluginSpec{ReleaseName: ""}}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginSpec{ReleaseName: "preset-release"}}}
		Expect(getReleaseName(plugin, preset)).To(Equal("preset-release"))
	})
})

// clusterSecret returns the secret for a cluster.
func clusterSecret(clusterName, supportGroupTeamName string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: test.TestNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: supportGroupTeamName},
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
	}
}

// cluster returns a cluster object with the given name.
func cluster(name, supportGroupTeamName string) *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				"cluster":                      name,
				greenhouseapis.LabelKeyOwnedBy: supportGroupTeamName,
				"foo":                          "bar",
			},
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}
}

func pluginPreset(name, selectorValue, supportGroupTeamName string) *greenhousev1alpha1.PluginPreset {
	preset := test.NewPluginPreset(name, test.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, supportGroupTeamName))
	preset.Spec = greenhousev1alpha1.PluginPresetSpec{
		Plugin: greenhousev1alpha1.PluginSpec{
			PluginDefinition: pluginPresetDefinitionName,
			ReleaseName:      releaseName,
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
	}
	return preset
}
