// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("pluginDefinitionWebhook", SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("clusterPluginDefinitionWebhook", SetupClusterPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", SetupPluginWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", SetupPluginPresetWebhookWithManager)
	test.RegisterWebhook("clusterValidation", SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", SetupSecretWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", SetupTeamWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("teamRoleBindingV1alpha1Webhook", SetupTeamRoleBindingWebhookWithManager)
	test.RegisterWebhook("teamRolebindingV1alpha2Webhook", v1alpha2.SetupTeamRoleBindingWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})
