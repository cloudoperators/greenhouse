// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func ConvertToPluginOptionValues(presetValues []greenhousev1alpha1.PluginPresetPluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(presetValues))
	for _, pv := range presetValues {
		ov := greenhousev1alpha1.PluginOptionValue{
			Name:  pv.Name,
			Value: pv.Value,
		}

		if pv.ValueFrom != nil {
			ov.ValueFrom = &greenhousev1alpha1.PluginValueFromSource{
				Secret: pv.ValueFrom.Secret,
			}
		}
		result = append(result, ov)
	}
	return result
}

func ConvertToPresetOptionValues(values []greenhousev1alpha1.PluginOptionValue) []greenhousev1alpha1.PluginPresetPluginOptionValue {
	result := make([]greenhousev1alpha1.PluginPresetPluginOptionValue, 0, len(values))
	for _, v := range values {
		pv := greenhousev1alpha1.PluginPresetPluginOptionValue{
			Name:  v.Name,
			Value: v.Value,
		}
		if v.ValueFrom != nil {
			pv.ValueFrom = &greenhousev1alpha1.PluginPresetPluginValueFromSource{
				Secret: v.ValueFrom.Secret,
			}
		}
		result = append(result, pv)
	}
	return result
}
