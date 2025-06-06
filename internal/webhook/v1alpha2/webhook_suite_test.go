// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("clusterValidation", v1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", v1alpha1.SetupSecretWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", v1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", v1alpha1.SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("teamRolebindingV1alpha2Webhook", SetupTeamRoleBindingWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})
