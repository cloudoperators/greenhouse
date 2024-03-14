// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/pkg/admission"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":      admission.SetupClusterWebhookWithManager,
	"secrets":      admission.SetupSecretWebhookWithManager,
	"organization": admission.SetupOrganizationWebhookWithManager,
	"plugin":       admission.SetupPluginWebhookWithManager,
	"pluginconfig": admission.SetupPluginConfigWebhookWithManager,
	// "role":         admission.SetupRoleWebhookWithManager,
	// "rolebinding":  admission.SetupRoleBindingWebhookWithManager,
	"team": admission.SetupTeamWebhookWithManager,
}
