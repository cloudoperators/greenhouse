// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha3

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha3 "github.com/cloudoperators/greenhouse/api/v1alpha3"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// validatePluginOptionValuesForPreset validates plugin options and their values, but skips the check for required options.
// Required options are checked at the Plugin creation level, because the preset can override options and we cannot predict what clusters will be a part of the PluginPreset later on.
func ValidatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha3.PluginPreset, pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := webhook.ValidatePluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = webhook.ValidatePluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinition.Name, pluginDefinition.Spec, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}
