// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	admission_v1alpha2 "github.com/cloudoperators/greenhouse/pkg/admission/v1alpha2"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":                  admission.SetupClusterWebhookWithManager,
	"secrets":                  admission.SetupSecretWebhookWithManager,
	"organization":             admission.SetupOrganizationWebhookWithManager,
	"pluginDefinition":         admission.SetupPluginDefinitionWebhookWithManager,
	"plugin":                   admission.SetupPluginWebhookWithManager,
	"pluginPreset":             admission.SetupPluginPresetWebhookWithManager,
	"teamrole":                 admission.SetupTeamRoleWebhookWithManager,
	"teamrolebinding_v1alpha1": admission.SetupTeamRoleBindingWebhookWithManager,
	"teamrolebinding_v1alpha2": admission_v1alpha2.SetupTeamRoleBindingWebhookWithManager,
	"team":                     admission.SetupTeamWebhookWithManager,
}
