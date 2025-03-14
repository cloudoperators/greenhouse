// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	ctrl "sigs.k8s.io/controller-runtime"

	admission "github.com/cloudoperators/greenhouse/internal/webhook"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":          admission.SetupClusterWebhookWithManager,
	"secrets":          admission.SetupSecretWebhookWithManager,
	"organization":     admission.SetupOrganizationWebhookWithManager,
	"pluginDefinition": admission.SetupPluginDefinitionWebhookWithManager,
	"plugin":           admission.SetupPluginWebhookWithManager,
	"pluginPreset":     admission.SetupPluginPresetWebhookWithManager,
	"teamrole":         admission.SetupTeamRoleWebhookWithManager,
	"teamrolebinding":  admission.SetupTeamRoleBindingWebhookWithManager,
	"team":             admission.SetupTeamWebhookWithManager,
}
