// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build catalogE2E

package catalog

import (
	"context"
	"fmt"
	"math/rand"
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
	var clusterPluginDefinitions *greenhousev1alpha1.ClusterPluginDefinitionList
	It("should successfully create a Catalog resource", func() {
		catalog = test.NewCatalog(
			e2eCatalogName,
			env.TestNamespace,
			test.WithRepositoryBranch("main"),
			test.WithRepositoryURL("https://github.com/cloudoperators/greenhouse-extensions"),
		)
		Expect(adminClient.Create(ctx, catalog)).To(Succeed(), "there should be no error creating the Catalog resource")
		expect.CatalogReady(ctx, adminClient, env.TestNamespace, catalog.Name)
		clusterPluginDefinitions = &greenhousev1alpha1.ClusterPluginDefinitionList{}
		Expect(adminClient.List(ctx, clusterPluginDefinitions)).To(Succeed(), "there should be no error listing ClusterPluginDefinitions")
		Expect(clusterPluginDefinitions.Items).ToNot(BeEmpty(), "there should be at least one ClusterPluginDefinition created from the Catalog")
	})
	It("should successfully patch a Catalog resource", func() {
		randomIndex := rand.Intn(len(clusterPluginDefinitions.Items))
		alias := clusterPluginDefinitions.Items[randomIndex].Name + fmt.Sprintf("-e2e-alias-%d", randomIndex)
		override := greenhousev1alpha1.CatalogOverrides{
			Name:  clusterPluginDefinitions.Items[randomIndex].Name,
			Alias: alias,
		}
		expect.CatalogOverride(ctx, adminClient, override, env.TestNamespace, catalog.Name)
		expect.CatalogReady(ctx, adminClient, env.TestNamespace, catalog.Name)

		Eventually(func(g Gomega) {
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err := adminClient.Get(ctx, client.ObjectKey{Name: alias}, clusterPluginDefinition)
			g.Expect(err).NotTo(HaveOccurred(), "there should be no error fetching the aliased ClusterPluginDefinition")
		}).Should(Succeed(), "the patched Catalog resource should be available with the new alias")

		override.Repository = catalogRepositoryPatch
		expect.CatalogOverride(ctx, adminClient, override, env.TestNamespace, catalog.Name)
		expect.CatalogReady(ctx, adminClient, env.TestNamespace, catalog.Name)

		Eventually(func(g Gomega) {
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err := adminClient.Get(ctx, client.ObjectKey{Name: alias}, clusterPluginDefinition)
			g.Expect(err).NotTo(HaveOccurred(), "there should be no error fetching the aliased ClusterPluginDefinition")
			g.Expect(clusterPluginDefinition.Spec.HelmChart.Repository).To(Equal(catalogRepositoryPatch), "the patched ClusterPluginDefinition should have the new repository URL")
		}).Should(Succeed(), "the patched Catalog resource should be available with the new repository URL")
	})
})
