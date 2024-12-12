// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership_test

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/pkg/controllers/team"
	"github.com/cloudoperators/greenhouse/pkg/controllers/teammembership"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	usersServer *httptest.Server
)

func TestTeammembership(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Teammembership Suite")
}

var _ = BeforeSuite(func() {
	By("mocking SCIM server")
	usersServer = scim.ReturnUserResponseMockServer()

	test.RegisterController("teammembershipUpdaterController",
		(&teammembership.TeamMembershipUpdaterController{}).SetupWithManager)
	test.RegisterController("teamController", (&team.TeamReconciler{}).SetupWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	usersServer.Close()

	test.TestAfterSuite()
})
