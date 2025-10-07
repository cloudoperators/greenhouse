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

const (
	objectPrefix  = "object"
	objectsPrefix = "objects"
)

type Evaluator struct {
	env *cel.Env
}

func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable(objectPrefix, cel.DynType),
		cel.Variable(objectsPrefix, cel.ListType(cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{env: env}, nil
}

// Evaluate evaluates a CEL expression against one or more Kubernetes objects.
// If a single object is provided, it is available as "object" in the expression.
// If multiple objects are provided, they are available as "objects" in the expression.
func (e *Evaluator) Evaluate(expression string, objs ...client.Object) (any, error) {
	if len(objs) == 0 {
		return nil, errors.New("at least one object must be provided")
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

	if len(objs) == 1 {
		return e.evaluateSingle(objs[0], prg)
	}

	return e.evaluateMultiple(objs, prg)
}

func (e *Evaluator) evaluateSingle(obj client.Object, prg cel.Program) (any, error) {
	if obj == nil {
		return nil, errors.New("object cannot be nil")
	}

	objMap, err := structToMap(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to map: %w", err)
	}

	out, _, err := prg.Eval(map[string]any{
		objectPrefix: objMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return convertCELValue(out)
}

func (e *Evaluator) evaluateMultiple(objs []client.Object, prg cel.Program) (any, error) {
	objMaps := make([]any, len(objs))
	for i, obj := range objs {
		if obj == nil {
			return nil, errors.New("object cannot be nil")
		}

		objMap, err := structToMap(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert object at index %d to map: %w", i, err)
		}
		objMaps[i] = objMap
	}

	out, _, err := prg.Eval(map[string]any{
		objectsPrefix: objMaps,
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
