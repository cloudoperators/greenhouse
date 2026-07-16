// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"slices"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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
			Name:       "dummy",
			Repository: "oci://greenhouse/helm-charts",
			Version:    "1.0.0",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:        "myRequiredOption",
			Description: "This is my required test plugin option",
			Required:    true,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}))
	pluginPresetPluginSpec = greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: pluginPresetDefinitionName,
		},
		ReleaseName:      releaseName,
		ReleaseNamespace: releaseNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "myRequiredOption",
				Value: test.MustReturnJSONFor("myValue"),
			},
		},
	}
)

// verifyHelmReleaseExists checks that a HelmRelease was created for a Plugin.
func verifyHelmReleaseExists(g Gomega, pluginName, namespace string) {
	GinkgoHelper()
	helmRelease := &helmv2.HelmRelease{}
	helmReleaseID := types.NamespacedName{Name: pluginName, Namespace: namespace}
	err := test.K8sClient.Get(test.Ctx, helmReleaseID, helmRelease)
	g.Expect(err).ToNot(HaveOccurred(), "HelmRelease should exist for Plugin %s", pluginName)
}

// verifyPluginCreatedWithHelmRelease checks that a Plugin exists and has a HelmRelease created for it.
func verifyPluginCreatedWithHelmRelease(g Gomega, pluginName types.NamespacedName) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := &greenhousev1alpha1.Plugin{}
	err := test.K8sClient.Get(test.Ctx, pluginName, plugin)
	g.Expect(err).ToNot(HaveOccurred(), "Plugin should exist")
	verifyHelmReleaseExists(g, pluginName.Name, pluginName.Namespace)
	return plugin
}

var _ = Describe("PluginPreset Controller Lifecycle", Ordered, func() {
	var defaultPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	BeforeAll(func() {
		format.MaxLength = 0
		By("creating a test PluginDefinition")
		err := test.K8sClient.Create(test.Ctx, pluginPresetDefinition)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginDefinition")
		test.MockHelmChartReady(test.Ctx, test.K8sClient, pluginPresetDefinition, flux.HelmRepositoryDefaultNamespace)

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

		By("creating the test Organization and Team")
		Expect(client.IgnoreAlreadyExists(test.K8sClient.Create(test.Ctx, test.NewOrganization(test.Ctx, test.TestNamespace)))).Should(Succeed(), "there should be no error creating a test Organization")
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
		test.MockHelmChartReady(test.Ctx, test.K8sClient, defaultPluginDefinition, flux.HelmRepositoryDefaultNamespace)
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
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))

		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("ensuring a Plugin has been created with HelmRelease")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-" + clusterA, Namespace: test.TestNamespace}
		var expPlugin *greenhousev1alpha1.Plugin
		Eventually(func(g Gomega) {
			expPlugin = verifyPluginCreatedWithHelmRelease(g, expPluginName)
		}).Should(Succeed(), "the Plugin should be created with HelmRelease")

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
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should reconcile a PluginPreset with plugin definition defaults", func() {
		By("ensuring a Plugin Preset has been created")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-2", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))

		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinitionWithDefaultsName,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-2-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		By("checking plugin options with plugin definition defaults and plugin preset values")
		Expect(expPlugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
			Name:  pluginPreset.Spec.Plugin.OptionValues[0].Name,
			Value: pluginPreset.Spec.Plugin.OptionValues[0].Value,
		}))

		Expect(expPlugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
			Name:  defaultPluginDefinition.Spec.Options[0].Name,
			Value: defaultPluginDefinition.Spec.Options[0].Default,
		}))

		By("removing plugin preset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should reconcile a PluginPreset on cluster changes", func() {
		By("creating a PluginPreset")
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
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
		test.MustDeleteCluster(test.Ctx, test.K8sClient, &cluster)
		Eventually(func(g Gomega) {
			err = test.K8sClient.List(test.Ctx, pluginList, client.InNamespace(cluster.GetNamespace()), client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pluginPresetName})
			g.Expect(err).NotTo(HaveOccurred(), "failed to list Plugins")
			g.Expect(pluginList.Items).To(HaveLen(1), "there should be only one Plugin")
		}).Should(Succeed(), "the PluginPreset should have removed the Plugin for the deleted Cluster")

		By("removing the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should delete a Plugin if the cluster no longer matches", func() {
		By("creating a PluginPreset")
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))

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
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should set the Status NotReady if ClusterSelector does not match", func() {
		// Create a PluginPreset with a ClusterSelector that does not match any cluster
		pluginPreset := test.NewPluginPreset("not-ready", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": "non-existing-cluster",
				},
			}))

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
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
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
			// Note: ReadyCondition may not be true since Flux is not running, but Plugin should exist
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(1), "PluginPreset Status should show exactly one plugin in total")
			g.Expect(testPluginPreset.Status.PluginDefinitionVersion).To(Equal(pluginPresetDefinition.Spec.Version), "PluginPreset status should expose the version of the referenced PluginDefinition")
			g.Expect(testPluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginDefinitionNotFoundCondition)).ToNot(BeNil(), "PluginDefinitionNotFoundCondition should be present")
			g.Expect(testPluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginDefinitionNotFoundCondition).IsFalse()).To(BeTrue(), "PluginDefinitionNotFoundCondition should be false when the PluginDefinition exists")
		}).Should(Succeed())

		By("verifying HelmRelease exists for the managed Plugin")
		expectedPluginName := testPluginPreset.Name + "-" + clusterA
		Eventually(func(g Gomega) {
			verifyHelmReleaseExists(g, expectedPluginName, test.TestNamespace)
		}).Should(Succeed(), "HelmRelease should exist for the managed Plugin")

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
				return status.PluginName == testPluginPreset.Name+"-"+clusterA
			})).To(BeTrue(), "status should be reported for the first plugin")
			g.Expect(slices.ContainsFunc(testPluginPreset.Status.PluginStatuses, func(status greenhousev1alpha1.ManagedPluginStatus) bool {
				return status.PluginName == testPluginPreset.Name+"-"+clusterB
			})).To(BeTrue(), "status should be reported for the additional plugin")
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(2), "PluginPreset Status should show exactly two plugins in total")
		}).Should(Succeed())

		By("verifying HelmReleases exist for both managed Plugins")
		Eventually(func(g Gomega) {
			verifyHelmReleaseExists(g, testPluginPreset.Name+"-"+clusterA, test.TestNamespace)
			verifyHelmReleaseExists(g, testPluginPreset.Name+"-"+clusterB, test.TestNamespace)
		}).Should(Succeed(), "HelmReleases should exist for both managed Plugins")

		By("deleting otherTestCluster to ensure the Plugin is deleted")
		test.MustDeleteCluster(test.Ctx, test.K8sClient, &cluster)
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
			g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(1), "PluginPreset Status should show exactly one plugin in total")
		}).Should(Succeed())

		By("removing the PluginPreset")
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
		test.MockHelmChartReady(test.Ctx, test.K8sClient, pluginDefinition, flux.HelmRepositoryDefaultNamespace)

		By("creating a PluginPreset with overrides")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-override1", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			},
			),
			test.WithClusterOverride(clusterA, []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "test-required-option-1", Value: test.MustReturnJSONFor(5)},
			}),
		)
		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinition.Name,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed(), "failed to create PluginPreset")
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("checking that Plugin has been created with overridden required option and HelmRelease")
		pluginObjectKey := types.NamespacedName{Name: pluginPresetName + "-override1-" + clusterA, Namespace: test.TestNamespace}
		plugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			plugin = verifyPluginCreatedWithHelmRelease(g, pluginObjectKey)
		}).Should(Succeed(), "the Plugin should be created successfully with HelmRelease")
		Expect(plugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
			Name:  pluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Name,
			Value: pluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Value,
		}),
			"ClusterOptionOverrides should be applied to the Plugin OptionValues")

		By("removing plugin preset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should save an error when Plugin creation failed due to required options being unset", func() {
		By("creating a PluginPreset based on PluginDefinition with required option")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-missing1", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinitionWithRequiredOptionName,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
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
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should successfully propagate labels from PluginPreset to Plugin", func() {
		By("ensuring a Plugin Preset has been created")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-label-propagation", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinitionWithDefaultsName,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
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
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should successfully resolve and set plugin dependencies from PluginPreset to Plugin", func() {
		By("ensuring a Plugin Preset has been created")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-wait-for", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinitionWithDefaultsName,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
		pluginPreset.Spec.WaitFor = []greenhousev1alpha1.WaitForItem{
			{
				PluginRef: greenhousev1alpha1.PluginRef{PluginPreset: "test-preset-1"},
			},
			{
				PluginRef: greenhousev1alpha1.PluginRef{Name: "dependent-plugin-1"},
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).ToNot(HaveOccurred())
		test.EventuallyCreated(test.Ctx, test.K8sClient, pluginPreset)

		By("ensuring a Plugin has been created")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-wait-for-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
		}).Should(Succeed(), "the Plugin should be created")

		By("ensuring Plugin has WaitFor dependencies copied from PluginPreset")
		Expect(expPlugin.Spec.WaitFor).To(Equal(pluginPreset.Spec.WaitFor), "the plugin should have the same plugin references as preset")

		By("removing plugin preset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should orphan Plugins with the delete-policy annotation set on a PluginPreset", func() {
		By("creating a PluginPreset")
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			},
			),
			test.WithPluginPresetDeletionPolicy(greenhouseapis.DeletionPolicyRetain))
		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("ensuring a Plugin has been created with HelmRelease")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			expPlugin = verifyPluginCreatedWithHelmRelease(g, expPluginName)
		}).Should(Succeed(), "the Plugin should be created with HelmRelease")

		Expect(expPlugin.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginPreset, pluginPresetName), "the Plugin should be labeled as managed by the PluginPreset")

		By("deleting the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)

		By("ensuring the Plugin is not deleted")
		expPluginPresetName := types.NamespacedName{Name: pluginPresetName, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginPresetName, testPluginPreset)
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the PluginPreset should be deleted")
			err = test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "the Plugin should not be deleted")
			g.Expect(expPlugin.Labels).To(HaveKey(greenhouseapis.LabelKeyPluginPreset), "the Plugin should still be labeled as managed by the PluginPreset")
			g.Expect(expPlugin.OwnerReferences).To(BeEmpty(), "the Plugin should not have an OwnerReference to the deleted PluginPreset")
		}).Should(Succeed(), "the Plugin should not be deleted after the PluginPreset is deleted")

		By("re-creating the preset")
		testPluginPreset = test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		err = test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).NotTo(HaveOccurred(), "failed to create PluginPreset")

		By("checking that the existing Plugin is now owned by the re-created PluginPreset")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginPresetName, testPluginPreset)
			g.Expect(err).ShouldNot(HaveOccurred(), "the PluginPreset should exist")
			err = test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "the Plugin should not be deleted")
			g.Expect(expPlugin.Labels).To(HaveKey(greenhouseapis.LabelKeyPluginPreset), "the Plugin should be labeled as managed by the PluginPreset")
			g.Expect(expPlugin.OwnerReferences).ToNot(BeEmpty(), "the Plugin should have an OwnerReference again")
		}).Should(Succeed(), "the Plugin should be owned by the PluginPreset again")

		By("deleting the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should recreate the Plugin when it gets deleted", func() {
		By("creating a PluginPreset")
		testPluginPreset := test.NewPluginPreset(pluginPresetName, test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(pluginPresetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))

		err := test.K8sClient.Create(test.Ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "failed to create test PluginPreset")

		By("ensuring a Plugin has been created with HelmRelease")
		expPluginName := types.NamespacedName{Name: pluginPresetName + "-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			expPlugin = verifyPluginCreatedWithHelmRelease(g, expPluginName)
		}).Should(Succeed(), "the Plugin should be created with HelmRelease")

		Expect(expPlugin.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginPreset, pluginPresetName), "the Plugin should be labeled as managed by the PluginPreset")

		By("deleting the Plugin and ensuring it is reconciled")
		originalUID := expPlugin.UID
		result, err := clientutil.Delete(test.Ctx, test.K8sClient, expPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the Plugin")
		Expect(result).To(Equal(clientutil.DeletionResultDeleted), "the Plugin should be deleted")

		By("ensuring the Plugin is recreated with HelmRelease")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Plugin")
			g.Expect(expPlugin.UID).ToNot(Equal(originalUID), "Recreated Plugin should have a new UID")
			// Also verify HelmRelease is recreated
			verifyHelmReleaseExists(g, expPluginName.Name, expPluginName.Namespace)
		}).Should(Succeed(), "the Plugin should be reconciled with HelmRelease")

		By("deleting the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
	})

	It("should reconcile a PluginPreset when the referenced ClusterPluginDefinition changes", func() {
		const watchTestDefinitionName = "watch-trigger-plugindefinition"

		By("creating a ClusterPluginDefinition with an initial default option")
		watchPluginDef := test.NewClusterPluginDefinition(test.Ctx, watchTestDefinitionName,
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:       "dummy",
				Repository: "oci://greenhouse/helm-charts",
				Version:    "1.0.0",
			}),
			test.AppendPluginOption(greenhousev1alpha1.PluginOption{
				Name:    "initialDefault",
				Type:    greenhousev1alpha1.PluginOptionTypeString,
				Default: test.MustReturnJSONFor("initialValue"),
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, watchPluginDef)).To(Succeed())
		test.MockHelmChartReady(test.Ctx, test.K8sClient, watchPluginDef, flux.HelmRepositoryDefaultNamespace)

		By("creating a PluginPreset referencing the ClusterPluginDefinition with no explicit option values")
		watchPluginSpec := greenhousev1alpha1.PluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: watchTestDefinitionName,
			},
		}
		testPluginPreset := test.NewPluginPreset("watch-trigger-preset", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetPluginSpec(watchPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, testPluginPreset)).To(Succeed())

		By("verifying the webhook set the clusterplugindefinition label on the PluginPreset")
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)).To(Succeed())
			g.Expect(testPluginPreset.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyClusterPluginDefinition, watchTestDefinitionName))
		}).Should(Succeed(), "webhook should have set the clusterplugindefinition label on the PluginPreset")

		By("waiting for the Plugin to be created with the initial default option value")
		expPluginName := types.NamespacedName{Name: "watch-trigger-preset-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)).To(Succeed())
			g.Expect(expPlugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
				Name:  "initialDefault",
				Value: test.MustReturnJSONFor("initialValue"),
			}))
		}).Should(Succeed(), "Plugin should be created with the initial default option")

		By("updating the ClusterPluginDefinition to add a new default option")
		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, watchPluginDef, func() error {
			watchPluginDef.Spec.Options = append(watchPluginDef.Spec.Options, greenhousev1alpha1.PluginOption{
				Name:    "newDefault",
				Type:    greenhousev1alpha1.PluginOptionTypeString,
				Default: test.MustReturnJSONFor("newDefaultValue"),
			})
			return nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to update ClusterPluginDefinition")

		By("verifying the Plugin is updated with the new default option after PluginPreset reconciliation")
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)).To(Succeed())
			g.Expect(expPlugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{
				Name:  "newDefault",
				Value: test.MustReturnJSONFor("newDefaultValue"),
			}))
		}).Should(Succeed(), "Plugin should have the new default option after ClusterPluginDefinition change triggers PluginPreset reconciliation")

		By("cleaning up")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, watchPluginDef)
	})

	It("should reflect a non-existing PluginDefinition in the PluginPreset status", func() {
		const nonExistingDefinitionName = "non-existing-plugindefinition"

		By("creating a PluginPreset referencing a non-existing ClusterPluginDefinition")
		pluginPreset := test.NewPluginPreset(pluginPresetName+"-missing-def", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}),
		)
		pluginPreset.Spec.Plugin.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: nonExistingDefinitionName,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed(), "failed to create PluginPreset")

		By("ensuring PluginFailedCondition is set due to missing PluginDefinition")
		Eventually(func(g Gomega) {
			presetKey := types.NamespacedName{Name: pluginPresetName + "-missing-def", Namespace: test.TestNamespace}
			g.Expect(test.K8sClient.Get(test.Ctx, presetKey, pluginPreset)).To(Succeed())
			pluginFailedCondition := pluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
			g.Expect(pluginFailedCondition).ToNot(BeNil(), "PluginFailedCondition should be set")
			g.Expect(pluginFailedCondition.Status).To(Equal(metav1.ConditionTrue), "PluginFailedCondition should be true")
			g.Expect(pluginFailedCondition.Message).To(ContainSubstring(nonExistingDefinitionName),
				"PluginFailedCondition message should reference the non-existing PluginDefinition")
		}).Should(Succeed(), "PluginPreset should reflect the missing PluginDefinition in its status")

		By("ensuring PluginDefinitionNotFoundCondition is set due to missing PluginDefinition")
		Eventually(func(g Gomega) {
			presetKey := types.NamespacedName{Name: pluginPresetName + "-missing-def", Namespace: test.TestNamespace}
			g.Expect(test.K8sClient.Get(test.Ctx, presetKey, pluginPreset)).To(Succeed())
			pdNotFoundCondition := pluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginDefinitionNotFoundCondition)
			g.Expect(pdNotFoundCondition).ToNot(BeNil(), "PluginDefinitionNotFoundCondition should be set")
			g.Expect(pdNotFoundCondition.IsTrue()).To(BeTrue(), "PluginDefinitionNotFoundCondition should be true")
			g.Expect(string(pdNotFoundCondition.Reason)).To(Equal(string(greenhousev1alpha1.PluginDefinitionNotFound)),
				"PluginDefinitionNotFoundCondition reason should be PluginDefinitionNotFound")
			g.Expect(pdNotFoundCondition.Message).To(ContainSubstring(nonExistingDefinitionName),
				"PluginDefinitionNotFoundCondition message should reference the non-existing PluginDefinition")
			g.Expect(pluginPreset.Status.PluginDefinitionVersion).To(BeEmpty(), "PluginDefinitionVersion should be empty when the PluginDefinition does not exist")
		}).Should(Succeed(), "PluginPreset should reflect the missing PluginDefinition via PluginDefinitionNotFoundCondition")

		By("ensuring ReadyCondition is False")
		Eventually(func(g Gomega) {
			presetKey := types.NamespacedName{Name: pluginPresetName + "-missing-def", Namespace: test.TestNamespace}
			g.Expect(test.K8sClient.Get(test.Ctx, presetKey, pluginPreset)).To(Succeed())
			readyCondition := pluginPreset.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set")
			g.Expect(readyCondition.IsFalse()).To(BeTrue(), "ReadyCondition should be false")
		}).Should(Succeed())

		By("removing plugin preset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should resolve a simple expression using clusterName", func() {
		By("creating a PluginPreset with an expression")
		expressionStr := `"app-${global.greenhouse.clusterName}.example.com"`
		presetPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName,
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:       "test.hostname",
					Expression: &expressionStr,
				},
			},
		}

		pluginPreset := test.NewPluginPreset("expr-simple", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(presetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed())

		By("ensuring Plugin has resolved expression value")
		expPluginName := types.NamespacedName{Name: "expr-simple-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ToNot(HaveOccurred(), "Plugin should exist")

			var hostnameFound bool
			for _, ov := range expPlugin.Spec.OptionValues {
				if ov.Name == "test.hostname" {
					hostnameFound = true
					g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
					g.Expect(string(ov.Value.Raw)).To(Equal(`"app-`+clusterA+`.example.com"`),
						"Expression should resolve with cluster name")
				}
			}
			g.Expect(hostnameFound).To(BeTrue(), "test.hostname should exist in Plugin")
		}).Should(Succeed())

		By("removing the PluginPreset")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should resolve expression with cluster metadata", func() {
		By("adding metadata labels to clusterA")
		clusterAObj := &greenhousev1alpha1.Cluster{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{
			Name: clusterA, Namespace: test.TestNamespace,
		}, clusterAObj)).To(Succeed())

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, clusterAObj, func() error {
			clusterAObj.Labels["metadata.greenhouse.sap/region"] = "eu-de-1"
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		By("creating a PluginPreset with metadata expression")
		expressionStr := `"service.${global.greenhouse.metadata.region}.example.com"`
		presetPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName,
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:       "test.serviceHost",
					Expression: &expressionStr,
				},
			},
		}

		pluginPreset := test.NewPluginPreset("expr-metadata", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(presetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed())

		By("ensuring Plugin has resolved metadata expression")
		expPluginName := types.NamespacedName{Name: "expr-metadata-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, ov := range expPlugin.Spec.OptionValues {
				if ov.Name == "test.serviceHost" {
					found = true
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal(`"service.eu-de-1.example.com"`))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())

		By("cleaning up metadata label")
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{
			Name: clusterA, Namespace: test.TestNamespace,
		}, clusterAObj)).To(Succeed())
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, clusterAObj, func() error {
			delete(clusterAObj.Labels, "metadata.greenhouse.sap/region")
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should keep direct values unchanged when resolving expressions", func() {
		expressionStr := `"generated-${global.greenhouse.clusterName}"`
		presetPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName,
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:  "direct.value",
					Value: test.MustReturnJSONFor("unchanged"),
				},
				{
					Name:       "expression.value",
					Expression: &expressionStr,
				},
			},
		}

		pluginPreset := test.NewPluginPreset("expr-mixed", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(presetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed())

		expPluginName := types.NamespacedName{Name: "expr-mixed-" + clusterA, Namespace: test.TestNamespace}
		expPlugin := &greenhousev1alpha1.Plugin{}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, expPluginName, expPlugin)
			g.Expect(err).ToNot(HaveOccurred())

			g.Expect(expPlugin.Spec.OptionValues).To(ContainElement(
				greenhousev1alpha1.PluginOptionValue{
					Name:  "direct.value",
					Value: test.MustReturnJSONFor("unchanged"),
				}), "Direct value should be unchanged")

			var exprResolved bool
			for _, ov := range expPlugin.Spec.OptionValues {
				if ov.Name == "expression.value" {
					exprResolved = true
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal(`"generated-` + clusterA + `"`))
				}
			}
			g.Expect(exprResolved).To(BeTrue())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should report error for invalid expression", func() {
		invalidExpressionStr := `"service.${global.greenhouse.nonexistent.field}.example.com"`
		presetPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName,
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:       "test.invalid",
					Expression: &invalidExpressionStr,
				},
			},
		}

		pluginPreset := test.NewPluginPreset("expr-invalid", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(presetPluginSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, pluginPreset)).To(Succeed())

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(pluginPreset), pluginPreset)
			g.Expect(err).ToNot(HaveOccurred())

			pluginFailedCondition := pluginPreset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
			g.Expect(pluginFailedCondition).ToNot(BeNil())
			g.Expect(pluginFailedCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(pluginFailedCondition.Message).To(ContainSubstring("failed to resolve"))
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, pluginPreset)
	})

	It("should return error when expression is set but ExpressionEvaluationEnabled is false", func() {
		reconciler := &PluginPresetReconciler{
			Client:                      test.K8sClient,
			ExpressionEvaluationEnabled: false,
		}

		expressionStr := `"app-${global.greenhouse.clusterName}.example.com"`
		preset := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flag-off-test",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
					OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
						{
							Name:  "direct.value",
							Value: test.MustReturnJSONFor("works"),
						},
						{
							Name:       "test.hostname",
							Expression: &expressionStr,
						},
					},
				},
			},
		}

		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: test.TestNamespace,
			},
		}

		_, err := reconciler.resolvePluginOptionValuesForPreset(test.Ctx, preset, cluster)
		Expect(err).To(HaveOccurred(), "should return error when expression exists but flag is disabled")
		Expect(err.Error()).To(ContainSubstring("expressionEvaluationEnabled"),
			"error should mention the flag")
		Expect(err.Error()).To(ContainSubstring("test.hostname"),
			"error should mention the option name")
	})

	It("should succeed when no expressions and flag is disabled", func() {
		reconciler := &PluginPresetReconciler{
			Client:                      test.K8sClient,
			ExpressionEvaluationEnabled: false,
		}

		preset := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flag-off-no-expr",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
					OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
						{
							Name:  "direct.value",
							Value: test.MustReturnJSONFor("works"),
						},
						{
							Name:  "another.value",
							Value: test.MustReturnJSONFor(42),
						},
					},
				},
			},
		}

		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: test.TestNamespace,
			},
		}

		result, err := reconciler.resolvePluginOptionValuesForPreset(test.Ctx, preset, cluster)
		Expect(err).ToNot(HaveOccurred(), "should succeed without expressions")
		Expect(result).To(HaveLen(2))
		Expect(result[0].Name).To(Equal("direct.value"))
		Expect(result[0].Value).To(Equal(test.MustReturnJSONFor("works")))
		Expect(result[1].Name).To(Equal("another.value"))
		Expect(result[1].Value).To(Equal(test.MustReturnJSONFor(42)))
	})

	It("`should resolve valueFrom.ref pointing to another PluginPreset with expression`", func() {
		By("creating source PluginPreset with expression")
		sourceExpressionStr := `"generated-${global.greenhouse.clusterName}"`
		srcPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-ref-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:       "source.value",
					Expression: &sourceExpressionStr,
				},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(srcPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin to be created with resolved expression")
		sourcePluginName := types.NamespacedName{Name: "ref-source-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			err := test.K8sClient.Get(test.Ctx, sourcePluginName, sourcePlugin)
			g.Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, ov := range sourcePlugin.Spec.OptionValues {
				if ov.Name == "source.value" {
					found = true
					g.Expect(ov.Expression).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal(`"generated-` + clusterA + `"`))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed(), "Source Plugin should have resolved expression")

		By("creating consumer PluginPreset that references source")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-ref-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name: "consumer.value",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-source",
							Expression: `${spec.optionValues.filter(v, v.name == "source.value")[0].value}`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has resolved reference")
		consumerPluginName := types.NamespacedName{Name: "ref-consumer-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			err := test.K8sClient.Get(test.Ctx, consumerPluginName, consumerPlugin)
			g.Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.value" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal(`"generated-` + clusterA + `"`))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed(), "Consumer Plugin should have resolved reference")

		By("verifying both values match")
		sourcePlugin := &greenhousev1alpha1.Plugin{}
		Expect(test.K8sClient.Get(test.Ctx, sourcePluginName, sourcePlugin)).To(Succeed())
		consumerPlugin := &greenhousev1alpha1.Plugin{}
		Expect(test.K8sClient.Get(test.Ctx, consumerPluginName, consumerPlugin)).To(Succeed())

		var sourceVal, consumerVal string
		for _, ov := range sourcePlugin.Spec.OptionValues {
			if ov.Name == "source.value" {
				sourceVal = string(ov.Value.Raw)
			}
		}
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == "consumer.value" {
				consumerVal = string(ov.Value.Raw)
			}
		}
		Expect(sourceVal).To(Equal(consumerVal), "Source and consumer values should match")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})

	It("should resolve valueFrom.ref with expression transformation", func() {
		By("creating source PluginPreset with expression")
		sourceExpressionStr := `"my-service.${global.greenhouse.clusterName}.example.com"`
		sourcePluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-transform-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name:       "service.hostname",
					Expression: &sourceExpressionStr,
				},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-transform-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(sourcePluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin to be created")
		sourcePluginName := types.NamespacedName{Name: "ref-transform-source-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, sourcePluginName, sourcePlugin)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer that transforms the referenced value")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-transform-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:  "myRequiredOption",
					Value: test.MustReturnJSONFor("myValue"),
				},
				{
					Name: "consumer.url",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-transform-source",
							Expression: `"https://" + spec.optionValues.filter(v, v.name == "service.hostname")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-transform-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": clusterA,
				},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has transformed value")
		consumerPluginName := types.NamespacedName{Name: "ref-transform-consumer-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, consumerPluginName, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.url" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal(
						`"https://my-service.` + clusterA + `.example.com"`))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed(), "Consumer should have transformed reference value")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})

	It("should resolve valueFrom.ref with selector pointing to multiple PluginPresets", func() {
		By("creating two source PluginPresets with selector label")
		sourceAExprStr := `"endpoint-a-${global.greenhouse.clusterName}"`
		sourceAPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-sel-a",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.endpoint", Expression: &sourceAExprStr},
			},
		}

		sourceAPreset := test.NewPluginPreset("sel-source-a", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetLabel("e2e.greenhouse.sap/selector-test", "true"),
			test.WithPresetPluginSpec(sourceAPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourceAPreset)).To(Succeed())

		sourceBExprStr := `"endpoint-b-${global.greenhouse.clusterName}"`
		sourceBPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-sel-b",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.endpoint", Expression: &sourceBExprStr},
			},
		}

		sourceBPreset := test.NewPluginPreset("sel-source-b", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPluginPresetLabel("e2e.greenhouse.sap/selector-test", "true"),
			test.WithPresetPluginSpec(sourceBPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourceBPreset)).To(Succeed())

		By("waiting for source Plugins")
		Eventually(func(g Gomega) {
			pluginA := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "sel-source-a-" + clusterA, Namespace: test.TestNamespace}, pluginA)).To(Succeed())
			pluginB := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "sel-source-b-" + clusterA, Namespace: test.TestNamespace}, pluginB)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer PluginPreset with selector reference")
		consumerSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-sel-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.endpoints",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind: greenhousev1alpha1.PluginPresetKind,
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"e2e.greenhouse.sap/selector-test": "true",
								},
							},
							Expression: `spec.optionValues.filter(v, v.name == "source.endpoint")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("sel-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has collected values from both sources")
		consumerPluginName := types.NamespacedName{Name: "sel-consumer-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, consumerPluginName, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.endpoints" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())

					var endpoints []any
					err := json.Unmarshal(ov.Value.Raw, &endpoints)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(endpoints).To(HaveLen(2))
					g.Expect(endpoints).To(ContainElement("endpoint-a-" + clusterA))
					g.Expect(endpoints).To(ContainElement("endpoint-b-" + clusterA))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed(), "Consumer should have collected values from both sources")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourceBPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourceAPreset)
	})

	It("should return empty array when selector matches no PluginPresets", func() {
		By("creating consumer PluginPreset with selector that matches nothing")
		consumerSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-empty-sel",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.value",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind: greenhousev1alpha1.PluginPresetKind,
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"non-existent-label": "true",
								},
							},
							Expression: `spec.optionValues[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("sel-empty", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin is created with empty array")
		consumerPluginName := types.NamespacedName{Name: "sel-empty-" + clusterA, Namespace: test.TestNamespace}
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, consumerPluginName, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.value" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
					g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
					g.Expect(string(ov.Value.Raw)).To(Equal("[]"),
						"Value should be empty array when no presets match selector")
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed(), "Consumer should handle empty selector result")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
	})

	It("should report error when referenced PluginPreset does not exist", func() {
		By("creating consumer PluginPreset referencing non-existent source")
		consumerSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-missing-ref",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.value",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "non-existent-preset",
							Expression: `spec.optionValues[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-missing", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring PluginPreset reports the error")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(consumerPreset), consumerPreset)
			g.Expect(err).ToNot(HaveOccurred())

			pluginFailedCondition := consumerPreset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
			g.Expect(pluginFailedCondition).ToNot(BeNil())
			g.Expect(pluginFailedCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(pluginFailedCondition.Message).To(ContainSubstring("non-existent-preset"))
		}).Should(Succeed(), "PluginPreset should report error for missing reference")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
	})

	It("should reject unsupported reference kind", func() {
		consumerSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-bad-kind",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.value",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       "Plugin",
							Name:       "some-plugin",
							Expression: `spec.optionValues[0].value`,
						},
					},
				},
			},
		}
		consumerPreset := test.NewPluginPreset("ref-bad-kind", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring PluginPreset reports unsupported kind error")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(consumerPreset), consumerPreset)
			g.Expect(err).ToNot(HaveOccurred())

			pluginFailedCondition := consumerPreset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
			g.Expect(pluginFailedCondition).ToNot(BeNil())
			g.Expect(pluginFailedCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(pluginFailedCondition.Message).To(ContainSubstring("unsupported reference kind"))
		}).Should(Succeed(), "PluginPreset should report error for unsupported kind")

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
	})

	It("should resolve valueFrom.ref returning integer value", func() {
		By("creating source PluginPreset with integer value")
		sourcePluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-int-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.replicas", Value: test.MustReturnJSONFor(3)},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-int-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(sourcePluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin")
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-int-source-" + clusterA, Namespace: test.TestNamespace}, sourcePlugin)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer that references integer value")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-int-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.replicas",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-int-source",
							Expression: `spec.optionValues.filter(v, v.name == "source.replicas")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-int-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has integer value")
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-int-consumer-" + clusterA, Namespace: test.TestNamespace}, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.replicas" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal("3"))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})

	It("should resolve valueFrom.ref returning boolean value", func() {
		By("creating source PluginPreset with boolean value")
		sourcePluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-bool-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.enabled", Value: test.MustReturnJSONFor(true)},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-bool-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(sourcePluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin")
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-bool-source-" + clusterA, Namespace: test.TestNamespace}, sourcePlugin)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer that references boolean value")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-bool-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.enabled",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-bool-source",
							Expression: `spec.optionValues.filter(v, v.name == "source.enabled")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-bool-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has boolean value")
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-bool-consumer-" + clusterA, Namespace: test.TestNamespace}, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.enabled" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())
					g.Expect(string(ov.Value.Raw)).To(Equal("true"))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})

	It("should resolve valueFrom.ref returning object/map value", func() {
		By("creating source PluginPreset with map value")
		sourcePluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-map-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.labels", Value: test.MustReturnJSONFor(map[string]string{
					"app":  "myapp",
					"team": "platform",
				})},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-map-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(sourcePluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin")
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-map-source-" + clusterA, Namespace: test.TestNamespace}, sourcePlugin)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer that references map value")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-map-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.labels",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-map-source",
							Expression: `spec.optionValues.filter(v, v.name == "source.labels")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-map-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has map value")
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-map-consumer-" + clusterA, Namespace: test.TestNamespace}, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.labels" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())

					var labels map[string]any
					err := json.Unmarshal(ov.Value.Raw, &labels)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(labels["app"]).To(Equal("myapp"))
					g.Expect(labels["team"]).To(Equal("platform"))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})

	It("should resolve valueFrom.ref returning array value", func() {
		By("creating source PluginPreset with array value")
		sourcePluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-arr-src",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{Name: "source.hosts", Value: test.MustReturnJSONFor([]string{"host-a.example.com", "host-b.example.com"})},
			},
		}

		sourcePreset := test.NewPluginPreset("ref-arr-source", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(sourcePluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, sourcePreset)).To(Succeed())

		By("waiting for source Plugin")
		Eventually(func(g Gomega) {
			sourcePlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-arr-source-" + clusterA, Namespace: test.TestNamespace}, sourcePlugin)).To(Succeed())
		}).Should(Succeed())

		By("creating consumer that references array value")
		consumerPluginPresetSpec := greenhousev1alpha1.PluginPresetPluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: pluginPresetDefinitionName,
			},
			ReleaseName:      releaseName + "-arr-consumer",
			ReleaseNamespace: releaseNamespace,
			OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "myRequiredOption", Value: test.MustReturnJSONFor("myValue")},
				{
					Name: "consumer.hosts",
					ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
						Ref: &greenhousev1alpha1.ExternalValueSource{
							Kind:       greenhousev1alpha1.PluginPresetKind,
							Name:       "ref-arr-source",
							Expression: `spec.optionValues.filter(v, v.name == "source.hosts")[0].value`,
						},
					},
				},
			},
		}

		consumerPreset := test.NewPluginPreset("ref-arr-consumer", test.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithPresetPluginSpec(consumerPluginPresetSpec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"cluster": clusterA},
			}))
		Expect(test.K8sClient.Create(test.Ctx, consumerPreset)).To(Succeed())

		By("ensuring consumer Plugin has array value")
		Eventually(func(g Gomega) {
			consumerPlugin := &greenhousev1alpha1.Plugin{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "ref-arr-consumer-" + clusterA, Namespace: test.TestNamespace}, consumerPlugin)).To(Succeed())

			var found bool
			for _, ov := range consumerPlugin.Spec.OptionValues {
				if ov.Name == "consumer.hosts" {
					found = true
					g.Expect(ov.ValueFrom).To(BeNil())
					g.Expect(ov.Value).ToNot(BeNil())

					var hosts []any
					err := json.Unmarshal(ov.Value.Raw, &hosts)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(hosts).To(HaveLen(2))
					g.Expect(hosts).To(ContainElement("host-a.example.com"))
					g.Expect(hosts).To(ContainElement("host-b.example.com"))
				}
			}
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, consumerPreset)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, sourcePreset)
	})
})

var _ = Describe("applyOverridesToPreset", func() {
	DescribeTable("test cases",
		func(preset *greenhousev1alpha1.PluginPreset, clusterName string, expectedOptionValues []greenhousev1alpha1.PluginPresetPluginOptionValue) {
			result := applyOverridesToPreset(preset, clusterName)
			Expect(result.Spec.Plugin.OptionValues).To(Equal(expectedOptionValues))
		},

		Entry("with no overrides defined",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
			},
		),

		Entry("with overrides for a different cluster",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterB,
							Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
								{Name: "option-1", Value: test.MustReturnJSONFor("overridden")},
							},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
			},
		),

		Entry("with overrides for matching cluster - replaces existing value",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor("original")},
							{Name: "option-2", Value: test.MustReturnJSONFor("unchanged")},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
								{Name: "option-1", Value: test.MustReturnJSONFor("overridden")},
							},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor("overridden")},
				{Name: "option-2", Value: test.MustReturnJSONFor("unchanged")},
			},
		),

		Entry("with overrides for matching cluster - appends new value",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
								{Name: "option-new", Value: test.MustReturnJSONFor("new-value")},
							},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor("value-1")},
				{Name: "option-new", Value: test.MustReturnJSONFor("new-value")},
			},
		),

		Entry("with multiple overrides - replaces and appends",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor(1)},
							{Name: "option-2", Value: test.MustReturnJSONFor(2)},
							{Name: "option-3", Value: test.MustReturnJSONFor(3)},
						},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
								{Name: "option-2", Value: test.MustReturnJSONFor(22)},
								{Name: "option-3", Value: test.MustReturnJSONFor(33)},
								{Name: "option-4", Value: test.MustReturnJSONFor(44)},
							},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor(1)},
				{Name: "option-2", Value: test.MustReturnJSONFor(22)},
				{Name: "option-3", Value: test.MustReturnJSONFor(33)},
				{Name: "option-4", Value: test.MustReturnJSONFor(44)},
			},
		),

		Entry("with empty option values and overrides adds values",
			&greenhousev1alpha1.PluginPreset{
				Spec: greenhousev1alpha1.PluginPresetSpec{
					Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
						OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{},
					},
					ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
						{
							ClusterName: clusterA,
							Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
								{Name: "option-1", Value: test.MustReturnJSONFor("added")},
							},
						},
					},
				},
			},
			clusterA,
			[]greenhousev1alpha1.PluginPresetPluginOptionValue{
				{Name: "option-1", Value: test.MustReturnJSONFor("added")},
			},
		),
	)

	It("should not mutate the original preset", func() {
		originalValue := test.MustReturnJSONFor("original")
		preset := &greenhousev1alpha1.PluginPreset{
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: greenhousev1alpha1.PluginPresetPluginSpec{
					OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
						{Name: "option-1", Value: originalValue},
					},
				},
				ClusterOptionOverrides: []greenhousev1alpha1.ClusterOptionOverride{
					{
						ClusterName: clusterA,
						Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
							{Name: "option-1", Value: test.MustReturnJSONFor("overridden")},
						},
					},
				},
			},
		}

		result := applyOverridesToPreset(preset, clusterA)

		// Result should have overridden value
		Expect(result.Spec.Plugin.OptionValues[0].Value).To(Equal(test.MustReturnJSONFor("overridden")))

		// Original preset should NOT be mutated
		Expect(preset.Spec.Plugin.OptionValues[0].Value).To(Equal(originalValue),
			"original preset should not be mutated by applyOverridesToPreset")
	})
})

var _ = Describe("getReleaseName", func() {
	It("returns plugin.Spec.ReleaseName if set", func() {
		plugin := &greenhousev1alpha1.Plugin{Spec: greenhousev1alpha1.PluginSpec{ReleaseName: "explicit-release"}}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginPresetPluginSpec{ReleaseName: "preset-release"}}}
		Expect(getReleaseName(plugin, preset)).To(Equal("explicit-release"))
	})

	It("returns plugin.Name if HelmReleaseStatus is set and ReleaseName is empty", func() {
		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: "plugin-name"},
			Spec:       greenhousev1alpha1.PluginSpec{ReleaseName: ""},
			Status:     greenhousev1alpha1.PluginStatus{HelmReleaseStatus: &greenhousev1alpha1.HelmReleaseStatus{}},
		}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginPresetPluginSpec{ReleaseName: "preset-release"}}}
		Expect(getReleaseName(plugin, preset)).To(Equal("plugin-name"))
	})

	It("returns preset.Spec.Plugin.ReleaseName if plugin.Spec.ReleaseName is empty and no HelmReleaseStatus", func() {
		plugin := &greenhousev1alpha1.Plugin{Spec: greenhousev1alpha1.PluginSpec{ReleaseName: ""}}
		preset := &greenhousev1alpha1.PluginPreset{Spec: greenhousev1alpha1.PluginPresetSpec{Plugin: greenhousev1alpha1.PluginPresetPluginSpec{ReleaseName: "preset-release"}}}
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
