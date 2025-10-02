// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	certManagerPluginTest = "test-cert-manager-plugin"
)

func PluginControllerHelmRollback(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	By("Creating a plugin definition with cert-manager helm chart")
	pluginDefinition := fixtures.PrepareCertManagerPluginDefinition(env.TestNamespace)
	err := adminClient.Create(ctx, pluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).To(Succeed(), "there should be no error creating the plugin definition")

	By("Setting the HELM_RELEASE_TIMEOUT to 5 seconds")
	Expect(os.Setenv("HELM_RELEASE_TIMEOUT", "5")).To(Succeed(), "there should be no error setting HELM_RELEASE_TIMEOUT env")

	By("Preparing the plugin")
	// Creating plugin with release name
	plugin := fixtures.PreparePlugin(certManagerPluginTest, env.TestNamespace,
		test.WithClusterPluginDefinition(pluginDefinition.Name),
		test.WithCluster(remoteClusterName),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithReleaseName(certManagerPluginTest),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
	)

	By("Installing release manually on the remote cluster")
	_, err = helm.ExportInstallHelmRelease(ctx, adminClient, env.RemoteRestClientGetter, pluginDefinition.Spec, plugin, false)
	Expect(err).To(HaveOccurred(), "there should be an error installing the helm chart")

	By("Creating helm config")
	cfg, err := helm.ExportNewHelmAction(env.RemoteRestClientGetter, plugin.Spec.ReleaseNamespace)
	Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")

	By("Checking if the release has status set to failed")
	Eventually(func(g Gomega) {
		getAction := action.NewGet(cfg)
		helmRelease, err := getAction.Run(plugin.Name)
		g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
		g.Expect(helmRelease.Info.Status).To(Equal(release.StatusFailed), "helm release status should be set to failed")
		g.Expect(helmRelease.Version).To(Equal(1), "helm release version should equal 1")
	}).Should(Succeed(), "helm release should be set to failed")

	By("Setting the HELM_RELEASE_TIMEOUT to 5 minutes")
	Expect(os.Setenv("HELM_RELEASE_TIMEOUT", "300")).To(Succeed(), "there should be no error setting HELM_RELEASE_TIMEOUT env")

	By("Creating the plugin")
	Expect(adminClient.Create(ctx, plugin)).To(Succeed(), "there should be no error creating the plugin")

	By("Checking if the rollback took place")
	Eventually(func(g Gomega) {
		namespacedName := types.NamespacedName{Name: plugin.Name, Namespace: env.TestNamespace}
		err := adminClient.Get(ctx, namespacedName, plugin)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the plugin")
		g.Expect(plugin.Status.HelmReleaseStatus).ToNot(BeNil(), "plugin HelmReleaseStatus should not be nil")
		g.Expect(plugin.Status.HelmReleaseStatus.Status).To(Equal("deployed"), "helm release status should be set to deployed")
	}).Should(Succeed(), "plugin should show HelmReleaseStatus as deployed")

	By("Checking if the release has status set to deployed")
	Eventually(func(g Gomega) {
		getAction := action.NewGet(cfg)
		helmRelease, err := getAction.Run(plugin.Name)
		g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
		g.Expect(helmRelease.Info.Status).To(Equal(release.StatusDeployed), "helm release status should be set back to deployed")
		g.Expect(helmRelease.Version).To(Equal(3), "helm release version should change to 3")
	}).Should(Succeed(), "helm release status should be set back to deployed")

	By("Checking the helm release history for rollback")
	historyAction := action.NewHistory(cfg)
	releases, err := historyAction.Run(plugin.Name)
	Expect(err).ToNot(HaveOccurred(), "there should be no error getting helm release history")
	Expect(releases).To(HaveLen(3), "there should be exactly 3 entries in helm release history")
	secondReleaseInfo := releases[1].Info
	Expect(secondReleaseInfo.Description).To(Equal("Rollback to 1"), "release history should show Rollback to 1 in description")
	Expect(secondReleaseInfo.Status).To(Equal(release.StatusSuperseded), "release history should show Superseded status")
	thirdReleaseInfo := releases[2].Info
	Expect(thirdReleaseInfo.Status).To(Equal(release.StatusDeployed), "the last release entry should have status set to deployed")

	By("Deleting the plugin")
	test.EventuallyDeleted(ctx, adminClient, plugin)

	By("Check that the deployment is deleted")
	Eventually(func(g Gomega) {
		deployment := &appsv1.Deployment{}
		deployment.SetName(certManagerPluginTest)
		deployment.SetNamespace(env.TestNamespace)
		err := remoteClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the error should be NotFound error")
	}).Should(Succeed(), "the deployment should be deleted")

	By("Deleting the plugin definition")
	test.EventuallyDeleted(ctx, adminClient, pluginDefinition)
}
