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
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("PluginPreset Conversion", Ordered, func() {
	const (
		clusterName      = "cluster-conversion-a"
		releaseNamespace = "test-namespace"
	)
	var (
		setup              *test.TestSetup
		team               *greenhousev1alpha1.Team
		pluginDefinition   *greenhousev1alpha1.ClusterPluginDefinition
		clusterARemoteEnv  *envtest.Environment
		clusterAKubeConfig []byte
	)

	BeforeAll(func() {
		By("bootstrapping the remote cluster")
		_, _, clusterARemoteEnv, clusterAKubeConfig = test.StartControlPlane("6888", false, false)
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "pluginpreset-conversion")

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
			test.WithClusterLabel("cluster", "a"),
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

	Context("Validate Conversion of PluginPreset resource between v1alpha1 and v1alpha2 versions", func() {
		It("should correctly convert the PluginPreset with ClusterSelector from v1alpha1 to the hub version (v1alpha2)", func() {
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
						OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
							{
								Name: "option1", Value: test.MustReturnJSONFor("value1"),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: "cluster-a", Overrides: []greenhousemetav1alpha1.PluginOptionValue{
								{Name: "option1", Value: test.MustReturnJSONFor("value2")},
							},
						},
					},
				},
			}
			Expect(setup.Create(test.Ctx, pluginPresetV1alpha1)).To(Succeed(), "PluginPreset in v1alpha1 version should be created successfully")

			By("validating the conversion to v1alpha2 version")
			pluginPresetV1alpha2 := &greenhousev1alpha2.PluginPreset{}
			presetKey := types.NamespacedName{Name: pluginPresetV1alpha1.Name, Namespace: pluginPresetV1alpha1.Namespace}
			Expect(setup.Get(test.Ctx, presetKey, pluginPresetV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 PluginPreset")

			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.Name).To(Equal(pluginPresetV1alpha1.Spec.ClusterName), ".Spec.ClusterName in PluginPreset should be correctly converted between versions")
			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(pluginPresetV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector in PluginPreset should be correctly converted between versions")

			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseNamespace).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseNamespace))
			Expect(pluginPresetV1alpha2.Spec.Plugin.DisplayName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.DisplayName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.PluginDefinition).To(Equal(pluginPresetV1alpha1.Spec.Plugin.PluginDefinition))
			Expect(pluginPresetV1alpha2.Spec.Plugin.OptionValues).To(Equal(pluginPresetV1alpha1.Spec.Plugin.OptionValues))
			Expect(toComparableClusterOptionsOverrides(pluginPresetV1alpha2.Spec.ClusterOptionOverrides)).
				To(ConsistOf(toComparableClusterOptionsOverrides(pluginPresetV1alpha1.Spec.ClusterOptionOverrides)))

			Expect(pluginPresetV1alpha2.Status.StatusConditions).To(Equal(pluginPresetV1alpha1.Status.StatusConditions))
			Expect(toComparableManagedPluginStatus(pluginPresetV1alpha2.Status.PluginStatuses)).
				To(ConsistOf(toComparableManagedPluginStatus(pluginPresetV1alpha1.Status.PluginStatuses)))
			Expect(pluginPresetV1alpha2.Status.AvailablePlugins).To(Equal(pluginPresetV1alpha1.Status.AvailablePlugins))
			Expect(pluginPresetV1alpha2.Status.ReadyPlugins).To(Equal(pluginPresetV1alpha1.Status.ReadyPlugins))
			Expect(pluginPresetV1alpha2.Status.FailedPlugins).To(Equal(pluginPresetV1alpha1.Status.FailedPlugins))

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPresetV1alpha2)
		})

		It("should correctly convert the PluginPreset with ClusterName from v1alpha1 to the hub version (v1alpha2)", func() {
			By("creating a PluginPreset with v1alpha1 version on the central cluster")
			pluginPresetV1alpha1 := &greenhousev1alpha1.PluginPreset{
				TypeMeta: metav1.TypeMeta{
					Kind:       greenhousev1alpha1.PluginPresetKind,
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      setup.RandomizeName("test-preset-2"),
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.PluginPresetSpec{
					ClusterName: "cluster-a",
					Plugin: greenhousev1alpha1.PluginSpec{
						ReleaseName:      "test-release-1",
						ReleaseNamespace: "release-namespace",
						DisplayName:      "Display name 1",
						PluginDefinition: pluginDefinition.Name,
						OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
							{
								Name: "option1", Value: test.MustReturnJSONFor("value1"),
							},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: "cluster-a", Overrides: []greenhousemetav1alpha1.PluginOptionValue{
								{Name: "option1", Value: test.MustReturnJSONFor("value2")},
							},
						},
					},
				},
			}
			Expect(setup.Create(test.Ctx, pluginPresetV1alpha1)).To(Succeed(), "PluginPreset in v1alpha1 version should be created successfully")

			By("validating the conversion to v1alpha2 version")
			pluginPresetV1alpha2 := &greenhousev1alpha2.PluginPreset{}
			presetKey := types.NamespacedName{Name: pluginPresetV1alpha1.Name, Namespace: pluginPresetV1alpha1.Namespace}
			Expect(setup.Get(test.Ctx, presetKey, pluginPresetV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 PluginPreset")

			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.Name).To(Equal(pluginPresetV1alpha1.Spec.ClusterName), ".Spec.ClusterName in PluginPreset should be correctly converted between versions")
			Expect(pluginPresetV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(pluginPresetV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector in PluginPreset should be correctly converted between versions")

			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.ReleaseNamespace).To(Equal(pluginPresetV1alpha1.Spec.Plugin.ReleaseNamespace))
			Expect(pluginPresetV1alpha2.Spec.Plugin.DisplayName).To(Equal(pluginPresetV1alpha1.Spec.Plugin.DisplayName))
			Expect(pluginPresetV1alpha2.Spec.Plugin.PluginDefinition).To(Equal(pluginPresetV1alpha1.Spec.Plugin.PluginDefinition))
			Expect(pluginPresetV1alpha2.Spec.Plugin.OptionValues).To(Equal(pluginPresetV1alpha1.Spec.Plugin.OptionValues))
			Expect(toComparableClusterOptionsOverrides(pluginPresetV1alpha2.Spec.ClusterOptionOverrides)).
				To(ConsistOf(toComparableClusterOptionsOverrides(pluginPresetV1alpha1.Spec.ClusterOptionOverrides)))

			Expect(pluginPresetV1alpha2.Status.StatusConditions).To(Equal(pluginPresetV1alpha1.Status.StatusConditions))
			Expect(toComparableManagedPluginStatus(pluginPresetV1alpha2.Status.PluginStatuses)).
				To(ConsistOf(toComparableManagedPluginStatus(pluginPresetV1alpha1.Status.PluginStatuses)))
			Expect(pluginPresetV1alpha2.Status.AvailablePlugins).To(Equal(pluginPresetV1alpha1.Status.AvailablePlugins))
			Expect(pluginPresetV1alpha2.Status.ReadyPlugins).To(Equal(pluginPresetV1alpha1.Status.ReadyPlugins))
			Expect(pluginPresetV1alpha2.Status.FailedPlugins).To(Equal(pluginPresetV1alpha1.Status.FailedPlugins))

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPresetV1alpha2)
		})

		It("should correctly convert the PluginPreset with LabelSelector from v1alpha2 to the v1alpha1 version", func() {
			By("creating v1alpha2 PluginPreset")
			pluginPreset := test.NewPluginPreset(setup.RandomizeName("test-preset-3"), test.TestNamespace,
				test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithPluginPresetClusterSelector(greenhousev1alpha2.ClusterSelector{
					LabelSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{"cluster": "a"},
					},
				}),
				test.WithPluginPresetPluginTemplateSpec(greenhousev1alpha2.PluginTemplateSpec{
					PluginDefinition: pluginDefinition.Name,
					ReleaseName:      "test-release-1",
					ReleaseNamespace: "release-namespace",
					DisplayName:      "Display name 1",
					OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
						{Name: "option1", Value: test.MustReturnJSONFor("value1")},
						{Name: "option2", Value: test.MustReturnJSONFor("value11")},
					},
				}),
				test.WithPluginPresetClusterOptionOverrides([]greenhousev1alpha2.ClusterOptionOverride{
					{
						ClusterName: "cluster-a", Overrides: []greenhousemetav1alpha1.PluginOptionValue{
							{Name: "option1", Value: test.MustReturnJSONFor("value2")},
							{Name: "option2", Value: test.MustReturnJSONFor("value12")},
						},
					},
				}),
			)
			err := setup.Create(test.Ctx, pluginPreset)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginPreset in v1alpha2")

			By("validating the conversion to v1alpha1 version")
			pluginPresetV1alpha1 := &greenhousev1alpha1.PluginPreset{}
			presetKey := types.NamespacedName{Name: pluginPreset.Name, Namespace: pluginPreset.Namespace}
			Expect(setup.Get(test.Ctx, presetKey, pluginPresetV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 PluginPreset")

			Expect(pluginPresetV1alpha1.Spec.ClusterName).To(Equal(pluginPreset.Spec.ClusterSelector.Name), ".Spec.ClusterSelector.ClusterName in PluginPreset should be correctly converted between versions")
			Expect(pluginPresetV1alpha1.Spec.ClusterSelector).To(Equal(pluginPreset.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector.LabelSelector in PluginPreset should be correctly converted between versions")

			Expect(pluginPresetV1alpha1.Spec.Plugin.ReleaseName).To(Equal(pluginPreset.Spec.Plugin.ReleaseName))
			Expect(pluginPresetV1alpha1.Spec.Plugin.ReleaseNamespace).To(Equal(pluginPreset.Spec.Plugin.ReleaseNamespace))
			Expect(pluginPresetV1alpha1.Spec.Plugin.DisplayName).To(Equal(pluginPreset.Spec.Plugin.DisplayName))
			Expect(pluginPresetV1alpha1.Spec.Plugin.PluginDefinition).To(Equal(pluginPreset.Spec.Plugin.PluginDefinition))
			Expect(pluginPresetV1alpha1.Spec.Plugin.OptionValues).To(Equal(pluginPreset.Spec.Plugin.OptionValues))
			Expect(toComparableClusterOptionsOverrides(pluginPresetV1alpha1.Spec.ClusterOptionOverrides)).
				To(ConsistOf(toComparableClusterOptionsOverrides(pluginPreset.Spec.ClusterOptionOverrides)))

			Expect(pluginPresetV1alpha1.Status.StatusConditions).To(Equal(pluginPreset.Status.StatusConditions))
			Expect(toComparableManagedPluginStatus(pluginPresetV1alpha1.Status.PluginStatuses)).
				To(ConsistOf(toComparableManagedPluginStatus(pluginPreset.Status.PluginStatuses)))
			Expect(pluginPresetV1alpha1.Status.AvailablePlugins).To(Equal(pluginPreset.Status.AvailablePlugins))
			Expect(pluginPresetV1alpha1.Status.ReadyPlugins).To(Equal(pluginPreset.Status.ReadyPlugins))
			Expect(pluginPresetV1alpha1.Status.FailedPlugins).To(Equal(pluginPreset.Status.FailedPlugins))

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPreset)
		})

		It("should correctly convert the PluginPreset with ClusterName from v1alpha2 to the v1alpha1 version", func() {
			By("creating v1alpha2 PluginPreset")
			pluginPreset := test.NewPluginPreset(setup.RandomizeName("test-preset-4"), test.TestNamespace,
				test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithPluginPresetClusterSelector(greenhousev1alpha2.ClusterSelector{
					Name: "cluster-a",
				}),
				test.WithPluginPresetPluginTemplateSpec(greenhousev1alpha2.PluginTemplateSpec{
					PluginDefinition: pluginDefinition.Name,
					ReleaseName:      "test-release-1",
					ReleaseNamespace: "release-namespace",
					DisplayName:      "Display name 1",
					OptionValues: []greenhousemetav1alpha1.PluginOptionValue{
						{Name: "option3", Value: test.MustReturnJSONFor("value3")},
						{Name: "option4", Value: test.MustReturnJSONFor("value4")},
					},
				}),
				test.WithPluginPresetClusterOptionOverrides([]greenhousev1alpha2.ClusterOptionOverride{
					{
						ClusterName: "cluster-a", Overrides: []greenhousemetav1alpha1.PluginOptionValue{
							{Name: "option3", Value: test.MustReturnJSONFor("value2")},
							{Name: "option4", Value: test.MustReturnJSONFor("value12")},
						},
					},
				}),
			)
			err := setup.Create(test.Ctx, pluginPreset)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginPreset in v1alpha2")

			By("validating the conversion to v1alpha1 version")
			pluginPresetV1alpha1 := &greenhousev1alpha1.PluginPreset{}
			presetKey := types.NamespacedName{Name: pluginPreset.Name, Namespace: pluginPreset.Namespace}
			Expect(setup.Get(test.Ctx, presetKey, pluginPresetV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 PluginPreset")

			Expect(pluginPresetV1alpha1.Spec.ClusterName).To(Equal(pluginPreset.Spec.ClusterSelector.Name), ".Spec.ClusterSelector.ClusterName in PluginPreset should be correctly converted between versions")
			Expect(pluginPresetV1alpha1.Spec.ClusterSelector).To(Equal(pluginPreset.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector.LabelSelector in PluginPreset should be correctly converted between versions")

			Expect(pluginPresetV1alpha1.Spec.Plugin.ReleaseName).To(Equal(pluginPreset.Spec.Plugin.ReleaseName))
			Expect(pluginPresetV1alpha1.Spec.Plugin.ReleaseNamespace).To(Equal(pluginPreset.Spec.Plugin.ReleaseNamespace))
			Expect(pluginPresetV1alpha1.Spec.Plugin.DisplayName).To(Equal(pluginPreset.Spec.Plugin.DisplayName))
			Expect(pluginPresetV1alpha1.Spec.Plugin.PluginDefinition).To(Equal(pluginPreset.Spec.Plugin.PluginDefinition))
			Expect(pluginPresetV1alpha1.Spec.Plugin.OptionValues).To(Equal(pluginPreset.Spec.Plugin.OptionValues))
			Expect(toComparableClusterOptionsOverrides(pluginPresetV1alpha1.Spec.ClusterOptionOverrides)).
				To(ConsistOf(toComparableClusterOptionsOverrides(pluginPreset.Spec.ClusterOptionOverrides)))

			Expect(pluginPresetV1alpha1.Status.StatusConditions).To(Equal(pluginPreset.Status.StatusConditions))
			Expect(toComparableManagedPluginStatus(pluginPresetV1alpha1.Status.PluginStatuses)).
				To(ConsistOf(toComparableManagedPluginStatus(pluginPreset.Status.PluginStatuses)))
			Expect(pluginPresetV1alpha1.Status.AvailablePlugins).To(Equal(pluginPreset.Status.AvailablePlugins))
			Expect(pluginPresetV1alpha1.Status.ReadyPlugins).To(Equal(pluginPreset.Status.ReadyPlugins))
			Expect(pluginPresetV1alpha1.Status.FailedPlugins).To(Equal(pluginPreset.Status.FailedPlugins))

			By("cleaning up the created PluginPreset")
			test.EventuallyDeleted(test.Ctx, setup.Client, pluginPreset)
		})
	})
})

func toComparableClusterOptionsOverrides(items any) []map[string]any {
	var result []map[string]any
	convert := func(clusterName string, overrides []greenhousemetav1alpha1.PluginOptionValue) map[string]any {
		return map[string]any{
			"ClusterName": clusterName,
			"Overrides":   overrides,
		}
	}
	switch v := items.(type) {
	case []greenhousev1alpha1.ClusterOptionOverride:
		for _, item := range v {
			result = append(result, convert(item.ClusterName, item.Overrides))
		}
	case []greenhousev1alpha2.ClusterOptionOverride:
		for _, item := range v {
			result = append(result, convert(item.ClusterName, item.Overrides))
		}
	}
	return result
}

func toComparableManagedPluginStatus(items any) []map[string]any {
	var result []map[string]any
	convert := func(pluginName string, readyCondition greenhousemetav1alpha1.Condition) map[string]any {
		return map[string]any{
			"PluginName":     pluginName,
			"ReadyCondition": readyCondition,
		}
	}
	switch v := items.(type) {
	case []greenhousev1alpha1.ManagedPluginStatus:
		for _, item := range v {
			result = append(result, convert(item.PluginName, item.ReadyCondition))
		}
	case []greenhousev1alpha2.ManagedPluginStatus:
		for _, item := range v {
			result = append(result, convert(item.PluginName, item.ReadyCondition))
		}
	}
	return result
}
