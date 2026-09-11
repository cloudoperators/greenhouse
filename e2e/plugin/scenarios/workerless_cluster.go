// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// WorkerlessClusterBlocksPlugin verifies that a Plugin targeting a workerless cluster has
// HelmReleaseCreated=False with reason ClusterPayloadNotSchedulable.
func WorkerlessClusterBlocksPlugin(ctx context.Context, adminClient client.Client, env *shared.TestEnv, workerlessClusterName, teamName string) {
	By("creating a ClusterPluginDefinition for podinfo")
	pluginDef := fixtures.PreparePodInfoClusterPluginDefinition(env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, pluginDef)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the plugin definition")

	By("verifying the plugin definition is ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pluginDef), pluginDef)).To(Succeed())
		g.Expect(pluginDef.Status.IsReadyTrue()).To(BeTrue(), "plugin definition should be ready")
	}).Should(Succeed(), "plugin definition should be ready")

	By("creating a Plugin targeting the workerless cluster")
	plugin := fixtures.PreparePlugin("test-workerless-plugin", env.TestNamespace,
		test.WithClusterPluginDefinition(pluginDef.Name),
		test.WithCluster(workerlessClusterName),
		test.WithReleaseName("test-workerless-plugin"),
		test.WithReleaseNamespace(env.TestNamespace),
	)
	err = adminClient.Create(ctx, plugin)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the plugin")

	By("verifying HelmReleaseCreated=False with ClusterPayloadNotSchedulable reason")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		helmReleaseCreated := plugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmReleaseCreatedCondition)
		g.Expect(helmReleaseCreated).ToNot(BeNil(), "Plugin HelmReleaseCreated condition must be set")
		g.Expect(helmReleaseCreated.Status).To(Equal(metav1.ConditionFalse), "Plugin HelmReleaseCreated condition must be false")
		g.Expect(helmReleaseCreated.Reason).To(Equal(greenhousev1alpha1.ClusterPayloadNotSchedulableReason),
			"Plugin HelmReleaseCreated reason must be ClusterPayloadNotSchedulable")
	}).Should(Succeed(), "Plugin should report HelmReleaseCreated=False/ClusterPayloadNotSchedulable")

	By("cleaning up the plugin")
	test.EventuallyDeleted(ctx, adminClient, plugin)
}
