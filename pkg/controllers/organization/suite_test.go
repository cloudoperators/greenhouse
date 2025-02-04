// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	organizationpkg "github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	dexstore "github.com/cloudoperators/greenhouse/pkg/dex/store"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
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
	dexter := dexMocks()
	test.RegisterController("organizationController", (&organizationpkg.OrganizationReconciler{Namespace: "default", Dexter: dexter}).SetupWithManager)
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

func dexMocks() dexstore.Dexter {
	dexter := &mocks.MockDexter{}
	dexter.On("CreateUpdateConnector", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dexter.On("CreateUpdateOauth2Client", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dexter.On("GetBackend").Return(dexstore.K8s)
	return dexter
}
