// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/cel"
)

// ResolveCelExpressions processes PluginOptionValues with CEL expressions and resolves them to concrete values.
// For performance, it caches compiled CEL programs when the same expression is used multiple times.
func ResolveCelExpressions(ctx context.Context, optionValues []greenhousev1alpha1.PluginOptionValue) ([]greenhousev1alpha1.PluginOptionValue, error) {
	resolvedValues := make([]greenhousev1alpha1.PluginOptionValue, 0, len(optionValues))

	templateData, err := buildTemplateData(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}

	programCache := make(map[string]any)
	for _, optionValue := range optionValues {
		if optionValue.CelExpression != nil {
			expression := *optionValue.CelExpression
			if _, exists := programCache[expression]; !exists {
				resolvedValue, err := cel.EvaluatePluginExpression(expression, templateData)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate CEL expression for option %s: %w", optionValue.Name, err)
				}
				programCache[expression] = resolvedValue
			}
		}
	}

	for _, optionValue := range optionValues {
		if optionValue.CelExpression != nil {
			resolvedValue := programCache[*optionValue.CelExpression]

			jsonValue, err := json.Marshal(resolvedValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resolved CEL value for option %s: %w", optionValue.Name, err)
			}

			resolvedOptionValue := greenhousev1alpha1.PluginOptionValue{
				Name:          optionValue.Name,
				Value:         &apiextensionsv1.JSON{Raw: jsonValue},
				ValueFrom:     nil,
				Template:      nil,
				CelExpression: nil,
			}
			resolvedValues = append(resolvedValues, resolvedOptionValue)
		} else {
			resolvedValues = append(resolvedValues, optionValue)
		}
	}

	return resolvedValues, nil
}
