// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
)

var (
	pluginCELEnv        *cel.Env
	errpluginCELEnvInit error
	pluginCELEnvOnce    sync.Once
)

// getPluginCELEnv returns a cached CEL environment for plugin expressions.
func getPluginCELEnv() (*cel.Env, error) {
	pluginCELEnvOnce.Do(func() {
		pluginCELEnv, errpluginCELEnvInit = createPluginCELEnv()
	})
	return pluginCELEnv, errpluginCELEnvInit
}

// EvaluatePluginExpression evaluates a CEL expression with access to global Greenhouse values.
func EvaluatePluginExpression(expression string, templateData map[string]any) (any, error) {
	if expression == "" {
		return nil, errors.New("expression cannot be empty")
	}

	env, err := getPluginCELEnv()
	if err != nil {
		return nil, err
	}

	prg, err := CompileExpressionWithEnv(expression, env)
	if err != nil {
		return nil, err
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
