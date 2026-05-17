// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build clusterplugindefinitionE2E

package clusterplugindefinition

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/e2e/clusterplugindefinition/scenarios"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	testStartTime time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClusterPluginDefinition E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv()

	var err error
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")

	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	shared.SetupOCIMirroringForOrg(ctx, adminClient, env.TestNamespace)

	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
	env.GenerateFluxControllerLogs(ctx, "helm-controller", testStartTime)
	shared.TeardownOCIMirroringForOrg(ctx, adminClient, env.TestNamespace)
})

var _ = Describe("ClusterPluginDefinition E2E", Ordered, func() {
	It("should replicate helm chart to registry mirror", func() {
		scenarios.ClusterPDChartReplication(ctx, adminClient)
	})

	It("should fail chart replication for non-existent chart version", func() {
		scenarios.ClusterPDChartReplicationFailure(ctx, adminClient)
	})
})
