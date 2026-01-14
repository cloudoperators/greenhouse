// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	ctrl "sigs.k8s.io/controller-runtime"

	webhookv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
	webhookv1alpha2 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":                  webhookv1alpha1.SetupClusterWebhookWithManager,
	"secrets":                  webhookv1alpha1.SetupSecretWebhookWithManager,
	"organization":             webhookv1alpha1.SetupOrganizationWebhookWithManager,
	"pluginDefinition":         webhookv1alpha1.SetupPluginDefinitionWebhookWithManager,
	"plugin":                   webhookv1alpha1.SetupPluginWebhookWithManager,
	"pluginPreset":             webhookv1alpha1.SetupPluginPresetWebhookWithManager,
	"teamrole":                 webhookv1alpha1.SetupTeamRoleWebhookWithManager,
	"teamrolebinding_v1alpha2": webhookv1alpha2.SetupTeamRoleBindingWebhookWithManager,
	"team":                     webhookv1alpha1.SetupTeamWebhookWithManager,
	"clusterPluginDefinition":  webhookv1alpha1.SetupClusterPluginDefinitionWebhookWithManager,
}
