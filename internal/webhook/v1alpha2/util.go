// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/webhook"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// validateClusterSelector checks if the resource has a valid clusterSelector.
func validateClusterSelector(cs greenhousev1alpha2.ClusterSelector, resourceGroupKind schema.GroupKind) error {
	if cs.Name != "" && (len(cs.LabelSelector.MatchLabels) > 0 || len(cs.LabelSelector.MatchExpressions) > 0) {
		return apierrors.NewInvalid(resourceGroupKind, cs.Name, field.ErrorList{field.Invalid(
			field.NewPath("spec", "clusterSelector", "name"),
			cs.Name,
			"cannot specify both spec.clusterSelector.Name and spec.clusterSelector.labelSelector",
		)})
	}

	if cs.Name == "" && (len(cs.LabelSelector.MatchLabels) == 0 && len(cs.LabelSelector.MatchExpressions) == 0) {
		return apierrors.NewInvalid(resourceGroupKind, cs.Name, field.ErrorList{field.Invalid(
			field.NewPath("spec", "clusterSelector", "name"),
			cs.Name,
			"must specify either spec.clusterSelector.name or spec.clusterSelector.labelSelector",
		)})
	}
	return nil
}

// validatePluginOptionValuesForPreset validates plugin options and their values, but skips the check for required options.
// Required options are checked at the Plugin creation level, because the preset can override options and we cannot predict what clusters will be a part of the PluginPreset later on.
func ValidatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha2.PluginPreset, pluginDefinition *greenhousev1alpha1.PluginDefinition) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := webhook.ValidatePluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinition, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = webhook.ValidatePluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinition, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}
