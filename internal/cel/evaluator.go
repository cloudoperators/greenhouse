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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EvaluateTyped evaluates a CEL expression against a single Kubernetes object.
// The result is unmarshalled into the specified generic type T.
// Returns the result of the evaluation or an error.
func EvaluateTyped[T any](expression string, obj client.Object) (T, error) {
	var zero T

	val, err := Evaluate(expression, obj)
	if err != nil {
		return zero, err
	}

	// Fast-path for direct type match (primitive types)
	if cast, ok := val.(T); ok {
		return cast, nil
	}

	// Fallback: normalize through JSON once for structured types
	data, err := json.Marshal(val)
	if err != nil {
		return zero, fmt.Errorf("marshal CEL result: %w", err)
	}

	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return zero, fmt.Errorf("unmarshal CEL result: %w", err)
	}
	return out, nil
}

// Evaluate evaluates a CEL expression against a single Kubernetes object.
// Returns the result of the evaluation.
func Evaluate(expression string, obj client.Object) (any, error) {
	if obj == nil {
		return nil, errors.New("object cannot be nil")
	}

	if expression == "" {
		return nil, errors.New("expression cannot be empty")
	}

	prg, err := compileExpression(expression)
	if err != nil {
		return nil, err
	}

	return evaluateObject(obj, prg)
}

// EvaluateList evaluates a CEL expression against multiple Kubernetes objects.
// Returns a slice of results, one for each input object.
func EvaluateList(expression string, objs []client.Object) ([]any, error) {
	if len(objs) == 0 {
		return nil, errors.New("at least one object must be provided")
	}

	if expression == "" {
		return nil, errors.New("expression cannot be empty")
	}

	prg, err := compileExpression(expression)
	if err != nil {
		return nil, err
	}

	results := make([]any, len(objs))
	for i, obj := range objs {
		result, err := evaluateObject(obj, prg)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate object at index %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

func compileExpression(expression string) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Variable("object", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	return prg, nil
}

func evaluateObject(obj client.Object, prg cel.Program) (any, error) {
	if obj == nil {
		return nil, errors.New("object cannot be nil")
	}

	objMap, err := structToMap(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to map: %w", err)
	}

	out, _, err := prg.Eval(map[string]any{
		"object": objMap,
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
