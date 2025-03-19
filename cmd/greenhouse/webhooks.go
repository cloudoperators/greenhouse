// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	ctrl "sigs.k8s.io/controller-runtime"

	admissionv1alpha1 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha1"
	admissionv1alpha2 "github.com/cloudoperators/greenhouse/internal/webhook/v1alpha2"
)

var knownWebhooks = map[string]func(mgr ctrl.Manager) error{
	"cluster":                  admissionv1alpha1.SetupClusterWebhookWithManager,
	"secrets":                  admissionv1alpha1.SetupSecretWebhookWithManager,
	"organization":             admissionv1alpha1.SetupOrganizationWebhookWithManager,
	"pluginDefinition":         admissionv1alpha1.SetupPluginDefinitionWebhookWithManager,
	"plugin":                   admissionv1alpha1.SetupPluginWebhookWithManager,
	"pluginPreset":             admissionv1alpha1.SetupPluginPresetWebhookWithManager,
	"teamrole":                 admissionv1alpha1.SetupTeamRoleWebhookWithManager,
	"teamrolebinding_v1alpha1": admissionv1alpha1.SetupTeamRoleBindingWebhookWithManager,
	"teamrolebinding_v1alpha2": admissionv1alpha2.SetupTeamRoleBindingWebhookWithManager,
	"team":                     admissionv1alpha1.SetupTeamWebhookWithManager,
}
