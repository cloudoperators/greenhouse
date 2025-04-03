// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("pluginDefinitionWebhook", SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", SetupPluginWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", SetupPluginPresetWebhookWithManager)
	test.RegisterWebhook("clusterValidation", SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", SetupSecretWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", SetupTeamWebhookWithManager)
	test.RegisterWebhook("roleWebhook", SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("rolebindingWebhook", setupRoleBindingWebhookForTest)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})
