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
