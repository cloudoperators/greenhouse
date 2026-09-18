// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
)

// initializeConditions sets the provided conditions to Unknown if they do not already exist.
func initializeConditions(resource common.GenericPluginDefinition, conditionTypes ...greenhousemetav1alpha1.ConditionType) {
	conditions := resource.GetConditions()
	for _, condType := range conditionTypes {
		if cond := conditions.GetConditionByType(condType); cond == nil {
			resource.SetCondition(
				greenhousemetav1alpha1.UnknownCondition(condType, greenhousev1alpha1.PluginDefinitionProgressingReason, "reconciliation in progress"),
			)
		}
	}
}

func setReadyCondition(resource common.GenericPluginDefinition) {
	if resource.GetPluginDefinitionSpec().HelmChart == nil {
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "No HelmChart defined"))
		return
	}

	conditions := resource.GetConditions()
	helmChartCondition := conditions.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
	switch {
	case helmChartCondition == nil:
		resource.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "HelmChart status unknown"))
		return
	case !helmChartCondition.IsTrue():
		resource.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousemetav1alpha1.ReadyCondition,
			helmChartCondition.Reason,
			helmChartCondition.Message))
		return
	}

	imageReplicationCondition := conditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
	switch {
	case imageReplicationCondition == nil:
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "PluginDefinition is ready"))
	case imageReplicationCondition.IsFalse():
		resource.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousemetav1alpha1.ReadyCondition,
			imageReplicationCondition.Reason,
			imageReplicationCondition.Message))
	default:
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "PluginDefinition is ready"))
	}
}
