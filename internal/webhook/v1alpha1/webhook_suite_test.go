// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
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
	test.RegisterWebhook("teamRoleWebhook", SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("teamRoleBindingV1alpha1Webhook", setupTeamRoleBindingV1alpha1WebhookForTest)
	test.RegisterWebhook("teamRolebindingV1alpha2Webhook", setupTeamRoleBindingV1alpha2WebhookForTest)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

// setupTeamRoleBindingV1alpha1WebhookForTest adds an indexField for '.spec.teamRoleRef', additionally to setting up the webhook for the v1alpha2 TeamRoleBinding resource. It is used in the integration tests.
// We can't add this to the webhook setup because it's already indexed in the main.go and indexing the field twice is not possible.
func setupTeamRoleBindingV1alpha1WebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if roleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the v1alpha1 TeamRoleBindings by teamRoleRef")
	return SetupTeamRoleBindingWebhookWithManager(mgr)
}

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
	return v1alpha2.SetupTeamRoleBindingWebhookWithManager(mgr)
}
