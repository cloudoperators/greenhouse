// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	organizationpkg "github.com/cloudoperators/greenhouse/internal/controller/organization"
	defpkg "github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"
	"github.com/cloudoperators/greenhouse/internal/dex"
	"github.com/cloudoperators/greenhouse/internal/scim"
	"github.com/cloudoperators/greenhouse/internal/test"
	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

var (
	DexStorageType string
	groupsServer   *httptest.Server
)

func TestOrganization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrganizationControllerSuite")
}

var _ = BeforeSuite(func() {
	By("mocking SCIM server")
	groupsServer = scim.ReturnDefaultGroupResponseMockServer()

	By("setting the dex storage type")
	// here we could set the dex storage type to be used in the tests to expect the right behavior
	// via environment variables in the future
	// for postgres we can start a postgres testcontainer
	DexStorageType = dex.K8s

	test.RegisterController("organizationController", (&organizationpkg.OrganizationReconciler{Namespace: "default", DexStorageType: DexStorageType}).SetupWithManager)
	test.RegisterController("pluginDefinitionController", (&defpkg.PluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterController("clusterPluginDefinition", (&defpkg.ClusterPluginDefinitionReconciler{}).SetupWithManager)
	test.RegisterWebhook("orgWebhook", webhookv1alpha1.SetupOrganizationWebhookWithManager)
	test.RegisterWebhook("teamWebhook", webhookv1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", webhookv1alpha1.SetupPluginWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", webhookv1alpha1.SetupTeamRoleWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	groupsServer.Close()
	test.TestAfterSuite()
})
