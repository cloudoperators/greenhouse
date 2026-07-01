// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	celgo "github.com/google/cel-go/cel"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/cel"
)

// resolvePluginOptionValuesForPreset resolves expressions and references in a PluginPreset's
// option values before writing to Plugin.
func (r *PluginPresetReconciler) resolvePluginOptionValuesForPreset(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	var resolvedValues []greenhousev1alpha1.PluginOptionValue

	if r.ExpressionEvaluationEnabled {
		var err error
		resolvedValues, err = r.resolveExpressionsForPreset(ctx, preset, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve expressions: %w", err)
		}
	} else {
		resolvedValues = util.ConvertToPluginOptionValues(preset.Spec.Plugin.OptionValues)
	}

	if r.IntegrationEnabled {
		var err error
		resolvedValues, err = r.resolveReferencesForPreset(ctx, cluster, preset.Namespace, preset.Spec.Plugin.OptionValues, resolvedValues)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve references: %w", err)
		}
	}

	return resolvedValues, nil
}

// resolveExpressionsForPreset evaluates all expression fields in PluginPreset option values.
func (r *PluginPresetReconciler) resolveExpressionsForPreset(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	hasExpressions := false
	for _, ov := range preset.Spec.Plugin.OptionValues {
		if ov.Expression != nil {
			hasExpressions = true
			break
		}
	}
	if !hasExpressions {
		return util.ConvertToPluginOptionValues(preset.Spec.Plugin.OptionValues), nil
	}

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

	greenhouseValuesList, err := helm.GetGreenhouseValues(ctx, r.Client, tempPlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to get greenhouse values: %w", err)
	}

	templateData, err := helm.BuildTemplateData(greenhouseValuesList)

	if err != nil {
		return nil, fmt.Errorf("failed to build template data: %w", err)
	}

	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(preset.Spec.Plugin.OptionValues))
	for _, optionValue := range preset.Spec.Plugin.OptionValues {
		if optionValue.Expression != nil {
			evaluatedValue, err := cel.EvaluateExpression(*optionValue.Expression, templateData)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression for option %s: %w", optionValue.Name, err)
			}
			result = append(result, greenhousev1alpha1.PluginOptionValue{
				Name:  optionValue.Name,
				Value: &apiextensionsv1.JSON{Raw: evaluatedValue},
			})
		} else {
			ov := greenhousev1alpha1.PluginOptionValue{
				Name:  optionValue.Name,
				Value: optionValue.Value,
			}
			if optionValue.ValueFrom != nil {
				ov.ValueFrom = &greenhousev1alpha1.PluginValueFromSource{
					Secret: optionValue.ValueFrom.Secret,
				}
				if optionValue.ValueFrom.Ref != nil {
					ov.ValueFrom.Ref = optionValue.ValueFrom.Ref
				}
			}
			result = append(result, ov)
		}
	}

	return result, nil
}

// resolveReferencesForPreset resolves all valueFrom.ref fields in option values.
// It reads refs from the original preset values (PluginPresetPluginOptionValue)
// and outputs resolved PluginOptionValues.
func (r *PluginPresetReconciler) resolveReferencesForPreset(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
	namespace string,
	presetOptionValues []greenhousev1alpha1.PluginPresetPluginOptionValue,
	resolvedValues []greenhousev1alpha1.PluginOptionValue,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	hasRefs := false
	for _, ov := range presetOptionValues {
		if ov.ValueFrom != nil && ov.ValueFrom.Ref != nil {
			hasRefs = true
			break
		}
	}
	if !hasRefs {
		return resolvedValues, nil
	}

	log := ctrl.LoggerFrom(ctx)
	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(presetOptionValues))

	for i, presetOV := range presetOptionValues {
		if presetOV.ValueFrom != nil && presetOV.ValueFrom.Ref != nil {
			log.Info("Resolving valueFrom.ref",
				"option", presetOV.Name,
				"refKind", presetOV.ValueFrom.Ref.Kind,
				"refName", presetOV.ValueFrom.Ref.Name)

			resolvedValue, err := r.resolveRef(ctx, presetOV.ValueFrom.Ref, cluster, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve reference for %s: %w", presetOV.Name, err)
			}

			byteVal, err := json.Marshal(resolvedValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resolved value for %s: %w", presetOV.Name, err)
			}

			result = append(result, greenhousev1alpha1.PluginOptionValue{
				Name:  presetOV.Name,
				Value: &apiextensionsv1.JSON{Raw: byteVal},
			})
		} else {
			result = append(result, resolvedValues[i])
		}
	}

	return result, nil
}

// resolveRef resolves a reference to another resource (PluginPreset or Plugin).
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
	default:
		return nil, fmt.Errorf("unsupported reference kind: %s", refKind)
	}
}

// resolvePluginPresetRef resolves a reference to PluginPreset(s).
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

	resolvedRefValues := r.resolveReferencedPresetValues(ctx, refPreset, cluster)
	celObject := buildCELObject(refPreset.Name, refPreset.Namespace, resolvedRefValues)

	value, err := evaluateCELWithObject(ref.Expression, celObject)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate reference expression: %w", err)
	}

	return value, nil
}

// resolvePluginPresetRefBySelector resolves references to multiple PluginPresets by label selector.
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
		resolvedRefValues := r.resolveReferencedPresetValues(ctx, refPreset, cluster)
		celObject := buildCELObject(refPreset.Name, refPreset.Namespace, resolvedRefValues)

		value, err := evaluateCELWithObject(ref.Expression, celObject)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate reference expression for PluginPreset %s: %w", refPreset.Name, err)
		}
		results = appendToResults(results, value)
	}

	return results, nil
}

// resolveReferencedPresetValues resolves expressions in a referenced PluginPreset
// if the ExpressionEvaluationEnabled flag is set.
func (r *PluginPresetReconciler) resolveReferencedPresetValues(
	ctx context.Context,
	refPreset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) []greenhousev1alpha1.PluginOptionValue {

	if !r.ExpressionEvaluationEnabled {
		return util.ConvertToPluginOptionValues(refPreset.Spec.Plugin.OptionValues)
	}

	// Apply cluster-specific overrides to referenced preset first
	refPresetWithOverrides := applyOverridesToPreset(refPreset, cluster.Name)

	resolvedRefValues, err := r.resolveExpressionsForPreset(ctx, refPresetWithOverrides, cluster)
	if err != nil {
		log := ctrl.LoggerFrom(ctx)
		log.Error(err, "Failed to resolve expressions in referenced PluginPreset, using raw values",
			"name", refPreset.Name)
		return util.ConvertToPluginOptionValues(refPresetWithOverrides.Spec.Plugin.OptionValues)
	}

	return resolvedRefValues
}

// buildCELObject creates a CEL-friendly object structure from option values.
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
// Supports multiple syntax styles:
//   - object.spec.optionValues.filter(...)  (legacy)
//   - spec.optionValues.filter(...)         (new)
//   - ${spec.optionValues.filter(...)}      (new with wrapper)
func evaluateCELWithObject(expression string, object map[string]any) (any, error) {
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		expr = expr[2 : len(expr)-1]
	}

	env, err := celgo.NewEnv(
		celgo.Variable("object", celgo.DynType),
		celgo.Variable("spec", celgo.DynType),
		celgo.Variable("metadata", celgo.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	evalData := map[string]any{
		"object":   object,
		"spec":     object["spec"],
		"metadata": object["metadata"],
	}

	return cel.EvaluateWithData(expr, env, evalData)
}

// applyOverridesToPreset returns a copy of the preset with cluster-specific overrides merged.
func applyOverridesToPreset(preset *greenhousev1alpha1.PluginPreset, clusterName string) *greenhousev1alpha1.PluginPreset {
	presetCopy := preset.DeepCopy()

	index := slices.IndexFunc(presetCopy.Spec.ClusterOptionOverrides, func(override greenhousev1alpha1.ClusterOptionOverride) bool {
		return override.ClusterName == clusterName
	})

	if index == -1 {
		return presetCopy
	}

	for _, overrideValue := range presetCopy.Spec.ClusterOptionOverrides[index].Overrides {
		valueIndex := slices.IndexFunc(presetCopy.Spec.Plugin.OptionValues, func(value greenhousev1alpha1.PluginPresetPluginOptionValue) bool {
			return value.Name == overrideValue.Name
		})

		if valueIndex == -1 {
			presetCopy.Spec.Plugin.OptionValues = append(presetCopy.Spec.Plugin.OptionValues, overrideValue)
		} else {
			presetCopy.Spec.Plugin.OptionValues[valueIndex] = overrideValue
		}
	}

	return presetCopy
}
