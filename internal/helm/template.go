// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// ResolveTemplatedValues processes PluginOptionValues with templates and resolves them to concrete values.
func ResolveTemplatedValues(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, optionValues []greenhousev1alpha1.PluginOptionValue) ([]greenhousev1alpha1.PluginOptionValue, error) {
	resolvedValues := make([]greenhousev1alpha1.PluginOptionValue, 0, len(optionValues))

	for _, optionValue := range optionValues {
		if optionValue.Template != nil {
			templateData, err := buildTemplateData(ctx, c, plugin)
			if err != nil {
				return nil, fmt.Errorf("failed to build template data: %w", err)
			}

			resolvedValue, err := resolveTemplate(*optionValue.Template, templateData)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve template for option %s: %w", optionValue.Name, err)
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

// buildTemplateData creates template data using the same data as existing greenhouse values.
func buildTemplateData(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (map[string]interface{}, error) {
	greenhouseValues, err := GetGreenhouseValues(ctx, c, *plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to get greenhouse values: %w", err)
	}

	// Convert flat values to nested structure using existing Helm conversion
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
