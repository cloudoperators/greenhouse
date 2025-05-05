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
	test.RegisterWebhook("roleWebhook", SetupTeamRoleWebhookWithManager)
	test.RegisterWebhook("rolebindingWebhook", setupRoleBindingWebhookForTest)
	test.RegisterWebhook("rolebindingV1alpha2Webhook", setupRoleBindingV1alpha2WebhookForTest)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

// setupRoleBindingWebhookForTest adds an indexField for '.spec.teamRoleRef', additionally to setting up the webhook for the RoleBinding resource. It is used in the webhook tests.
// we can't add this to the webhook setup because it's already indexed by the controller and indexing the field twice is not possible.
// This is to have the webhook tests run independently of the controller.
func setupRoleBindingV1alpha2WebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha2.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*greenhousev1alpha2.TeamRoleBinding)
		if roleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the rolebindings by roleRef")
	return v1alpha2.SetupTeamRoleBindingWebhookWithManager(mgr)
}
