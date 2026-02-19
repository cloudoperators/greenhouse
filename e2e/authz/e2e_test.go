// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build authzE2E

package authz

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/authz/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const remoteClusterName = "remote-authz-cluster"

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	remoteClient  client.Client
	testStartTime time.Time

	// Test resources
	teamObservability *greenhousev1alpha1.Team
	teamDemo          *greenhousev1alpha1.Team
	pluginDefPodInfo  *greenhousev1alpha1.PluginDefinition
	pluginPodInfoObs  *greenhousev1alpha1.Plugin
	pluginPodInfoDemo *greenhousev1alpha1.Plugin
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authorization E2E Suite")
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
	testStartTime = time.Now().UTC()

	By("creating test Teams")
	teamObservability = test.NewTeam(ctx, "observability", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, teamObservability)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "should create Team observability")
	}
	teamDemo = test.NewTeam(ctx, "demo", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, teamDemo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "should create Team demo")
	}

	By("onboarding remote cluster")
	shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, teamDemo.Name)

	By("creating plugindefinition-podinfo")
	pluginDefPodInfo = fixtures.PreparePodInfoPluginDefinition(ctx, env.TestNamespace, "6.10.1")
	err = adminClient.Create(ctx, pluginDefPodInfo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "should create plugindefinition-podinfo")
	}

	By("creating plugin-podinfo-obs")
	pluginPodInfoObs = test.NewPlugin(ctx, "podinfo-obs", env.TestNamespace,
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamObservability.Name),
		test.WithPluginDefinition(pluginDefPodInfo.Name),
		test.WithCluster(remoteClusterName),
	)
	err = adminClient.Create(ctx, pluginPodInfoObs)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "should create plugin-podinfo-obs")
	}
	test.EventuallyCreated(ctx, adminClient, pluginPodInfoObs)

	By("creating plugin-podinfo-demo")
	pluginPodInfoDemo = test.NewPlugin(ctx, "podinfo-demo", env.TestNamespace,
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamDemo.Name),
		test.WithPluginDefinition(pluginDefPodInfo.Name),
		test.WithCluster(remoteClusterName),
	)
	err = adminClient.Create(ctx, pluginPodInfoDemo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "should create plugin-podinfo-demo")
	}
	test.EventuallyCreated(ctx, adminClient, pluginPodInfoDemo)
})

var _ = AfterSuite(func() {
	By("cleaning up test resources")
	if pluginPodInfoDemo != nil {
		test.EventuallyDeleted(ctx, adminClient, pluginPodInfoDemo)
	}
	if pluginPodInfoObs != nil {
		test.EventuallyDeleted(ctx, adminClient, pluginPodInfoObs)
	}
	if pluginDefPodInfo != nil {
		test.EventuallyDeleted(ctx, adminClient, pluginDefPodInfo)
	}

	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)

	if teamObservability != nil {
		test.EventuallyDeleted(ctx, adminClient, teamObservability)
	}
	if teamDemo != nil {
		test.EventuallyDeleted(ctx, adminClient, teamDemo)
	}

	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
})

var _ = Describe("Authorization Webhook E2E", Ordered, func() {
	Describe("GET operations", func() {
		Context("when user has matching support-group", func() {
			It("should allow user1 to GET podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoObs), plugin)
				Expect(err).ToNot(HaveOccurred(), "user1 with support-group:observability should be able to GET podinfo-obs")
			})

			It("should allow user2 to GET podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoDemo), plugin)
				Expect(err).ToNot(HaveOccurred(), "user2 with support-group:demo should be able to GET podinfo-demo")
			})
		})

		Context("when user does not have matching support-group", func() {
			It("should deny user1 to GET podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoDemo), plugin)
				Expect(err).To(HaveOccurred(), "user1 with support-group:observability should NOT be able to GET podinfo-demo")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})

			It("should deny user2 to GET podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoObs), plugin)
				Expect(err).To(HaveOccurred(), "user2 with support-group:demo should NOT be able to GET podinfo-obs")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})
		})
	})

	Describe("UPDATE operations", func() {
		Context("when user has matching support-group", func() {
			It("should allow user1 to UPDATE podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoObs), plugin)
				Expect(err).ToNot(HaveOccurred(), "should get current plugin")

				plugin.Spec.DisplayName = "podinfo observability updated"
				err = impClient.Update(ctx, plugin)
				Expect(err).ToNot(HaveOccurred(), "user1 with support-group:observability should be able to UPDATE podinfo-obs")
			})

			It("should allow user2 to UPDATE podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoDemo), plugin)
				Expect(err).ToNot(HaveOccurred(), "should get current plugin")

				plugin.Spec.DisplayName = "podinfo demo updated"
				err = impClient.Update(ctx, plugin)
				Expect(err).ToNot(HaveOccurred(), "user2 with support-group:demo should be able to UPDATE podinfo-demo")
			})
		})

		Context("when user does not have matching support-group", func() {
			It("should deny user1 to UPDATE podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoDemo), plugin)
				Expect(err).ToNot(HaveOccurred(), "should get current plugin")

				plugin.Spec.DisplayName = "podinfo demo updated"
				err = impClient.Update(ctx, plugin)
				Expect(err).To(HaveOccurred(), "user1 with support-group:observability should NOT be able to UPDATE podinfo-demo")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})

			It("should deny user2 to UPDATE podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoObs), plugin)
				Expect(err).ToNot(HaveOccurred(), "should get current plugin")

				plugin.Spec.DisplayName = "podinfo observability updated"
				err = impClient.Update(ctx, plugin)
				Expect(err).To(HaveOccurred(), "user2 with support-group:demo should NOT be able to UPDATE podinfo-obs")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})
		})
	})

	Describe("DELETE operations", func() {
		Context("when user has matching support-group", func() {
			It("should allow user1 to DELETE and recreate podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				By("deleting the plugin")
				err = impClient.Delete(ctx, pluginPodInfoObs)
				Expect(err).ToNot(HaveOccurred(), "user1 with support-group:observability should be able to DELETE podinfo-obs")

				By("waiting for plugin to be deleted")
				Eventually(func(g Gomega) {
					plugin := &greenhousev1alpha1.Plugin{}
					err := adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoObs), plugin)
					g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "plugin should be deleted")
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				By("recreating the plugin")
				pluginPodInfoObs = test.NewPlugin(ctx, "podinfo-obs", env.TestNamespace,
					test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamObservability.Name),
					test.WithPluginDefinition(pluginDefPodInfo.Name),
					test.WithCluster(remoteClusterName),
				)
				err = adminClient.Create(ctx, pluginPodInfoObs)
				Expect(err).ToNot(HaveOccurred(), "should recreate plugin-podinfo-obs")

				By("waiting for plugin to be recreated")
				test.EventuallyCreated(ctx, adminClient, pluginPodInfoObs)
			})

			It("should allow user2 to DELETE and recreate podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				By("deleting the plugin")
				err = impClient.Delete(ctx, pluginPodInfoDemo)
				Expect(err).ToNot(HaveOccurred(), "user2 with support-group:demo should be able to DELETE podinfo-demo")

				By("waiting for plugin to be deleted")
				Eventually(func(g Gomega) {
					plugin := &greenhousev1alpha1.Plugin{}
					err := adminClient.Get(ctx, client.ObjectKeyFromObject(pluginPodInfoDemo), plugin)
					g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "plugin should be deleted")
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				By("recreating the plugin")
				pluginPodInfoDemo = test.NewPlugin(ctx, "podinfo-demo", env.TestNamespace,
					test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamDemo.Name),
					test.WithPluginDefinition(pluginDefPodInfo.Name),
					test.WithCluster(remoteClusterName),
				)
				err = adminClient.Create(ctx, pluginPodInfoDemo)
				Expect(err).ToNot(HaveOccurred(), "should recreate plugin-podinfo-demo")

				By("waiting for plugin to be recreated")
				test.EventuallyCreated(ctx, adminClient, pluginPodInfoDemo)
			})
		})

		Context("when user does not have matching support-group", func() {
			It("should deny user1 to DELETE podinfo-demo plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user1", []string{"support-group:observability"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				plugin.Name = pluginPodInfoDemo.Name
				plugin.Namespace = pluginPodInfoDemo.Namespace
				err = impClient.Delete(ctx, plugin)
				Expect(err).To(HaveOccurred(), "user1 with support-group:observability should NOT be able to DELETE podinfo-demo")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})

			It("should deny user2 to DELETE podinfo-obs plugin", func() {
				impClient, err := createImpersonatedClient("my-test-user2", []string{"support-group:demo"})
				Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

				plugin := &greenhousev1alpha1.Plugin{}
				plugin.Name = pluginPodInfoObs.Name
				plugin.Namespace = pluginPodInfoObs.Namespace
				err = impClient.Delete(ctx, plugin)
				Expect(err).To(HaveOccurred(), "user2 with support-group:demo should NOT be able to DELETE podinfo-obs")
				Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
			})
		})
	})
})

// createImpersonatedClient creates a Kubernetes client with user impersonation
func createImpersonatedClient(user string, groups []string) (client.Client, error) {
	config, err := env.AdminRestClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	// Configure impersonation
	impersonatedConfig := rest.CopyConfig(config)
	impersonatedConfig.Impersonate = rest.ImpersonationConfig{
		UserName: user,
		Groups:   groups,
	}

	// Create a new client with the impersonated config
	scheme, err := greenhousev1alpha1.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	return client.New(impersonatedConfig, client.Options{Scheme: scheme})
}
