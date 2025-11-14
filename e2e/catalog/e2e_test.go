// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build catalogE2E

package catalog

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/e2e/catalog/expect"
	"github.com/cloudoperators/greenhouse/e2e/catalog/scenarios"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

const (
	greenhouseOrgYaml       = "./testdata/greenhouse_organization.yaml"
	e2eOrgYaml              = "./testdata/catalog_e2e_organization.yaml"
	catalogBranchYaml       = "./testdata/catalog_scenario_branch.yaml"
	catalogCommitYaml       = "./testdata/catalog_scenario_commit.yaml"
	catalogCPDYaml          = "./testdata/catalog_scenario_cpd.yaml"
	catalogMultiYaml        = "./testdata/catalog_scenario_multi_source.yaml"
	catalogArtifactFailYaml = "./testdata/catalog_scenario_artifact_fail.yaml"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	testStartTime time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	var err error
	ctx = context.Background()
	env = shared.NewExecutionEnv()
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating a Team")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	expect.AllCatalogDeleted(ctx, adminClient)
	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
	env.GenerateFluxControllerLogs(ctx, "source-controller", testStartTime)
	env.GenerateFluxControllerLogs(ctx, "source-watcher", testStartTime)
	env.GenerateFluxControllerLogs(ctx, "kustomize-controller", testStartTime)
})

var _ = Describe("Catalog E2E", Ordered, func() {
	DescribeTable("Catalog scenarios",
		func(orgYamlPath, catalogYamlPath, secretName string, secretType shared.SecretType, execute func(scenarios.IScenario, string)) {
			env := env.WithOrganization(ctx, adminClient, orgYamlPath)
			if !env.IsRealCluster {
				env = env.WithGitHubSecret(ctx, adminClient, secretName, secretType)
			}
			testNamespace := env.TestNamespace
			scenario := scenarios.NewScenario(adminClient, catalogYamlPath)
			execute(scenario, testNamespace)
		},
		Entry("Catalog Branch scenario",
			e2eOrgYaml,
			catalogBranchYaml,
			"github-com-token",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteSuccessScenario(ctx, ns) },
		),
		Entry("Catalog Commit scenario",
			e2eOrgYaml,
			catalogCommitYaml,
			"github-com-token",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteSuccessScenario(ctx, ns) },
		),
		Entry("Catalog CPD scenario",
			greenhouseOrgYaml,
			catalogCPDYaml,
			"github-com-app",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteSuccessScenario(ctx, ns) },
		),
		Entry("Catalog Multi Source scenario",
			e2eOrgYaml,
			catalogMultiYaml,
			"github-com-app",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteSuccessScenario(ctx, ns) },
		),
		Entry("Catalog CPD Fail scenario",
			e2eOrgYaml,
			catalogCPDYaml,
			"github-com-app",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteCPDFailScenario(ctx, ns) },
		),
		Entry("Catalog Artifact Fail scenario",
			e2eOrgYaml,
			catalogArtifactFailYaml,
			"github-com-app",
			shared.GitHubSecretTypeAPP,
			func(s scenarios.IScenario, ns string) { s.ExecuteArtifactFailScenario(ctx, ns) },
		),
		Entry("Catalog Git Auth Fail scenario",
			e2eOrgYaml,
			catalogBranchYaml,
			"github-com-token",
			shared.GitHubSecretTypeFake,
			func(s scenarios.IScenario, ns string) { s.ExecuteGitAuthFailScenario(ctx, ns) },
		),
	)
})
