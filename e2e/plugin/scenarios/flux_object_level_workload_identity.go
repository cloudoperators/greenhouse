// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"encoding/base64"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// FluxObjectLevelWorkloadIdentity checks that an OIDC-onboarded cluster gets a Flux access
// ConfigMap and that its Plugin installs through it.
func FluxObjectLevelWorkloadIdentity(ctx context.Context, adminClient client.Client, env *shared.TestEnv, oidcClusterName, teamName string) {
	By("onboarding a remote cluster via OIDC")
	restClientGetter := clientutil.NewRestClientGetterFromBytes(env.RemoteKubeConfigBytes, env.TestNamespace)
	restConfig, err := restClientGetter.ToRESTConfig()
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote REST config")
	remoteCA := make([]byte, base64.StdEncoding.EncodedLen(len(restConfig.CAData)))
	base64.StdEncoding.Encode(remoteCA, restConfig.CAData)
	shared.OnboardRemoteOIDCCluster(ctx, adminClient, remoteCA, restConfig.Host, oidcClusterName, env.TestNamespace, teamName)

	By("checking the cluster resource is created and marked as OIDC")
	cluster := &greenhousev1alpha1.Cluster{}
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: oidcClusterName, Namespace: env.TestNamespace}, cluster)).To(Succeed())
		g.Expect(cluster.Annotations).To(HaveKeyWithValue(
			greenhouseapis.ClusterConnectivityAnnotation, greenhouseapis.ClusterConnectivityOIDC))
	}).Should(Succeed(), "the cluster should be onboarded as an OIDC cluster")

	By("checking the flux access ConfigMap is rendered from the cluster secret")
	configMap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: oidcClusterName, Namespace: env.TestNamespace}, configMap)).To(Succeed())
		g.Expect(configMap.Data).To(HaveKeyWithValue(fluxmeta.KubeConfigKeyProvider, "generic"))
		g.Expect(configMap.Data).To(HaveKeyWithValue(fluxmeta.KubeConfigKeyAddress, restConfig.Host))
		g.Expect(configMap.Data).To(HaveKeyWithValue(fluxmeta.KubeConfigKeyAudiences, greenhouseapis.OIDCAudience))
		g.Expect(configMap.Data).To(HaveKeyWithValue(fluxmeta.KubeConfigKeyServiceAccountName, oidcClusterName))
		g.Expect(configMap.Data).To(HaveKeyWithValue(fluxmeta.KubeConfigKeyCACert, string(restConfig.CAData)))
	}).Should(Succeed(), "the flux access ConfigMap should be rendered")

	By("creating a plugin definition and a plugin on the OIDC cluster")
	pluginDefinition := fixtures.PreparePodInfoClusterPluginDefinition(env.TestNamespace, "6.9.0")
	Expect(client.IgnoreAlreadyExists(adminClient.Create(ctx, pluginDefinition))).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, pluginDefinition) })

	plugin := test.NewPlugin(ctx, "podinfo-oidc", env.TestNamespace,
		test.WithClusterPluginDefinition(pluginDefinition.Name),
		test.WithCluster(oidcClusterName),
		test.WithReleaseName("podinfo-oidc"),
		test.WithReleaseNamespace(env.TestNamespace),
	)
	Expect(client.IgnoreAlreadyExists(adminClient.Create(ctx, plugin))).ToNot(HaveOccurred())
	DeferCleanup(func() { test.EventuallyDeleted(ctx, adminClient, plugin) })

	By("checking the HelmRelease authenticates via configMapRef instead of the kubeconfig secret")
	Eventually(func(g Gomega) {
		helmRelease := &helmv2.HelmRelease{}
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), helmRelease)).To(Succeed())
		g.Expect(helmRelease.Spec.KubeConfig).ToNot(BeNil(), "the HelmRelease must reference cluster access")
		g.Expect(helmRelease.Spec.KubeConfig.ConfigMapRef).ToNot(BeNil(), "an OIDC cluster must be reached via configMapRef")
		g.Expect(helmRelease.Spec.KubeConfig.ConfigMapRef.Name).To(Equal(oidcClusterName))
		g.Expect(helmRelease.Spec.KubeConfig.SecretRef).To(BeNil(), "secretRef must be cleared for OIDC clusters")
	}).Should(Succeed(), "the HelmRelease should use the flux access ConfigMap")

	By("checking helm-controller installs the release through the ConfigMap")
	Eventually(func(g Gomega) {
		helmRelease := &helmv2.HelmRelease{}
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), helmRelease)).To(Succeed())
		g.Expect(helmRelease.Status.ObservedGeneration).To(BeNumerically(">=", helmRelease.Generation))
		releaseReady := meta.FindStatusCondition(helmRelease.Status.Conditions, fluxmeta.ReadyCondition)
		g.Expect(releaseReady).ToNot(BeNil(), "HelmRelease Ready condition must be set")
		g.Expect(releaseReady.Status).To(Equal(metav1.ConditionTrue), "HelmRelease must become Ready")
	}).Should(Succeed(), "the release should install via ObjectLevelWorkloadIdentity")
}
