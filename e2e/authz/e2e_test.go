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
	teamDemo         *greenhousev1alpha1.Team
	teamForbidden    *greenhousev1alpha1.Team
	pluginDefPodInfo *greenhousev1alpha1.PluginDefinition
	pluginDemo       *greenhousev1alpha1.Plugin
	pluginForbidden  *greenhousev1alpha1.Plugin
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
	teamDemo = test.NewTeam(ctx, "demo", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, teamDemo)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "should create Team demo")
	teamForbidden = test.NewTeam(ctx, "forbidden", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, teamForbidden)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "should create Team forbidden")

	By("onboarding remote cluster")
	shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, teamDemo.Name)

	By("creating plugindefinition-podinfo")
	pluginDefPodInfo = fixtures.PreparePodInfoPluginDefinition(ctx, env.TestNamespace, "6.10.1")
	err = adminClient.Create(ctx, pluginDefPodInfo)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "should create plugindefinition-podinfo")

	By("creating plugin-demo")
	pluginDemo = test.NewPlugin(ctx, "plugin-demo", env.TestNamespace,
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamDemo.Name),
		test.WithPluginDefinition(pluginDefPodInfo.Name),
		test.WithCluster(remoteClusterName),
	)
	err = adminClient.Create(ctx, pluginDemo)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "should create plugin-demo")
	test.EventuallyCreated(ctx, adminClient, pluginDemo)

	By("creating plugin-forbidden")
	pluginForbidden = test.NewPlugin(ctx, "plugin-forbidden", env.TestNamespace,
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamForbidden.Name),
		test.WithPluginDefinition(pluginDefPodInfo.Name),
		test.WithCluster(remoteClusterName),
	)
	err = adminClient.Create(ctx, pluginForbidden)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "should create plugin-forbidden")
	test.EventuallyCreated(ctx, adminClient, pluginForbidden)
})

var _ = AfterSuite(func() {
	By("cleaning up test resources")
	test.EventuallyDeleted(ctx, adminClient, pluginDemo)
	test.EventuallyDeleted(ctx, adminClient, pluginForbidden)
	test.EventuallyDeleted(ctx, adminClient, pluginDefPodInfo)

	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)

	test.EventuallyDeleted(ctx, adminClient, teamDemo)
	test.EventuallyDeleted(ctx, adminClient, teamForbidden)

	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
})

var _ = Describe("Authorization Webhook E2E", Ordered, func() {
	Describe("GET operations", func() {
		It("should allow demo-user to GET their team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			plugin := &greenhousev1alpha1.Plugin{}
			err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginDemo), plugin)
			Expect(err).ToNot(HaveOccurred(), "demo-user should be able to GET plugin-demo")
		})

		It("should deny demo-user to GET another team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			plugin := &greenhousev1alpha1.Plugin{}
			err = impClient.Get(ctx, client.ObjectKeyFromObject(pluginForbidden), plugin)
			Expect(err).To(HaveOccurred(), "demo-user should NOT be able to GET plugin-forbidden")
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
		})
	})

	Describe("UPDATE operations", func() {
		It("should allow demo-user to UPDATE their team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			plugin := &greenhousev1alpha1.Plugin{}
			err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginDemo), plugin)
			Expect(err).ToNot(HaveOccurred(), "should get current plugin")

			plugin.Spec.DisplayName = "plugin-demo updated"
			err = impClient.Update(ctx, plugin)
			Expect(err).ToNot(HaveOccurred(), "demo-user should be able to UPDATE plugin-demo")
		})

		It("should deny demo-user to UPDATE another team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			plugin := &greenhousev1alpha1.Plugin{}
			err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginForbidden), plugin)
			Expect(err).ToNot(HaveOccurred(), "should get current plugin")

			plugin.Spec.DisplayName = "plugin-forbidden updated"
			err = impClient.Update(ctx, plugin)
			Expect(err).To(HaveOccurred(), "demo-user should NOT be able to UPDATE plugin-forbidden")
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
		})
	})

	Describe("DELETE operations", func() {
		It("should allow demo-user to DELETE and recreate their team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			By("deleting the plugin")
			err = impClient.Delete(ctx, pluginDemo)
			Expect(err).ToNot(HaveOccurred(), "demo-user should be able to DELETE plugin-demo")

			By("waiting for plugin to be deleted")
			Eventually(func() bool {
				plugin := &greenhousev1alpha1.Plugin{}
				err := adminClient.Get(ctx, client.ObjectKeyFromObject(pluginDemo), plugin)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "plugin should be deleted")

			By("recreating the plugin")
			pluginDemo = test.NewPlugin(ctx, "plugin-demo", env.TestNamespace,
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamDemo.Name),
				test.WithPluginDefinition(pluginDefPodInfo.Name),
				test.WithCluster(remoteClusterName),
			)
			err = adminClient.Create(ctx, pluginDemo)
			Expect(err).ToNot(HaveOccurred(), "should recreate plugin-demo")

			By("waiting for plugin to be recreated")
			test.EventuallyCreated(ctx, adminClient, pluginDemo)
		})

		It("should deny demo-user to DELETE another team's plugin", func() {
			impClient, err := createImpersonatedClient("demo-user", []string{"support-group:demo"})
			Expect(err).ToNot(HaveOccurred(), "should create impersonated client")

			plugin := &greenhousev1alpha1.Plugin{}
			plugin.Name = pluginForbidden.Name
			plugin.Namespace = pluginForbidden.Namespace
			err = impClient.Delete(ctx, plugin)
			Expect(err).To(HaveOccurred(), "demo-user should NOT be able to DELETE plugin-forbidden")
			Expect(apierrors.IsForbidden(err)).To(BeTrue(), "error should indicate forbidden access")
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
