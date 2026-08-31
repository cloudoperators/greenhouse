// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"strings"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

type CELResolver struct {
	templateData map[string]any
}

// NewCELResolver creates a new CELResolver for a given Plugin.
func NewCELResolver(optionValues []greenhousev1alpha1.PluginOptionValue) (*CELResolver, error) {
	templateData, err := BuildTemplateData(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}
	return &CELResolver{templateData: templateData}, nil
}

// BuildTemplateData extracts global.greenhouse.* values to build template data for CEL evaluation.
func BuildTemplateData(optionValues []greenhousev1alpha1.PluginOptionValue) (map[string]any, error) {
	greenhouseValues := make([]greenhousev1alpha1.PluginOptionValue, 0)
	for _, optionValue := range optionValues {
		// Include global.greenhouse.* values for CEL evaluation.
		if strings.HasPrefix(optionValue.Name, "global.greenhouse") {
			greenhouseValues = append(greenhouseValues, optionValue)
		}
	}

	templateData, err := ConvertFlatValuesToHelmValues(greenhouseValues)
	if err != nil {
		return nil, fmt.Errorf("failed to convert values to helm structure: %w", err)
	}

	return templateData, nil
}
