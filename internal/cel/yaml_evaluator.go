// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

// yamlExpressionPattern matches ${...} placeholders in YAML strings
var yamlExpressionPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// EvaluateYamlExpression takes a YAML string with ${...} placeholders and evaluates them as CEL expressions.
func EvaluateYamlExpression(yamlStr string, templateData map[string]any) ([]byte, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("yaml expression cannot be empty")
	}

	matches := yamlExpressionPattern.FindAllStringSubmatch(yamlStr, -1)
	if len(matches) == 0 {
		// No placeholders, convert directly to JSON
		return sigsyaml.YAMLToJSON([]byte(yamlStr))
	}

	resolvedYaml := yamlStr
	for _, match := range matches {
		placeholder := match[0]
		expression := match[1]

		celResult, err := EvaluatePluginExpression(expression, templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate CEL expression '%s': %w", expression, err)
		}

		value, err := valueToYamlString(celResult)
		if err != nil {
			return nil, fmt.Errorf("failed to convert CEL result to YAML string for expression '%s': %w", expression, err)
		}

		resolvedYaml = strings.ReplaceAll(resolvedYaml, placeholder, value)
	}

	return sigsyaml.YAMLToJSON([]byte(resolvedYaml))
}

// valueToYamlString converts a CEL result to a string suitable for YAML substitution.
// Simple types become their string representation, complex types get marshaled to YAML.
func valueToYamlString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "null", nil
	default:
		yamlBytes, err := yaml.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("failed to marshal complex value to YAML: %w", err)
		}
		// yaml.Marshal adds a trailing newline, strip it.
		yamlStr := string(yamlBytes)
		if len(yamlStr) > 0 && yamlStr[len(yamlStr)-1] == '\n' {
			yamlStr = yamlStr[:len(yamlStr)-1]
		}
		return yamlStr, nil
	}
}
