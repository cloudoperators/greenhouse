// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build teamrolebindingE2E

package teamrolebinding

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/e2e/teamrolebinding/scenarios"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	remoteClusterName = "remote-trb-cluster"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	remoteClient  client.Client
	testStartTime time.Time

	teamAlpha *greenhousev1alpha1.Team
	teamBeta  *greenhousev1alpha1.Team
	teamRole  *greenhousev1alpha1.TeamRole

	trb scenarios.ITRBScenario
)

func TestTeamRoleBindingE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TeamRoleBinding E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()

	var err error
	env = shared.NewExecutionEnv()
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")

	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")

	By("creating shared Teams")
	teamAlpha = test.NewTeam(ctx, "trb-team-alpha", env.TestNamespace,
		test.WithMappedIDPGroup("idp-group-trb-alpha"),
		test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"),
	)
	err = adminClient.Create(ctx, teamAlpha)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating teamAlpha")

	teamBeta = test.NewTeam(ctx, "trb-team-beta", env.TestNamespace,
		test.WithMappedIDPGroup("idp-group-trb-beta"),
		test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"),
	)
	err = adminClient.Create(ctx, teamBeta)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating teamBeta")

	By("onboarding the remote cluster")
	shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, teamAlpha.Name)
	shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)

	By("creating shared TeamRole")
	teamRole = test.NewTeamRole(ctx, "trb-test-role", env.TestNamespace)
	err = adminClient.Create(ctx, teamRole)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating teamRole")

	By("building scenario runner")
	trb = scenarios.NewScenario(
		adminClient, remoteClient,
		env.TestNamespace, remoteClusterName,
		teamAlpha, teamBeta,
		teamRole,
	)

	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	By("deleting shared resources")
	test.EventuallyDeleted(ctx, adminClient, teamRole)
	test.EventuallyDeleted(ctx, adminClient, teamAlpha)
	test.EventuallyDeleted(ctx, adminClient, teamBeta)

	By("off-boarding the remote cluster")
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)

	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
})

var _ = Describe("TeamRoleBinding E2E", Ordered, func() {
	DescribeTable("TeamRoleBinding scenarios",
		func(execute func(scenarios.ITRBScenario, context.Context)) {
			execute(trb, ctx)
		},
		Entry("Single teamRef (baseline)",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteSingleTeamRefScenario(c) },
		),
		Entry("Multiple teamRefs",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteMultipleTeamRefsScenario(c) },
		),
		Entry("Deprecated teamRef migration",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteDeprecatedTeamRefMigrationScenario(c) },
		),
		Entry("Mutation of teamRefs (add/remove)",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteTeamRefsMutationScenario(c) },
		),
		Entry("Partial failure (some teams missing)",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecutePartialFailureScenario(c) },
		),
		Entry("Namespace creation with multiple teams",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteNamespaceCreationScenario(c) },
		),
		Entry("Cluster selector",
			func(s scenarios.ITRBScenario, c context.Context) { s.ExecuteClusterSelectorScenario(c) },
		),
	)
})
