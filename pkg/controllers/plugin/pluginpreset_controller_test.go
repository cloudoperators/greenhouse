// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	pluginPresetName           = "test-pluginpreset"
	pluginPresetDefinitionName = "test-plugindefinition"

	releaseNamespace = "test-namespace"

	clusterA = "cluster-a"
	clusterB = "cluster-b"
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

	testPluginPreset = &greenhousev1alpha1.PluginPreset{
		TypeMeta: metav1.TypeMeta{
			Kind:       greenhousev1alpha1.PluginPresetKind,
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginPresetName,
			Namespace: test.TestNamespace,
		},
		Spec: greenhousev1alpha1.PluginPresetSpec{
			PluginDefinition: testPluginDefinition.Name,
			ReleaseNamespace: releaseNamespace,
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			},
		},
	}
)

var _ = Describe("PluginPreset Controller", Ordered, func() {
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
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: testPluginDefinition.Name + "-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		Expect(expPlugin.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginPreset, pluginPresetName), "the Plugin should be labeled as managed by the PluginPreset")

		By("modifying the Plugin and ensuring it is reconciled")
		expPlugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
			{Name: "option1", Value: test.MustReturnJSONFor("value1")},
		}
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, expPlugin, func() error {
			expPlugin.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
				{Name: "option1", Value: test.MustReturnJSONFor("value1")},
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update Plugin")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Plugin")
			g.Expect(expPlugin.Spec.OptionValues).ToNot(ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "option1", Value: test.MustReturnJSONFor("value1")}), "the Plugin should be reconciled")
		}).Should(Succeed(), "the Plugin should be reconciled")

		By("manually creating a Plugin with OwnerReference but cluster not matching the selector")
		pluginNotExp := &greenhousev1alpha1.Plugin{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Plugin",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin-" + clusterB,
				Namespace: test.TestNamespace,
				Labels: map[string]string{
					greenhouseapis.LabelKeyPluginPreset: pluginPresetName,
				},
				OwnerReferences: expPlugin.OwnerReferences, // copy the OwnerReference to ensure same behavior
			},
			Spec: greenhousev1alpha1.PluginSpec{
				ClusterName:      clusterB,
				PluginDefinition: testPluginDefinition.Name,
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginNotExp)).Should(Succeed(), "failed to create test Plugin")

		By("ensuring the Plugin is deleted")
		Eventually(func(g Gomega) error {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: pluginNotExp.Name, Namespace: pluginNotExp.Namespace}, pluginNotExp)
			g.Expect(err).To(HaveOccurred(), "there should be an error getting the Plugin")
			return client.IgnoreNotFound(err)
		}).Should(Succeed(), "the Plugin should be deleted")

		By("deleting the PluginPreset to ensure all Plugins are deleted")
		Expect(test.K8sClient.Delete(test.Ctx, testPluginPreset)).Should(Succeed(), "failed to delete test PluginPreset")
		Eventually(func(g Gomega) error {
			err := test.K8sClient.Get(test.Ctx, expPluginName, pluginNotExp)
			g.Expect(err).To(HaveOccurred(), "there should be an error getting the Plugin")
			return client.IgnoreNotFound(err)
		}).Should(Succeed(), "the Plugin should be deleted")
	})
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
func cluster(clusterName string) *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: test.TestNamespace,
			Labels: map[string]string{
				"cluster": clusterName,
			},
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}
}
