// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// AsJSONList decodes an option value as a JSON list. An option that sets a value next to a
// valueFrom.ref has the two merged into one list, so the value has to be a list as well.
func AsJSONList(value *apiextensionsv1.JSON) ([]any, error) {
	var list []any
	if err := json.Unmarshal(value.Raw, &list); err != nil {
		return nil, fmt.Errorf("value %s is not a list", string(value.Raw))
	}
	return list, nil
}

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
