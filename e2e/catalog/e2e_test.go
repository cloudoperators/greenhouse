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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/catalog/expect"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	catalogRepositoryPatch = "oci://docker.io/greenhouse-extensions/charts"
	e2eCatalogName         = "greenhouse-catalog-e2e"
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
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating a Team")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	expect.CatalogDeleted(ctx, adminClient, env.TestNamespace, e2eCatalogName)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Catalog E2E", Ordered, func() {
	var catalog *greenhousev1alpha1.Catalog
	var pluginDefinitions *greenhousev1alpha1.PluginDefinitionList
	It("should successfully create a Catalog resource", func() {
		source := test.NewCatalogSource(
			test.WithRepositoryBranch("main"),
			test.WithRepository("https://github.com/cloudoperators/greenhouse-extensions"),
			test.WithCatalogResources([]string{
				"perses/plugindefinition.yaml",
				"kube-monitoring/plugindefinition.yaml",
			}))
		catalog = test.NewCatalog(
			e2eCatalogName,
			env.TestNamespace,
			source,
		)
		Expect(adminClient.Create(ctx, catalog)).To(Succeed(), "there should be no error creating the Catalog resource")
		expect.CatalogReady(ctx, adminClient, env.TestNamespace, catalog.Name)
		pluginDefinitions = &greenhousev1alpha1.PluginDefinitionList{}
		Expect(adminClient.List(ctx, pluginDefinitions)).To(Succeed(), "there should be no error listing pluginDefinitions")
		Expect(pluginDefinitions.Items).ToNot(BeEmpty(), "there should be at least one pluginDefinition created from the Catalog")
		Expect(pluginDefinitions.Items).To(HaveLen(2), "there should be exactly two pluginDefinitions created from the Catalog")
	})
})
