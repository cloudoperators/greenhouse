// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// PluginImageReplication verifies that the Plugin controller pre-replicates container images via the registry mirror before applying the HelmRelease.
func PluginImageReplication(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName string) {
	pd := test.NewPluginDefinition(ctx, "podinfo-image-replication", env.TestNamespace,
		test.WithPluginDefinitionVersion("6.7.1"),
		test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "6.7.1",
		}),
	)

	By("creating PluginDefinition and waiting for chart replication")
	err := adminClient.Create(ctx, pd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, pd) })

	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue())
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationSucceededReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "chart must be replicated before Plugin creation")

	By("creating Plugin targeting the remote cluster")
	plugin := test.NewPlugin(ctx, "podinfo-image-replication", env.TestNamespace,
		test.WithPluginDefinition(pd.Name),
		test.WithCluster(remoteClusterName),
		test.WithReleaseName("podinfo-image-replication"),
		test.WithReleaseNamespace(env.TestNamespace),
	)
	err = adminClient.Create(ctx, plugin)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, plugin) })

	By("verifying image replication is recorded in Plugin status with upstream refs")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		g.Expect(plugin.Status.ImageReplication).NotTo(BeEmpty(),
			"Plugin controller should have replicated container images via the registry mirror")
		g.Expect(plugin.Status.ImageReplication).To(ContainElement(ContainSubstring("ghcr.io/stefanprodan/podinfo")),
			"imageReplication should record upstream ref, not the mirrored one")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying HelmReleaseCreated condition is True with no ImageReplicationFailed reason")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		cond := plugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmReleaseCreatedCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue())
		g.Expect(cond.Reason).NotTo(Equal(greenhousev1alpha1.ImageReplicationFailedReason),
			"HelmRelease creation must not be blocked by image replication failure")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying Plugin is Ready")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		g.Expect(plugin.Status.IsReadyTrue()).To(BeTrue())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying image replication is idempotent on re-reconciliation")
	Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
	originalImageReplication := append([]string(nil), plugin.Status.ImageReplication...)

	ann := plugin.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[lifecycle.ReconcileAnnotation] = time.Now().UTC().Format(time.RFC3339Nano)
	plugin.SetAnnotations(ann)
	Expect(adminClient.Update(ctx, plugin)).To(Succeed())

	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		g.Expect(plugin.Status.IsReadyTrue()).To(BeTrue())
		g.Expect(plugin.Status.ImageReplication).To(Equal(originalImageReplication),
			"re-reconciliation must not change the image replication list")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("verifying PostRenderer rewrote container images to baseDomain on remote cluster")
	Eventually(func(g Gomega) {
		podList := &corev1.PodList{}
		g.Expect(remoteClient.List(ctx, podList, client.InNamespace(env.TestNamespace))).To(Succeed())
		g.Expect(podList.Items).NotTo(BeEmpty(), "pods should be running on remote cluster")
		g.Expect(podList.Items).To(ContainElement(
			HaveField("Spec.Containers", ContainElement(
				HaveField("Image", HavePrefix("registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/podinfo")),
			)),
		), "PostRenderer should rewrite podinfo image to baseDomain/subPath/repo")
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())
}

// PluginImageReplicationFailure verifies that the Plugin controller sets HelmReleaseCreated=False with ImageReplicationFailed when the primaryMirror is unreachable.
func PluginImageReplicationFailure(ctx context.Context, adminClient client.Client, env *shared.TestEnv, remoteClusterName string) {
	DeferCleanup(func() {
		By("restoring ConfigMap to valid primaryMirror")
		shared.CreateMirrorConfigMap(ctx, adminClient, env.TestNamespace)
	})

	By("breaking primaryMirror in ConfigMap to simulate unreachable registry")
	mirrorCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-replication-config",
			Namespace: env.TestNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, adminClient, mirrorCM, func() error {
		mirrorCM.Data = map[string]string{
			"containerRegistryConfig": `primaryMirror: "broken-registry:9999"
registryMirrors:
  ghcr.io:
    baseDomain: "broken-cdn:9999"
    subPath: "greenhouse-ghcr-io-mirror"`,
		}
		return nil
	})
	Expect(err).ToNot(HaveOccurred())

	pd := test.NewPluginDefinition(ctx, "podinfo-image-failure", env.TestNamespace,
		test.WithPluginDefinitionVersion("6.7.1"),
		test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "podinfo",
			Repository: "oci://registry:5000/greenhouse-ghcr-io-mirror/stefanprodan/charts",
			Version:    "6.7.1",
		}),
	)

	By("creating PluginDefinition - registry:5000 does not match broken primaryMirror, replication not configured")
	err = adminClient.Create(ctx, pd)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, pd) })

	By("verifying OCIReplicationNotConfigured is True for the PluginDefinition")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		cond := pd.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeTrue())
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.OCIReplicationNotConfiguredReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	By("waiting for PluginDefinition Ready - Flux fetches chart directly from registry:5000")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pd), pd)).To(Succeed())
		g.Expect(pd.Status.IsReadyTrue()).To(BeTrue())
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed())

	plugin := test.NewPlugin(ctx, "podinfo-image-failure", env.TestNamespace,
		test.WithPluginDefinition(pd.Name),
		test.WithCluster(remoteClusterName),
		test.WithReleaseName("podinfo-image-failure"),
		test.WithReleaseNamespace(env.TestNamespace),
	)

	By("creating Plugin - image replication to broken-registry:9999 should fail")
	err = adminClient.Create(ctx, plugin)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, plugin) })

	By("verifying HelmReleaseCreated is False with ImageReplicationFailed reason")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)).To(Succeed())
		cond := plugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmReleaseCreatedCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.IsTrue()).To(BeFalse())
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha1.ImageReplicationFailedReason))
	}).WithTimeout(shared.OCIReplicationTimeout).Should(Succeed(), "Plugin controller should block HelmRelease creation on image replication failure")
}
