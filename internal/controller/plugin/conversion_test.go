// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("PluginPreset Conversion", Ordered, func() {
	const (
		clusterName      = "test-cluster-a"
		releaseNamespace = "test-namespace"
	)
	var (
		setup *test.TestSetup
		// clusterA           *greenhousev1alpha1.Cluster
		team             *greenhousev1alpha1.Team
		pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
		// clusterAKubeClient client.Client
		clusterARemoteEnv  *envtest.Environment
		clusterAKubeConfig []byte
	)

	BeforeAll(func() {
		By("bootstrapping the remote cluster")
		_, _, clusterARemoteEnv, clusterAKubeConfig = test.StartControlPlane("6886", false, false)
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac-conversion")

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the remote cluster")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(clusterAKubeConfig, releaseNamespace, clientutil.WithPersistentConfig())
		remoteK8sClient, err := clientutil.NewK8sClientFromRestClientGetter(remoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = remoteK8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: releaseNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

		By("creating the test team")
		team = setup.CreateTeam(test.Ctx, "test-team-conversion", test.WithMappedIDPGroup("test-idp-group"))

		By("creating the test cluster")
		setup.CreateCluster(test.Ctx, clusterName,
			test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithClusterLabel("cluster", clusterName),
			test.WithClusterLabel("foo", "bar"),
			test.WithAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		)

		By("creating a secret with a valid kubeconfig for a remote cluster")
		setup.CreateSecret(test.Ctx, clusterName,
			test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
			test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: clusterAKubeConfig}),
		)

		By("creating the PluginDefinition")
		pluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "test-plugin-definition-1")
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, setup.Client, team)

		By("tearing down the remote test environment")
		err := clusterARemoteEnv.Stop()
		Expect(err).NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
	})

	Context("Validate Conversion of PluginPreset resource", func() {
		FIt("should not convert to v1alpha1 when creating v1alpha2 PluginPreset", func() {
			By("creating v1alpha2 PluginPreset")
			pluginPreset := test.NewPluginPreset("test-preset-1", test.TestNamespace,
				test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithPluginPresetClusterSelector(greenhousev1alpha2.ClusterSelector{
					Name: clusterName,
				}),
			)
			err := setup.Create(test.Ctx, pluginPreset)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginPreset in v1alpha2")

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPreset)
		})

		It("should correctly convert the PluginPreset with LabelSelector from v1alpha1 to the hub version (v1alpha2)", func() {
			By("creating a PluginPreset with v1alpha1 version on the central cluster")
			pluginPresetV1alpha1 := &greenhousev1alpha1.PluginPreset{
				TypeMeta: metav1.TypeMeta{
					Kind:       greenhousev1alpha1.PluginPresetKind,
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      setup.RandomizeName("test-preset-1"),
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{"cluster": "a"},
					},
					Plugin: greenhousev1alpha1.PluginSpec{
						ReleaseName:      "test-release-1",
						ReleaseNamespace: "release-namespace",
						DisplayName:      "Display name 1",
						PluginDefinition: pluginDefinition.Name,
						//OptionValues: , // TODO
					},
				},
			}
			Expect(setup.Create(test.Ctx, pluginPresetV1alpha1)).To(Succeed(), "PluginPreset in v1alpha1 version should be created successfully")

			By("validating the conversion to v1alpha2 version")
			pluginPresetV1alpha2 := &greenhousev1alpha2.PluginPreset{}
			presetKey := types.NamespacedName{Name: pluginPresetV1alpha1.Name, Namespace: pluginPresetV1alpha1.Namespace}
			Expect(setup.Get(test.Ctx, presetKey, pluginPresetV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 PluginPreset")

			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.Name).To(BeEmpty(), ".Spec.ClusterSelector.ClusterName in PluginPreset should be empty")
			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(pluginPresetV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector.LabelSelector in PluginPreset should be correctly converted between versions")

			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseNamespace).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseNamespace))
			Expect(pluginPresetV1alpha2.Spec.Plugin.DisplayName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.DisplayName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.PluginDefinition).To(Equal(pluginPresetV1alpha1.Spec.Plugin.PluginDefinition))

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPresetV1alpha2)
		})
	})
})
