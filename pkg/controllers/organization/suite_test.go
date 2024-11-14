// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	organizationpkg "github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	groupsServer *httptest.Server
)

func TestOrganization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrganizationControllerSuite")
}

var _ = BeforeSuite(func() {
	By("mocking SCIM server")
	groupsServer = scim.ReturnDefaultGroupResponseMockServer()

	test.RegisterController("organizationController", (&organizationpkg.OrganizationReconciler{}).SetupWithManager)
	// test.RegisterController("organizationRBAC", (&organizationpkg.RBACReconciler{}).SetupWithManager)
	// test.RegisterController("organizationServiceProxy", (&organizationpkg.ServiceProxyReconciler{}).SetupWithManager)
	test.RegisterController("teamRoleSeeder", (&organizationpkg.TeamRoleSeederReconciler{}).SetupWithManager)
	test.RegisterWebhook("orgWebhook", admission.SetupOrganizationWebhookWithManager)
	test.RegisterWebhook("teamWebhook", admission.SetupTeamWebhookWithManager)
	test.RegisterWebhook("pluginDefinitionWebhook", admission.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", admission.SetupPluginWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", admission.SetupTeamRoleWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	groupsServer.Close()

	test.TestAfterSuite()
})
