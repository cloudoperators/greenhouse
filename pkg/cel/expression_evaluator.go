// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	sigsyaml "sigs.k8s.io/yaml"
)

var (
	// expressionPattern matches ${...} placeholders in expression strings.
	expressionPattern = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// createExpressionCELEnv creates a CEL environment for expressions.
func createExpressionCELEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("global", cel.DynType),
		// See: https://github.com/google/cel-go/tree/master/ext#strings
		ext.Strings(),
	)
}

// EvaluateExpression takes a YAML string with ${...} placeholders and evaluates them as CEL expressions.
func EvaluateExpression(yamlStr string, templateData map[string]any) ([]byte, error) {
	if yamlStr == "" {
		return nil, errors.New("expression cannot be empty")
	}

	matches := expressionPattern.FindAllStringSubmatch(yamlStr, -1)
	if len(matches) == 0 {
		// No placeholders, convert directly to JSON.
		return sigsyaml.YAMLToJSON([]byte(yamlStr))
	}

	env, err := createExpressionCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	resolvedYaml := yamlStr
	for _, match := range matches {
		placeholder := match[0]
		expression := match[1]

		celResult, err := EvaluateWithData(expression, env, templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate CEL expression '%s': %w", expression, err)
		}

		value, err := formatValue(celResult)
		if err != nil {
			return nil, fmt.Errorf("failed to format value for expression '%s': %w", expression, err)
		}

		resolvedYaml = strings.ReplaceAll(resolvedYaml, placeholder, value)
	}

	return sigsyaml.YAMLToJSON([]byte(resolvedYaml))
}

// formatValue converts a CEL result to a string suitable for YAML substitution.
// Strings are returned as-is, all other types are JSON-marshaled.
func formatValue(value any) (string, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
