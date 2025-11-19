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

// ResolveCelExpressions processes PluginOptionValues with CEL or YAML expressions and resolves them to concrete values.
func ResolveCelExpressions(ctx context.Context, optionValues []greenhousev1alpha1.PluginOptionValue) ([]greenhousev1alpha1.PluginOptionValue, error) {
	resolvedValues := make([]greenhousev1alpha1.PluginOptionValue, 0, len(optionValues))

	templateData, err := buildTemplateData(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}

	for _, optionValue := range optionValues {
		if optionValue.CelExpression != nil {
			resolvedValue, err := cel.EvaluatePluginExpression(*optionValue.CelExpression, templateData)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate CEL expression for option %s: %w", optionValue.Name, err)
			}

			jsonValue, err := json.Marshal(resolvedValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resolved CEL value for option %s: %w", optionValue.Name, err)
			}

			resolvedValues = append(resolvedValues, greenhousev1alpha1.PluginOptionValue{
				Name:           optionValue.Name,
				Value:          &apiextensionsv1.JSON{Raw: jsonValue},
				ValueFrom:      nil,
				Template:       nil,
				CelExpression:  nil,
				YamlExpression: nil,
			})
		} else if optionValue.YamlExpression != nil {
			jsonValue, err := cel.EvaluateYamlExpression(*optionValue.YamlExpression, templateData)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate YAML expression for option %s: %w", optionValue.Name, err)
			}

			resolvedValues = append(resolvedValues, greenhousev1alpha1.PluginOptionValue{
				Name:           optionValue.Name,
				Value:          &apiextensionsv1.JSON{Raw: jsonValue},
				ValueFrom:      nil,
				Template:       nil,
				CelExpression:  nil,
				YamlExpression: nil,
			})
		} else {
			resolvedValues = append(resolvedValues, optionValue)
		}
	}

	return resolvedValues, nil
}
