// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func FluxControllerPodInfoByPlugin(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName string) {
	By("Creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoPluginDefinition(env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(greenhousemetav1alpha1.ReadyCondition),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Checking the helm repository is ready")
	Eventually(func(g Gomega) {
		helmRepository, err := flux.FindHelmRepositoryByURL(ctx, adminClient, testPluginDefinition.Spec.HelmChart.Repository, flux.HelmRepositoryDefaultNamespace)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(helmRepository.Spec.URL).To(Equal("oci://ghcr.io/stefanprodan/charts"), "the helm repository URL should match the expected value")
	}).Should(Succeed(), "the helm repository should eventually be created and ready")

	By("Prepare the plugin spec for preset")
	testPlugin := fixtures.PreparePlugin("test-podinfo-plugin", env.TestNamespace,
		test.WithClusterPluginDefinition(testPluginDefinition.Name),
		test.WithReleaseName("test-podinfo-plugin"),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}))

	By("Add labels to remote cluster")
	remoteCluster := &greenhousev1alpha1.Cluster{}
	err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
	Expect(err).ToNot(HaveOccurred())
	remoteCluster.Labels = map[string]string{
		"app": "test-cluster",
	}
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("Creating the plugin preset")
	testPluginPreset := &greenhousev1alpha1.PluginPreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-podinfo-plugin-preset",
			Namespace: env.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.GreenhouseHelmDeliveryToolLabel: greenhouseapis.GreenhouseHelmDeliveryToolFlux,
			},
			Annotations: map[string]string{
				lifecycle.PropagateLabelsAnnotation: greenhouseapis.GreenhouseHelmDeliveryToolLabel,
			},
		},
		Spec: greenhousev1alpha1.PluginPresetSpec{
			Plugin: testPlugin.Spec,
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-cluster",
				},
			},
		},
	}
	err = adminClient.Create(ctx, testPluginPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the plugin is created")
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.GreenhouseHelmDeliveryToolLabel: greenhouseapis.GreenhouseHelmDeliveryToolFlux})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1))
	}).Should(Succeed())

	helmRelease := &helmv2.HelmRelease{}
	By("Checking the HelmRelease is installed")
	Eventually(func(g Gomega) {
		helmReleaseList := &helmv2.HelmReleaseList{}
		err = adminClient.List(ctx, helmReleaseList, client.InNamespace(env.TestNamespace))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(helmReleaseList.Items).To(HaveLen(1))
		helmRelease = &helmReleaseList.Items[0]
		g.Expect(helmRelease.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(helmv2.ReleasedCondition),
			"Reason": Equal(helmv2.InstallSucceededReason),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Checking the deployment is created on the remote cluster")
	Eventually(func(g Gomega) {
		deploymentList := &appsv1.DeploymentList{}
		err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace), client.MatchingLabels{"helm.sh/chart": "podinfo-6.9.0"})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(deploymentList.Items).To(HaveLen(1), "there should be exactly one deployment")
		g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(1))), "the deployment should have 1 replica")
	}).Should(Succeed())

	By("Upgrading the plugin definition")
	updatedTestPluginDefinition := fixtures.PreparePodInfoPluginDefinition(env.TestNamespace, "6.9.2")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the podinfo plugin definition")
		testPluginDefinition.Spec = updatedTestPluginDefinition.Spec
		err = adminClient.Update(ctx, testPluginDefinition)
		g.Expect(err).To(Succeed(), "there should be no error updating the podinfo plugin definition")
	}).Should(Succeed(), "updating the plugin definition should eventually succeed")

	By("Checking the HelmRelease is upgraded")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(helmRelease), helmRelease)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(helmRelease.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(helmv2.ReleasedCondition),
			"Reason": Equal(helmv2.UpgradeSucceededReason),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Checking the deployment is updated on the remote cluster")
	Eventually(func(g Gomega) {
		deploymentList := &appsv1.DeploymentList{}
		err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace), client.MatchingLabels{"helm.sh/chart": "podinfo-6.9.2"})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(deploymentList.Items).To(HaveLen(1), "there should be exactly one deployment")
		g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(1))), "the deployment should have 1 replica")
	}).Should(Succeed())

	By("Deleting the plugin preset")
	test.EventuallyDeleted(ctx, adminClient, testPluginPreset)
	By("Verifying the HelmRelease is deleted")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(helmRelease), helmRelease)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "there should be a not found error getting the flux HelmRelease")
	}).Should(Succeed(), "the flux HelmRelease should eventually be deleted")
	By("Deleting the plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}

func FluxControllerPluginDependenciesOnPreset(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	By("Creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoPluginDefinition(env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(greenhousemetav1alpha1.ReadyCondition),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Checking the helm repository is ready")
	Eventually(func(g Gomega) {
		helmRepository, err := flux.FindHelmRepositoryByURL(ctx, adminClient, testPluginDefinition.Spec.HelmChart.Repository, flux.HelmRepositoryDefaultNamespace)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(helmRepository.Spec.URL).To(Equal("oci://ghcr.io/stefanprodan/charts"), "the helm repository URL should match the expected value")
	}).Should(Succeed(), "the helm repository should eventually be created and ready")

	By("Prepare the plugin spec for preset")
	testPlugin := fixtures.PreparePlugin("test-podinfo-plugin", env.TestNamespace,
		test.WithClusterPluginDefinition(testPluginDefinition.Name),
		test.WithReleaseName("test-podinfo-plugin"),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}))

	By("Add labels to remote cluster")
	remoteCluster := &greenhousev1alpha1.Cluster{}
	err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
	Expect(err).ToNot(HaveOccurred())
	remoteCluster.Labels = map[string]string{
		"app": "test-cluster",
	}
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	standalonePluginPresetName := "test-standalone-plugin-preset"
	standalonePluginPresetPluginName := standalonePluginPresetName + "-" + remoteClusterName

	By("Creating plugin preset dependent on Plugin from standalone preset")
	dependentPluginPreset := &greenhousev1alpha1.PluginPreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dependent-plugin-preset",
			Namespace: env.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.GreenhouseHelmDeliveryToolLabel: greenhouseapis.GreenhouseHelmDeliveryToolFlux,
				greenhouseapis.LabelKeyOwnedBy:                 teamName,
			},
			Annotations: map[string]string{
				lifecycle.PropagateLabelsAnnotation: greenhouseapis.GreenhouseHelmDeliveryToolLabel,
			},
		},
		Spec: greenhousev1alpha1.PluginPresetSpec{
			Plugin: testPlugin.Spec,
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-cluster",
				},
			},
			WaitFor: []greenhousev1alpha1.WaitForItem{
				{
					PluginRef: greenhousev1alpha1.PluginRef{
						PluginPreset: standalonePluginPresetName,
					},
				},
			},
		},
	}
	err = adminClient.Create(ctx, dependentPluginPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the dependent plugin is created with correctly resolved dependency")
	Eventually(func(g Gomega) {
		plugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, types.NamespacedName{Name: dependentPluginPreset.Name + "-" + remoteClusterName, Namespace: env.TestNamespace}, plugin)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(plugin.Spec.WaitFor).To(HaveLen(1))
		pluginRef := plugin.Spec.WaitFor[0]
		g.Expect(pluginRef.Name).To(Equal(standalonePluginPresetPluginName), "PluginRef on Plugin should be set to the name of the Plugin it depends on")
		g.Expect(pluginRef.PluginPreset).To(BeEmpty(), "PluginRef should have the PluginPreset name cleared after resolving")
	}).Should(Succeed())

	dependentHelmRelease := &helmv2.HelmRelease{}
	By("Checking the HelmRelease is created but Flux waits with installation")
	Eventually(func(g Gomega) {
		helmReleaseList := &helmv2.HelmReleaseList{}
		err = adminClient.List(ctx, helmReleaseList, client.InNamespace(env.TestNamespace))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(helmReleaseList.Items).To(HaveLen(1))
		dependentHelmRelease = &helmReleaseList.Items[0]
		g.Expect(dependentHelmRelease.Spec.DependsOn).To(HaveLen(1), "there should be only one dependency in HelmRelease")
		helmReleaseDependency := dependentHelmRelease.Spec.DependsOn[0]
		g.Expect(helmReleaseDependency.Name).To(Equal(standalonePluginPresetPluginName),
			"HelmRelease dependency should be set to a HelmRelease with the same name as the Plugin it depends on")
		g.Expect(dependentHelmRelease.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(fluxmeta.ReadyCondition),
			"Reason": Equal(helmv2.DependencyNotReadyReason),
			"Status": Equal(metav1.ConditionFalse),
		})))
	}).Should(Succeed())

	By("Creating standalone plugin preset")
	standalonePluginPreset := &greenhousev1alpha1.PluginPreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      standalonePluginPresetName,
			Namespace: env.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.GreenhouseHelmDeliveryToolLabel: greenhouseapis.GreenhouseHelmDeliveryToolFlux,
				greenhouseapis.LabelKeyOwnedBy:                 teamName,
			},
			Annotations: map[string]string{
				lifecycle.PropagateLabelsAnnotation: greenhouseapis.GreenhouseHelmDeliveryToolLabel,
			},
		},
		Spec: greenhousev1alpha1.PluginPresetSpec{
			Plugin: testPlugin.Spec,
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-cluster",
				},
			},
		},
	}
	err = adminClient.Create(ctx, standalonePluginPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the standalone plugin is created")
	Eventually(func(g Gomega) {
		plugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, types.NamespacedName{Name: standalonePluginPresetPluginName, Namespace: env.TestNamespace}, plugin)
		g.Expect(err).NotTo(HaveOccurred())
	}).Should(Succeed())

	standaloneHelmRelease := &helmv2.HelmRelease{}
	By("Checking the standalone HelmRelease is installed")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, types.NamespacedName{Name: standalonePluginPresetPluginName, Namespace: env.TestNamespace}, standaloneHelmRelease)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(standaloneHelmRelease.Spec.DependsOn).To(BeEmpty())
		g.Expect(standaloneHelmRelease.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(helmv2.ReleasedCondition),
			"Reason": Equal(helmv2.InstallSucceededReason),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Checking the dependent HelmRelease is upgraded")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(dependentHelmRelease), dependentHelmRelease)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(dependentHelmRelease.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(helmv2.ReleasedCondition),
			"Reason": Equal(helmv2.UpgradeSucceededReason),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())

	By("Deleting both plugin presets")
	test.EventuallyDeleted(ctx, adminClient, standalonePluginPreset)
	test.EventuallyDeleted(ctx, adminClient, dependentPluginPreset)
	By("Verifying both HelmReleases are deleted")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(standaloneHelmRelease), standaloneHelmRelease)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "there should be a not found error getting the standalone flux HelmRelease")
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(dependentHelmRelease), dependentHelmRelease)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "there should be a not found error getting the dependent flux HelmRelease")
	}).Should(Succeed(), "the flux HelmReleases should eventually be deleted")
	By("Deleting the plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
