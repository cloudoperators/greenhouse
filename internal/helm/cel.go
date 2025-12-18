// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"encoding/json"
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/cel"
)

type CELResolver struct {
	templateData map[string]any
}

// NewCELResolver creates a new CELResolver for a given Plugin.
func NewCELResolver(optionValues []greenhousev1alpha1.PluginOptionValue) (*CELResolver, error) {
	templateData, err := buildTemplateData(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}
	return &CELResolver{templateData: templateData}, nil
}

func (c *CELResolver) ResolveExpression(optionValue greenhousev1alpha1.PluginOptionValue, expressionEvaluationEnabled bool) (*greenhousev1alpha1.PluginOptionValue, error) {
	// early return if there is no expression to resolve
	if optionValue.Expression == nil {
		return &optionValue, nil
	}
	// copy the expression into the value field if expression evaluation is disabled
	if !expressionEvaluationEnabled {
		jsonValue, err := json.Marshal(*optionValue.Expression)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal literal expression for option %s: %w", optionValue.Name, err)
		}
		return &greenhousev1alpha1.PluginOptionValue{
			Name:       optionValue.Name,
			Value:      &apiextensionsv1.JSON{Raw: jsonValue},
			ValueFrom:  nil,
			Expression: nil,
		}, nil
	}
	// evaluate the expression using CEL
	jsonValue, err := cel.EvaluateExpression(*optionValue.Expression, c.templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression for option %s: %w", optionValue.Name, err)
	}

	return &greenhousev1alpha1.PluginOptionValue{
		Name:       optionValue.Name,
		Value:      &apiextensionsv1.JSON{Raw: jsonValue},
		ValueFrom:  nil,
		Expression: nil,
	}, nil
}

// buildTemplateData extracts global.greenhouse.* values to build template data for CEL evaluation.
func buildTemplateData(optionValues []greenhousev1alpha1.PluginOptionValue) (map[string]any, error) {
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
