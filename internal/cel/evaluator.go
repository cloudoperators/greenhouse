// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

type Evaluator struct {
	env *cel.Env
}

func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("plugin", cel.DynType),
		cel.Variable("plugins", cel.ListType(cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{env: env}, nil
}

// EvaluatePluginExpression evaluates a CEL expression against a Plugin.
// The plugin is available as the "plugin" variable in the expression.
func (e *Evaluator) EvaluatePluginExpression(plugin *greenhousev1alpha1.Plugin, expression string) (any, error) {
	if plugin == nil {
		return nil, errors.New("plugin cannot be nil")
	}
	if expression == "" {
		return nil, errors.New("expression cannot be empty")
	}

	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	pluginMap, err := structToMap(plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plugin to map: %w", err)
	}

	out, _, err := prg.Eval(map[string]any{
		"plugin": pluginMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return convertCELValue(out)
}

// EvaluatePluginListExpression evaluates a CEL expression against a list of Plugins.
// The plugins are available as the "plugins" variable in the expression.
func (e *Evaluator) EvaluatePluginListExpression(plugins []greenhousev1alpha1.Plugin, expression string) (any, error) {
	if len(plugins) == 0 {
		return nil, errors.New("plugins list cannot be empty")
	}
	if expression == "" {
		return nil, errors.New("expression cannot be empty")
	}

	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	pluginMaps := make([]any, len(plugins))
	for i, plugin := range plugins {
		pluginMap, err := structToMap(&plugin)
		if err != nil {
			return nil, fmt.Errorf("failed to convert plugin at index %d to map: %w", i, err)
		}
		pluginMaps[i] = pluginMap
	}

	out, _, err := prg.Eval(map[string]any{
		"plugins": pluginMaps,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return convertCELValue(out)
}

func convertCELValue(val ref.Val) (any, error) {
	if types.IsError(val) {
		return nil, fmt.Errorf("CEL evaluation error: %v", val)
	}

	if listVal, ok := val.Value().([]ref.Val); ok {
		result := make([]any, len(listVal))
		for i, item := range listVal {
			result[i] = item.Value()
		}
		return result, nil
	}

	return val.Value(), nil
}

func structToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}
