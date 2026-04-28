// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// PDChartReplication verifies that the PluginDefinition controller replicates a Helm chart OCI artifact via the registry mirror.
func PDChartReplication(ctx context.Context, adminClient client.Client, env *shared.TestEnv) {
	pd := test.NewPluginDefinition(ctx, "podinfo-chart-replication", env.TestNamespace,
		test.WithPluginDefinitionVersion("6.7.1"),
		test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "6.7.1",
		}),
	)

	By("creating PluginDefinition with chart URL at registry mirror")
	err := adminClient.Create(ctx, pd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, pd) })

	By("verifying OCIReplicationReady condition becomes True with OCIReplicationSucceeded")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil(), "OCIReplicationReady condition should be set")
		g.Expect(cond.IsTrue()).To(BeTrue(), "chart replication should succeed")
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationSucceededReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "registry mirror should pull-through the podinfo chart from ghcr.io")

	By("verifying LastSyncedArtifact is populated")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		artifact := pd.GetLastSyncedArtifact()
		g.Expect(artifact).NotTo(BeNil(), "LastSyncedArtifact should be set after replication")
		g.Expect(artifact.Registry).To(Equal("registry:5000"))
		g.Expect(artifact.ChartName).To(Equal("greenhouse-ghcr-io-mirror/stefanprodan/charts/podinfo"))
		g.Expect(artifact.Version).To(Equal("6.7.1"))
		g.Expect(artifact.Digest).NotTo(BeEmpty(), "digest should be populated after successful replication")
		g.Expect(artifact.ReplicationStatus).To(Equal(greenhousev1alpha1.ReplicationStatusReplicated))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying HelmChartReady condition becomes True")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue(), "Flux should fetch the chart from the registry mirror over HTTP")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying PluginDefinition is Ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		g.Expect(pd.Status.IsReadyTrue()).To(BeTrue())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying chart replication is idempotent on re-reconciliation")
	Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
	originalDigest := pd.GetLastSyncedArtifact().Digest

	ann := pd.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[lifecycle.ReconcileAnnotation] = time.Now().UTC().Format(time.RFC3339Nano)
	pd.SetAnnotations(ann)
	Expect(adminClient.Update(ctx, pd)).To(Succeed())

	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue())
		artifact := pd.GetLastSyncedArtifact()
		g.Expect(artifact).NotTo(BeNil())
		g.Expect(artifact.Digest).To(Equal(originalDigest), "re-reconciliation must not change the replicated artifact digest")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())
}

// PDChartReplicationFailure verifies that the PluginDefinition controller sets OCIReplicationFailed for a non-existent chart version.
func PDChartReplicationFailure(ctx context.Context, adminClient client.Client, env *shared.TestEnv) {
	pd := test.NewPluginDefinition(ctx, "podinfo-chart-failure", env.TestNamespace,
		test.WithPluginDefinitionVersion("99.99.99"),
		test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "99.99.99",
		}),
	)

	By("creating PluginDefinition with non-existent chart version")
	err := adminClient.Create(ctx, pd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, pd) })

	By("verifying OCIReplicationReady condition becomes False with OCIReplicationFailed reason")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeFalse())
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationFailedReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "registry mirror pull-through should fail for a non-existent chart version")

	By("verifying PluginDefinition is not Ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		g.Expect(pd.Status.IsReadyTrue()).To(BeFalse())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())
}
