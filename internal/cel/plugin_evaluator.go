// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
)

// EvaluatePluginExpression evaluates a CEL expression with access to global Greenhouse values.
func EvaluatePluginExpression(expression string, templateData map[string]any) (any, error) {
	if expression == "" {
		return nil, fmt.Errorf("expression cannot be empty")
	}

	env, err := createPluginCELEnv()
	if err != nil {
		return nil, err
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	out, _, err := prg.Eval(templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	return convertCELValue(out)
}

// createPluginCELEnv creates a CEL environment for plugin expressions.
func createPluginCELEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("global", cel.DynType),

		// See: https://github.com/google/cel-go/tree/master/ext#strings
		ext.Strings(),
	)
}
