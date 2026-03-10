// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build pluginE2E

package plugin

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/cluster/expect"
	"github.com/cloudoperators/greenhouse/e2e/plugin/scenarios"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	remoteClusterName                = "remote-plugin-cluster"
	remoteIntegrationCluster         = "remote-integration-cluster"
	remoteOIDCClusterRoleBindingName = "oidc-plugin-cluster-role-binding"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	remoteClient  client.Client
	testStartTime time.Time
	team          *greenhousev1alpha1.Team
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv()

	var err error
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	team = test.NewTeam(ctx, "test-plugin-e2e-team", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, team)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating a Team")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteIntegrationCluster, env.TestNamespace)
	test.EventuallyDeleted(ctx, adminClient, team)
	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
	env.GenerateFluxControllerLogs(ctx, "helm-controller", testStartTime)
})

var _ = Describe("Plugin E2E", Ordered, func() {
	It("should onboard remote cluster", func() {
		By("onboarding remote cluster")
		shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, team.Name)
	})
	It("should have a cluster resource created", func() {
		By("verifying if the cluster resource is created")
		Eventually(func(g Gomega) {
			err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
			g.Expect(err).ToNot(HaveOccurred())
		}).Should(Succeed(), "cluster resource should be created")

		By("verifying the cluster status is ready")
		shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
	})

	It("should deploy the plugin with flux", func() {
		scenarios.FluxControllerPodInfoByPlugin(ctx, adminClient, remoteClient, env, remoteClusterName)
	})

	It("should reconcile the UI-only plugin with flux", func() {
		scenarios.FluxControllerUIOnlyPlugin(ctx, adminClient, env, remoteClusterName)
	})

	It("should deploy plugin with dependency via flux", func() {
		scenarios.FluxControllerPluginDependencies(ctx, adminClient, env, remoteClusterName, team.Name)
	})

	It("should retain the helm release when Plugin `.spec.deletePolicy` is set to `Retain`", func() {
		scenarios.FluxControllerPluginDeletePolicyRetain(ctx, adminClient, env, remoteClusterName, team.Name)
	})

	It("should resolve option values from direct plugin reference", func() {
		By("setting up cluster role binding for OIDC on remote cluster")
		expect.SetupOIDCClusterRoleBinding(ctx, remoteClient, remoteOIDCClusterRoleBindingName, remoteIntegrationCluster, env.TestNamespace)

		By("onboarding remote cluster")
		restClient := clientutil.NewRestClientGetterFromBytes(env.RemoteKubeConfigBytes, env.TestNamespace)
		restConfig, err := restClient.ToRESTConfig()
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the remote REST config")
		remoteAPIServerURL := restConfig.Host
		remoteCA := make([]byte, base64.StdEncoding.EncodedLen(len(restConfig.CAData)))
		base64.StdEncoding.Encode(remoteCA, restConfig.CAData)
		shared.OnboardRemoteOIDCCluster(ctx, adminClient, remoteCA, remoteAPIServerURL, remoteIntegrationCluster, env.TestNamespace, team.Name)

		By("verifying the cluster status is ready")
		shared.ClusterIsReady(ctx, adminClient, remoteIntegrationCluster, env.TestNamespace)

		By("executing the plugin integration scenario with direct plugin reference")
		scenarios.PluginIntegrationByDirectReference(ctx, adminClient, remoteClient, env, remoteIntegrationCluster)
	})

	It("should resolve option values from plugin reference by label selector", func() {
		By("executing the plugin integration scenario with plugin reference by label selector")
		scenarios.PluginIntegrationBySelector(ctx, adminClient, remoteClient, env, remoteIntegrationCluster)
	})
})
