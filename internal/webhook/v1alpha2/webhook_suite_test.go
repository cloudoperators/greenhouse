// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("pluginDefinitionWebhook", v1alpha1.SetupPluginDefinitionWebhookWithManager)
	test.RegisterWebhook("pluginWebhook", v1alpha1.SetupPluginWebhookWithManager)
	test.RegisterWebhook("pluginPresetWebhook", v1alpha1.SetupPluginPresetWebhookWithManager)
	test.RegisterWebhook("clusterValidation", v1alpha1.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", v1alpha1.SetupSecretWebhookWithManager)
	test.RegisterWebhook("teamsWebhook", v1alpha1.SetupTeamWebhookWithManager)
	test.RegisterWebhook("teamRoleWebhook", v1alpha1.SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("teamRolebindingV1alpha2Webhook", setupTeamRoleBindingV1alpha2WebhookForTest)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

// setupTeamRoleBindingV1alpha2WebhookForTest adds an indexField for '.spec.teamRoleRef', additionally to setting up the webhook for the v1alpha2 TeamRoleBinding resource. It is used in the integration tests.
// We can't add this to the webhook setup because it's already indexed in the main.go and indexing the field twice is not possible.
func setupTeamRoleBindingV1alpha2WebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha2.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*greenhousev1alpha2.TeamRoleBinding)
		if roleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the TeamRoleBindings by teamRoleRef")
	return SetupTeamRoleBindingWebhookWithManager(mgr)
}
