// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	organizationpkg "github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	"github.com/cloudoperators/greenhouse/pkg/dex"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	"github.com/cloudoperators/greenhouse/pkg/test"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	mockDB  = "mock"
	mockUsr = "mock"
	mockPwd = "mock_pwd"
)

var (
	groupsServer *httptest.Server
	mockPgTc     *postgres.PostgresContainer
)

func TestOrganization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrganizationControllerSuite")
}

var _ = BeforeSuite(func() {
	var err error
	ctx := context.Background()
	By("mocking SCIM server")
	groupsServer = scim.ReturnDefaultGroupResponseMockServer()

	mockPgTc, err = startPgTC(ctx)
	Expect(err).NotTo(HaveOccurred())
	host, err := mockPgTc.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Setenv("PG_HOST", host)).ToNot(HaveOccurred())
	port, err := mockPgTc.MappedPort(ctx, "5432/tcp")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Setenv("PG_PORT", port.Port())).ToNot(HaveOccurred())
	Expect(os.Setenv("PG_USER", mockUsr)).ToNot(HaveOccurred())
	Expect(os.Setenv("PG_PASSWORD", mockPwd)).ToNot(HaveOccurred())
	Expect(os.Setenv("PG_DATABASE", mockDB)).ToNot(HaveOccurred())

	test.RegisterController("organizationController", (&organizationpkg.OrganizationReconciler{Namespace: "default", DexStorageType: dex.Postgres}).SetupWithManager)
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
	err := testcontainers.TerminateContainer(mockPgTc)
	Expect(err).NotTo(HaveOccurred())
	test.TestAfterSuite()
})

func startPgTC(ctx context.Context) (*postgres.PostgresContainer, error) {
	return postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase(mockDB),
		postgres.WithUsername(mockUsr),
		postgres.WithPassword(mockPwd),
		testcontainers.WithWaitStrategy(
			// First, we wait for the container to log readiness twice.
			// This is because it will restart itself after the first startup.
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			// Then, we wait for docker to actually serve the port on localhost.
			// For non-linux OSes like Mac and Windows, Docker or Rancher Desktop will have to start a separate proxy.
			// Without this, the tests will be flaky on those OSes!
			wait.ForListeningPort("5432/tcp"),
		))
}
