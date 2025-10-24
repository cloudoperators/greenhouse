// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/features"
)

// ResolveTemplatedValues processes PluginOptionValues with templates and resolves them to concrete values.
// If template rendering is disabled via feature flag, the template string itself is used as the value.
func ResolveTemplatedValues(ctx context.Context, optionValues []greenhousev1alpha1.PluginOptionValue, featureFlags features.Getter) ([]greenhousev1alpha1.PluginOptionValue, error) {
	resolvedValues := make([]greenhousev1alpha1.PluginOptionValue, 0, len(optionValues))

	var renderingEnabled bool
	if featureFlags != nil && featureFlags.IsTemplateRenderingEnabled(ctx) {
		renderingEnabled = true
	}

	templateData, err := buildTemplateData(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}

	for _, optionValue := range optionValues {
		if optionValue.Template != nil {
			var resolvedValue string

			if renderingEnabled {
				resolvedValue, err = resolveTemplate(*optionValue.Template, templateData)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve template for option %s: %w", optionValue.Name, err)
				}
			} else {
				resolvedValue = *optionValue.Template
			}

			jsonValue, err := json.Marshal(resolvedValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resolved template value for option %s: %w", optionValue.Name, err)
			}

			resolvedOptionValue := greenhousev1alpha1.PluginOptionValue{
				Name:      optionValue.Name,
				Value:     &apiextensionsv1.JSON{Raw: jsonValue},
				ValueFrom: nil,
				Template:  nil,
			}
			resolvedValues = append(resolvedValues, resolvedOptionValue)
		} else {
			// Keep non-template values as-is.
			resolvedValues = append(resolvedValues, optionValue)
		}
	}

	return resolvedValues, nil
}

// buildTemplateData creates template data by extracting only the global greenhouse values from existing option values.
func buildTemplateData(optionValues []greenhousev1alpha1.PluginOptionValue) (map[string]any, error) {
	greenhouseValues := make([]greenhousev1alpha1.PluginOptionValue, 0)
	for _, optionValue := range optionValues {
		// Include global.greenhouse.* values for templating.
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

func resolveTemplate(templateStr string, templateData map[string]interface{}) (string, error) {
	tmpl := template.New("plugin-option").Funcs(sprig.TxtFuncMap())

	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
