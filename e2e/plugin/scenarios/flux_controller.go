// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	By("Prepare the plugin")
	testPlugin := fixtures.PreparePlugin("test-podinfo-plugin", env.TestNamespace,
		test.WithPluginDefinition(testPluginDefinition.Name),
		test.WithReleaseName("test-podinfo-plugin"),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil))

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
		g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
	}).Should(Succeed())

	helmRelease := &helmv2.HelmRelease{}
	By("Checking the HelmRelease is installed")
	Eventually(func(g Gomega) {
		helmReleaseList := &helmv2.HelmReleaseList{}
		err = adminClient.List(ctx, helmReleaseList, client.InNamespace(env.TestNamespace))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(helmReleaseList.Items)).To(BeEquivalentTo(1))
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
		g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1), "there should be exactly one deployment")
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
		g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1), "there should be exactly one deployment")
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
