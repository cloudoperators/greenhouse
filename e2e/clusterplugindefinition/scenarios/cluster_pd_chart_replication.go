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

// ClusterPDChartReplication verifies that the ClusterPluginDefinition controller replicates a Helm chart OCI artifact via the registry mirror configured on the greenhouse Organization.
func ClusterPDChartReplication(ctx context.Context, adminClient client.Client) {
	cpd := test.NewClusterPluginDefinition(ctx, "podinfo-cluster-chart-replication",
		test.WithVersion("6.7.1"),
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "6.7.1",
		}),
	)

	By("creating ClusterPluginDefinition with chart URL at registry mirror")
	err := adminClient.Create(ctx, cpd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, cpd) })

	By("verifying OCIReplicationReady condition becomes True with OCIReplicationSucceeded")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		cond := cpd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil(), "OCIReplicationReady condition should be set")
		g.Expect(cond.IsTrue()).To(BeTrue(), "chart replication should succeed")
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationSucceededReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "registry mirror should pull-through the podinfo chart from ghcr.io")

	By("verifying LastSyncedArtifact is populated")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		artifact := cpd.GetLastSyncedArtifact()
		g.Expect(artifact).NotTo(BeNil(), "LastSyncedArtifact should be set after replication")
		g.Expect(artifact.Registry).To(Equal("registry:5000"))
		g.Expect(artifact.ChartName).To(Equal("greenhouse-ghcr-io-mirror/stefanprodan/charts/podinfo"))
		g.Expect(artifact.Version).To(Equal("6.7.1"))
		g.Expect(artifact.Digest).NotTo(BeEmpty(), "digest should be populated after successful replication")
		g.Expect(artifact.ReplicationStatus).To(Equal(greenhousev1alpha1.ReplicationStatusReplicated))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying HelmChartReady condition becomes True")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		cond := cpd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue(), "Flux should fetch the chart from the registry mirror over HTTP")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying ClusterPluginDefinition is Ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		g.Expect(cpd.Status.IsReadyTrue()).To(BeTrue())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying chart replication is idempotent on re-reconciliation")
	Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
	originalDigest := cpd.GetLastSyncedArtifact().Digest

	ann := cpd.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[lifecycle.ReconcileAnnotation] = time.Now().UTC().Format(time.RFC3339Nano)
	cpd.SetAnnotations(ann)
	Expect(adminClient.Update(ctx, cpd)).To(Succeed())

	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		cond := cpd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue())
		artifact := cpd.GetLastSyncedArtifact()
		g.Expect(artifact).NotTo(BeNil())
		g.Expect(artifact.Digest).To(Equal(originalDigest), "re-reconciliation must not change the replicated artifact digest")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())
}

// ClusterPDChartReplicationFailure verifies that the ClusterPluginDefinition controller sets OCIReplicationFailed for a non-existent chart version.
func ClusterPDChartReplicationFailure(ctx context.Context, adminClient client.Client) {
	cpd := test.NewClusterPluginDefinition(ctx, "podinfo-cluster-chart-failure",
		test.WithVersion("99.99.99"),
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "99.99.99",
		}),
	)

	By("creating ClusterPluginDefinition with non-existent chart version")
	err := adminClient.Create(ctx, cpd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, cpd) })

	By("verifying OCIReplicationReady condition becomes False with OCIReplicationFailed reason")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		cond := cpd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeFalse())
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationFailedReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "registry mirror pull-through should fail for a non-existent chart version")

	By("verifying ClusterPluginDefinition is not Ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cpd), cpd)).To(Succeed())
		g.Expect(cpd.Status.IsReadyTrue()).To(BeFalse())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())
}
