// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	celgo "github.com/google/cel-go/cel"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/pkg/cel"
)

// resolvePluginOptionValuesForPreset resolves all expressions and references
// in a PluginPreset's option values before writing to Plugin.
func (r *PluginPresetReconciler) resolvePluginOptionValuesForPreset(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	// Phase 1: Resolve ALL expressions first
	resolvedExpressions, err := r.resolveExpressionsForPreset(ctx, preset, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expressions: %w", err)
	}

	// Phase 2: Resolve ALL references (expressions are now resolved)
	finalValues, err := r.resolveReferencesForPreset(ctx, cluster, preset.Namespace, resolvedExpressions)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	return finalValues, nil
}

// resolveExpressionsForPreset evaluates all expression fields in PluginPreset option values
func (r *PluginPresetReconciler) resolveExpressionsForPreset(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	// Check if any expressions exist - if not, return early
	hasExpressions := false
	for _, ov := range preset.Spec.Plugin.OptionValues {
		if ov.Expression != nil {
			hasExpressions = true
			break
		}
	}
	if !hasExpressions {
		return preset.Spec.Plugin.OptionValues, nil
	}

	// Build greenhouse values for CEL template data
	templateData, err := r.buildTemplateData(ctx, preset, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}

	// Evaluate each option value
	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(preset.Spec.Plugin.OptionValues))
	for _, optionValue := range preset.Spec.Plugin.OptionValues {
		if optionValue.Expression != nil {
			// Evaluate expression
			evaluatedValue, err := cel.EvaluateExpression(*optionValue.Expression, templateData)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression for option %s: %w", optionValue.Name, err)
			}

			// Replace expression with resolved value
			result = append(result, greenhousev1alpha1.PluginOptionValue{
				Name:  optionValue.Name,
				Value: &apiextensionsv1.JSON{Raw: evaluatedValue},
			})
		} else {
			// Keep as-is (direct value, valueFrom, etc.)
			result = append(result, optionValue)
		}
	}

	return result, nil
}

// resolveReferencesForPreset resolves all valueFrom.ref fields in option values.
// At this point, expressions in the current preset are already resolved.
// When referencing another PluginPreset, that preset's expressions are also resolved first.
func (r *PluginPresetReconciler) resolveReferencesForPreset(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
	optionValues []greenhousev1alpha1.PluginOptionValue,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	hasRefs := false
	for _, ov := range optionValues {
		if ov.ValueFrom != nil && ov.ValueFrom.Ref != nil {
			hasRefs = true
			break
		}
	}
	if !hasRefs {
		return optionValues, nil
	}

	log := ctrl.LoggerFrom(ctx)
	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(optionValues))

	for _, optionValue := range optionValues {
		if optionValue.ValueFrom != nil && optionValue.ValueFrom.Ref != nil {
			log.Info("Resolving valueFrom.ref",
				"option", optionValue.Name,
				"refKind", optionValue.ValueFrom.Ref.Kind,
				"refName", optionValue.ValueFrom.Ref.Name)

			resolvedValue, err := r.resolveRef(ctx, optionValue.ValueFrom.Ref, cluster, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve reference for %s: %w", optionValue.Name, err)
			}

			byteVal, err := json.Marshal(resolvedValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resolved value for %s: %w", optionValue.Name, err)
			}

			result = append(result, greenhousev1alpha1.PluginOptionValue{
				Name:  optionValue.Name,
				Value: &apiextensionsv1.JSON{Raw: byteVal},
			})
		} else {
			result = append(result, optionValue)
		}
	}

	return result, nil
}

// resolveRef resolves a reference to another resource (PluginPreset or Plugin)
func (r *PluginPresetReconciler) resolveRef(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
) (any, error) {

	refKind := ref.Kind
	if refKind == "" {
		refKind = greenhousev1alpha1.PluginPresetKind
	}

	switch refKind {
	case greenhousev1alpha1.PluginPresetKind:
		return r.resolvePluginPresetRef(ctx, ref, cluster, namespace)
	case greenhousev1alpha1.PluginKind:
		return r.resolvePluginRef(ctx, ref, namespace)
	default:
		return nil, fmt.Errorf("unsupported reference kind: %s", refKind)
	}
}

// resolvePluginPresetRef resolves a reference to PluginPreset(s).
// Supports both name-based and selector-based resolution.
func (r *PluginPresetReconciler) resolvePluginPresetRef(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
) (any, error) {

	switch {
	case ref.Name != "":
		return r.resolvePluginPresetRefByName(ctx, ref, cluster, namespace)
	case ref.Selector != nil:
		return r.resolvePluginPresetRefBySelector(ctx, ref, cluster, namespace)
	default:
		return nil, errors.New("either name or selector must be set in valueFrom.ref for PluginPreset")
	}
}

// resolvePluginPresetRefByName resolves a reference to a single PluginPreset by name.
func (r *PluginPresetReconciler) resolvePluginPresetRefByName(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
) (any, error) {

	log := ctrl.LoggerFrom(ctx)

	refPreset := &greenhousev1alpha1.PluginPreset{}
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, refPreset); err != nil {
		return nil, fmt.Errorf("failed to get PluginPreset %s: %w", ref.Name, err)
	}
	log.Info("Resolving reference to PluginPreset by name",
		"name", ref.Name,
		"expression", ref.Expression)

	resolvedRefValues, err := r.resolveExpressionsForPreset(ctx, refPreset, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expressions in referenced PluginPreset %s: %w", ref.Name, err)
	}
	celObject := buildCELObject(refPreset.Name, refPreset.Namespace, resolvedRefValues)
	value, err := evaluateCELWithObject(ref.Expression, celObject)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate reference expression: %w", err)
	}
	return value, nil
}

// resolvePluginPresetRefBySelector resolves references to multiple PluginPresets by label selector.
// The CEL expression is evaluated against each matching PluginPreset and results are collected.
func (r *PluginPresetReconciler) resolvePluginPresetRefBySelector(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
) (any, error) {

	log := ctrl.LoggerFrom(ctx)

	selector, err := metav1.LabelSelectorAsSelector(ref.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}
	presetList := &greenhousev1alpha1.PluginPresetList{}
	if err := r.List(ctx, presetList,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		return nil, fmt.Errorf("failed to list PluginPresets by selector: %w", err)
	}
	if len(presetList.Items) == 0 {
		log.Info("No PluginPresets found matching selector", "selector", ref.Selector)
		return []any{}, nil
	}
	log.Info("Resolving reference to PluginPresets by selector",
		"selector", ref.Selector,
		"matchCount", len(presetList.Items),
		"expression", ref.Expression)
	results := make([]any, 0, len(presetList.Items))
	for i := range presetList.Items {
		refPreset := &presetList.Items[i]
		resolvedRefValues, err := r.resolveExpressionsForPreset(ctx, refPreset, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve expressions in referenced PluginPreset %s: %w", refPreset.Name, err)
		}
		celObject := buildCELObject(refPreset.Name, refPreset.Namespace, resolvedRefValues)
		value, err := evaluateCELWithObject(ref.Expression, celObject)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate reference expression for PluginPreset %s: %w", refPreset.Name, err)
		}
		results = appendToResults(results, value)
	}
	return results, nil
}

// resolvePluginRef resolves a reference to Plugin(s).
// Supports both name-based and selector-based resolution.
func (r *PluginPresetReconciler) resolvePluginRef(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	namespace string,
) (any, error) {

	switch {
	case ref.Name != "":
		return r.resolvePluginRefByName(ctx, ref, namespace)
	case ref.Selector != nil:
		return r.resolvePluginRefBySelector(ctx, ref, namespace)
	default:
		return nil, errors.New("either name or selector must be set in valueFrom.ref for Plugin")
	}
}

// resolvePluginRefByName resolves a reference to a single Plugin by name.
func (r *PluginPresetReconciler) resolvePluginRefByName(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	namespace string,
) (any, error) {

	log := ctrl.LoggerFrom(ctx)

	refPlugin := &greenhousev1alpha1.Plugin{}
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, refPlugin); err != nil {
		return nil, fmt.Errorf("failed to get Plugin %s: %w", ref.Name, err)
	}
	log.Info("Resolving reference to Plugin by name",
		"name", ref.Name,
		"expression", ref.Expression)
	celObject := buildCELObject(refPlugin.Name, refPlugin.Namespace, refPlugin.Spec.OptionValues)
	value, err := evaluateCELWithObject(ref.Expression, celObject)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate reference expression: %w", err)
	}
	return value, nil
}

// resolvePluginRefBySelector resolves references to multiple Plugins by label selector.
// The CEL expression is evaluated against each matching Plugin and results are collected.
func (r *PluginPresetReconciler) resolvePluginRefBySelector(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	namespace string,
) (any, error) {

	log := ctrl.LoggerFrom(ctx)

	selector, err := metav1.LabelSelectorAsSelector(ref.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}
	pluginList := &greenhousev1alpha1.PluginList{}
	if err := r.List(ctx, pluginList,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		return nil, fmt.Errorf("failed to list Plugins by selector: %w", err)
	}
	if len(pluginList.Items) == 0 {
		log.Info("No Plugins found matching selector", "selector", ref.Selector)
		return []any{}, nil
	}
	log.Info("Resolving reference to Plugins by selector",
		"selector", ref.Selector,
		"matchCount", len(pluginList.Items),
		"expression", ref.Expression)
	results := make([]any, 0, len(pluginList.Items))
	for i := range pluginList.Items {
		refPlugin := &pluginList.Items[i]
		celObject := buildCELObject(refPlugin.Name, refPlugin.Namespace, refPlugin.Spec.OptionValues)
		value, err := evaluateCELWithObject(ref.Expression, celObject)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate reference expression for Plugin %s: %w", refPlugin.Name, err)
		}
		results = appendToResults(results, value)
	}
	return results, nil
}

// buildCELObject creates a flat CEL-friendly object structure from option values.
func buildCELObject(name, namespace string, optionValues []greenhousev1alpha1.PluginOptionValue) map[string]any {
	celOptionValues := make([]map[string]any, 0, len(optionValues))
	for _, ov := range optionValues {
		item := map[string]any{
			"name": ov.Name,
		}
		if ov.Value != nil && len(ov.Value.Raw) > 0 {
			var val any
			if err := json.Unmarshal(ov.Value.Raw, &val); err == nil {
				item["value"] = val
			}
		}
		celOptionValues = append(celOptionValues, item)
	}
	return map[string]any{
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"optionValues": celOptionValues,
		},
	}
}

// appendToResults appends a value to results, flattening slices to avoid nested arrays.
func appendToResults(results []any, value any) []any {
	switch v := value.(type) {
	case []any:
		results = append(results, v...)
	default:
		results = append(results, value)
	}
	return results
}

// evaluateCELWithObject evaluates a CEL expression against an object map.
// Supports two syntax styles:
//   - ${...} syntax:  ${spec.optionValues.filter(v, v.name == "foo")[0].value}
//   - Plain syntax:   spec.optionValues.filter(v, v.name == "foo")[0].value
func evaluateCELWithObject(expression string, object map[string]any) (any, error) {
	// Strip ${...} wrapper if present
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		expr = expr[2 : len(expr)-1]
	}

	env, err := celgo.NewEnv(
		celgo.Variable("spec", celgo.DynType),
		celgo.Variable("metadata", celgo.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Pass spec and metadata directly as top-level variables
	out, _, err := prg.Eval(map[string]any{
		"spec":     object["spec"],
		"metadata": object["metadata"],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return out.Value(), nil
}

// buildTemplateData creates the template data map for CEL expression evaluation
func (r *PluginPresetReconciler) buildTemplateData(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) (map[string]any, error) {

	// Create temporary Plugin to reuse existing GetGreenhouseValues
	tempPlugin := greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preset.Name,
			Namespace: preset.Namespace,
			Labels:    preset.Labels,
		},
		Spec: greenhousev1alpha1.PluginSpec{
			ClusterName: cluster.Name,
		},
	}

	// Get greenhouse values (clusterName, metadata, teams, etc.)
	greenhouseValuesList, err := helm.GetGreenhouseValues(ctx, r.Client, tempPlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to get greenhouse values: %w", err)
	}

	// Convert flat dotted keys to nested map
	// e.g., "global.greenhouse.clusterName" → map["global"]["greenhouse"]["clusterName"]
	templateData := make(map[string]any)
	for _, gv := range greenhouseValuesList {
		if gv.Value != nil {
			var value any
			if err := json.Unmarshal(gv.Value.Raw, &value); err != nil {
				return nil, fmt.Errorf("failed to unmarshal greenhouse value %s: %w", gv.Name, err)
			}
			parts := strings.Split(gv.Name, ".")
			setNestedValue(templateData, parts, value)
		}
	}

	return templateData, nil
}

// setNestedValue sets a value in a nested map using a slice of keys
func setNestedValue(m map[string]any, keys []string, value any) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		m[keys[0]] = value
		return
	}
	if _, ok := m[keys[0]]; !ok {
		m[keys[0]] = make(map[string]any)
	}
	if nested, ok := m[keys[0]].(map[string]any); ok {
		setNestedValue(nested, keys[1:], value)
	}
}
