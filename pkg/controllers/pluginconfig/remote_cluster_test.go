// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package pluginconfig_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var testPluginConfig = &greenhousev1alpha1.PluginConfig{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PluginConfig",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-plugin",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginConfigSpec{
		Plugin: "test-plugin",
	},
}

var testPCwithSR = &greenhousev1alpha1.PluginConfig{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PluginConfig",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-pluginconfig-secretref",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.PluginConfigSpec{
		Plugin: "test-plugin",
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name: "secretValue",
				ValueFrom: &greenhousev1alpha1.ValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "test-secret",
						Key:  "test-key",
					},
				},
			},
		},
	},
}

var testSecret = corev1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: corev1.GroupName,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-secret",
		Namespace: test.TestNamespace,
	},
	Data: map[string][]byte{
		"test-key": []byte("secret-value"),
	},
}

var testPlugin = &greenhousev1alpha1.Plugin{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Plugin",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-plugin",
		Namespace: corev1.NamespaceDefault,
	},
	Spec: greenhousev1alpha1.PluginSpec{
		Description: "Testplugin",
		Version:     "1.0.0",
		HelmChart: &greenhousev1alpha1.HelmChartReference{
			Name:       "./../../test/fixtures/myChart",
			Repository: "dummy",
			Version:    "1.0.0",
		},
	},
}

var testCluster = &greenhousev1alpha1.Cluster{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	Spec: greenhousev1alpha1.ClusterSpec{
		AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
	},
}

var testClusterK8sSecret = corev1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: corev1.GroupName,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	Type: greenhouseapis.SecretTypeKubeConfig,
}

var (
	remoteKubeConfig []byte
	remoteEnvTest    *envtest.Environment
)

var _ = Describe("Validate pluginConfig clusterName", Ordered, func() {

	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin")

		By("bootstrapping remote cluster")
		bootstrapRemoteCluster()

		By("creating a cluster")
		Expect(test.K8sClient.Create(test.Ctx, testCluster)).Should(Succeed(), "there should be no error creating the cluster resource")

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the cluster")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPluginConfig.Namespace, clientutil.WithPersistentConfig())
		remoteK8sClient, err := clientutil.NewK8sClientFromRestClientGetter(remoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = remoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testPluginConfig.Namespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating a secret with a valid kubeconfig for a remote cluster")
		testClusterK8sSecret.Data = map[string][]byte{
			greenhouseapis.KubeConfigKey: remoteKubeConfig,
		}
		Expect(test.K8sClient.Create(test.Ctx, &testClusterK8sSecret)).Should(Succeed())

	})

	AfterAll(func() {
		err := remoteEnvTest.Stop()
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error stopping the remote environment")

	})

	It("should correctly handle the pluginConfig on a referenced cluster", func() {
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPluginConfig.Namespace, clientutil.WithPersistentConfig())

		By("creating a pluginconfig referencing the cluster")
		testPluginConfig.Spec.ClusterName = "test-cluster"
		Expect(test.K8sClient.Create(test.Ctx, testPluginConfig)).Should(Succeed(), "there should be no error updating the pluginConfig")

		By("checking the ClusterAccessReadyCondition on the pluginConfig")
		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginConfig.Name, Namespace: testPluginConfig.Namespace}, testPluginConfig)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the pluginConfig")
			clusterAccessReadyCondition := testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
			readyCondition := testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
			g.Expect(clusterAccessReadyCondition).ToNot(BeNil(), "the ClusterAccessReadyCondition should not be nil")
			g.Expect(readyCondition).ToNot(BeNil(), "the ReadyCondition should not be nil")
			g.Expect(testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition).IsFalse()).Should(BeTrue(), "the ClusterAccessReadyCondition should be false")
			g.Expect(testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition).IsFalse()).Should(BeTrue(), "the ReadyCondition should be false")
			return true
		}).Should(BeTrue(), "the ClusterAccessReadyCondition should be false")

		By("setting the ready condition on the test-cluster")
		testCluster.Status.StatusConditions.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ReadyCondition, "", ""))
		Expect(test.K8sClient.Status().Update(test.Ctx, testCluster)).Should(Succeed(), "there should be no error updating the cluster resource")

		By("triggering setting a label on the pluginConfig to trigger reconciliation")
		testPluginConfig.Labels = map[string]string{"test": "label"}
		Expect(test.K8sClient.Update(test.Ctx, testPluginConfig)).Should(Succeed(), "there should be no error updating the pluginConfig")

		By("checking the ClusterAccessReadyCondition on the pluginConfig")
		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginConfig.Name, Namespace: testPluginConfig.Namespace}, testPluginConfig)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the pluginConfig")
			g.Expect(testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition).IsTrue()).Should(BeTrue(), "the ClusterAccessReadyCondition should be true")
			g.Expect(testPluginConfig.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition).IsTrue()).Should(BeTrue(), "the ReadyCondition should be true")
			return true
		}).Should(BeTrue(), "the ClusterAccessReadyCondition should be true")

		By("checking the helm releases deployed to the remote cluster")
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPluginConfig.Namespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
		listAction := action.NewList(helmConfig)

		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(ContainElement(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal("test-plugin")}))), "the helm release should be deployed to the remote cluster")

		By("deleting the pluginConfig")
		Expect(test.K8sClient.Delete(test.Ctx, testPluginConfig)).Should(Succeed(), "there should be no error deleting the pluginConfig")

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

	It("should correctly handle the pluginConfig on a referenced cluster with a secret reference", func() {
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPluginConfig.Namespace, clientutil.WithPersistentConfig())

		By("creating a secret holding the OptionValue referenced by the PluginConfig")
		Expect(test.K8sClient.Create(test.Ctx, &testSecret)).Should(Succeed())

		By("creating a pluginconfig referencing the cluster")
		testPCwithSR.Spec.ClusterName = "test-cluster"
		Expect(test.K8sClient.Create(test.Ctx, testPCwithSR)).Should(Succeed(), "there should be no error updating the pluginConfig")

		By("checking the helm releases deployed to the remote cluster")
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPCwithSR.Namespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
		listAction := action.NewList(helmConfig)

		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(ContainElement(
			gstruct.PointTo(
				gstruct.MatchFields(
					gstruct.IgnoreExtras, gstruct.Fields{
						"Name":   Equal("test-pluginconfig-secretref"),
						"Config": gstruct.MatchKeys(gstruct.IgnoreExtras, gstruct.Keys{"secretValue": Equal("secret-value")})}))), "the helm release should be deployed to the remote cluster")

		By("deleting the pluginConfig")
		Expect(test.K8sClient.Delete(test.Ctx, testPCwithSR)).Should(Succeed(), "there should be no error deleting the pluginConfig")

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

})

func bootstrapRemoteCluster() {
	_, _, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
}
