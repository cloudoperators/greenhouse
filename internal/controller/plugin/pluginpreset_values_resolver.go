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
	"sync"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	celgo "cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"

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
		for _, ov := range preset.Spec.Plugin.OptionValues {
			if ov.Expression != nil {
				return nil, fmt.Errorf("option %s has expression but expressionEvaluationEnabled is disabled for PluginPreset controller", ov.Name)
			}
		}
		resolvedValues = util.ConvertToPluginOptionValues(preset.Spec.Plugin.OptionValues)
	}

	if r.IntegrationEnabled {
		var err error
		resolvedValues, err = r.resolveReferencesForPreset(ctx, cluster, preset.Namespace, preset.Spec.Plugin.OptionValues, resolvedValues)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve references: %w", err)
		}
	} else {
		for _, ov := range preset.Spec.Plugin.OptionValues {
			if ov.ValueFrom != nil && ov.ValueFrom.Ref != nil {
				return nil, fmt.Errorf("option %s has valueFrom.ref but integrationEnabled is disabled for PluginPreset controller", ov.Name)
			}
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

	resolvedByName := make(map[string]greenhousev1alpha1.PluginOptionValue, len(resolvedValues))
	for _, rv := range resolvedValues {
		resolvedByName[rv.Name] = rv
	}

	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(presetOptionValues))
	for _, presetOV := range presetOptionValues {
		resolvedOV, ok := resolvedByName[presetOV.Name]
		if !ok {
			return nil, fmt.Errorf("resolved value for option %s not found", presetOV.Name)
		}
		if presetOV.ValueFrom == nil || presetOV.ValueFrom.Ref == nil {
			result = append(result, resolvedOV)
			continue
		}

		log.Info("Resolving valueFrom.ref",
			"option", presetOV.Name,
			"refKind", presetOV.ValueFrom.Ref.Kind,
			"refName", presetOV.ValueFrom.Ref.Name)

		resolvedValue, err := r.resolveRef(ctx, presetOV.ValueFrom.Ref, cluster, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference for %s: %w", presetOV.Name, err)
		}

		resolvedValue, err = mergeSelfOptionValue(resolvedOV.Value, resolvedValue)
		if err != nil {
			return nil, fmt.Errorf("failed to merge the value of %s with its reference: %w", presetOV.Name, err)
		}

		byteVal, err := json.Marshal(dropDuplicates(resolvedValue))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resolved value for %s: %w", presetOV.Name, err)
		}

		result = append(result, greenhousev1alpha1.PluginOptionValue{
			Name:  presetOV.Name,
			Value: &apiextensionsv1.JSON{Raw: byteVal},
		})
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
	case greenhousev1alpha1.PluginKind:
		return r.resolvePluginRef(ctx, ref, namespace)
	default:
		return nil, fmt.Errorf("unsupported reference kind: %s", refKind)
	}
}

// resolvePluginRef resolves a reference to Plugin
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

func (r *PluginPresetReconciler) resolvePluginRefByName(
	ctx context.Context,
	ref *greenhousev1alpha1.ExternalValueSource,
	namespace string,
) (any, error) {

	log := ctrl.LoggerFrom(ctx)

	plugin := &greenhousev1alpha1.Plugin{}
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, plugin); err != nil {
		return nil, fmt.Errorf("failed to get Plugin %s: %w", ref.Name, err)
	}

	log.Info("Resolving reference to Plugin by name",
		"name", ref.Name,
		"expression", ref.Expression)

	program, err := compileRefExpression(ref.Expression)
	if err != nil {
		return nil, err
	}
	return evaluateRef(program, plugin)
}

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
		return nil, fmt.Errorf("no Plugins found matching selector %v", ref.Selector)
	}

	slices.SortFunc(pluginList.Items, func(a, b greenhousev1alpha1.Plugin) int {
		return strings.Compare(a.Name, b.Name)
	})

	log.Info("Resolving reference to Plugins by selector",
		"selector", ref.Selector,
		"matchCount", len(pluginList.Items),
		"expression", ref.Expression)

	program, err := compileRefExpression(ref.Expression)
	if err != nil {
		return nil, err
	}

	results := make([]any, 0, len(pluginList.Items))
	for i := range pluginList.Items {
		value, err := evaluateRef(program, &pluginList.Items[i])
		if err != nil {
			return nil, err
		}
		results = appendToResults(results, value)
	}
	return results, nil
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

	resolvedPreset, err := r.resolveReferencedPresetValues(ctx, refPreset, cluster)
	if err != nil {
		return nil, err
	}

	program, err := compileRefExpression(ref.Expression)
	if err != nil {
		return nil, err
	}
	return evaluateRef(program, resolvedPreset)
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
		return nil, fmt.Errorf("no PluginPresets found matching selector %v: referenced presets may not be created yet", ref.Selector)
	}

	slices.SortFunc(presetList.Items, func(a, b greenhousev1alpha1.PluginPreset) int {
		return strings.Compare(a.Name, b.Name)
	})

	log.Info("Resolving reference to PluginPresets by selector",
		"selector", ref.Selector,
		"matchCount", len(presetList.Items),
		"expression", ref.Expression)

	program, err := compileRefExpression(ref.Expression)
	if err != nil {
		return nil, err
	}

	results := make([]any, 0, len(presetList.Items))
	for i := range presetList.Items {
		resolvedPreset, err := r.resolveReferencedPresetValues(ctx, &presetList.Items[i], cluster)
		if err != nil {
			return nil, err
		}

		value, err := evaluateRef(program, resolvedPreset)
		if err != nil {
			return nil, err
		}
		results = appendToResults(results, value)
	}

	return results, nil
}

// resolveReferencedPresetValues returns the referenced PluginPreset with overrides and expressions resolved.
func (r *PluginPresetReconciler) resolveReferencedPresetValues(
	ctx context.Context,
	refPreset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) (*greenhousev1alpha1.PluginPreset, error) {

	// Always apply cluster-specific overrides to referenced preset
	refPresetWithOverrides := applyOverridesToPreset(refPreset, cluster.Name)

	if !r.ExpressionEvaluationEnabled {
		return refPresetWithOverrides, nil
	}

	resolvedRefValues, err := r.resolveExpressionsForPreset(ctx, refPresetWithOverrides, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expression in referenced PluginPreset %s: %w",
			refPreset.Name, err)
	}
	refPresetWithOverrides.Spec.Plugin.OptionValues = util.ConvertToPresetOptionValues(resolvedRefValues)
	return refPresetWithOverrides, nil
}

// evaluateRef runs a compiled reference expression against one object, minus the spec copies in metadata.
func evaluateRef(program celgo.Program, obj client.Object) (any, error) {
	object, err := cel.StructToMap(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to build a CEL object for %s: %w", obj.GetName(), err)
	}
	if metadata, ok := object["metadata"].(map[string]any); ok {
		delete(metadata, "managedFields")
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			delete(annotations, corev1.LastAppliedConfigAnnotation)
		}
	}
	hideValueSources(object)

	value, err := cel.EvaluateProgram(program, map[string]any{
		"object":   object,
		"spec":     object["spec"],
		"metadata": object["metadata"],
		"status":   object["status"],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate the reference expression against %s: %w",
			obj.GetName(), withMissingValueHint(err, obj))
	}
	return value, nil
}

// hideValueSources drops valueFrom from every option value, so an expression can't read secret references.
func hideValueSources(object map[string]any) {
	spec, ok := object["spec"].(map[string]any)
	if !ok {
		return
	}

	optionValueLists := []any{spec["optionValues"]}
	if plugin, ok := spec["plugin"].(map[string]any); ok {
		optionValueLists = append(optionValueLists, plugin["optionValues"])
	}
	if overrides, ok := spec["clusterOptionOverrides"].([]any); ok {
		for _, override := range overrides {
			if o, ok := override.(map[string]any); ok {
				optionValueLists = append(optionValueLists, o["overrides"])
			}
		}
	}

	for _, list := range optionValueLists {
		optionValues, ok := list.([]any)
		if !ok {
			continue
		}
		for _, optionValue := range optionValues {
			if ov, ok := optionValue.(map[string]any); ok {
				delete(ov, "valueFrom")
			}
		}
	}
}

// withMissingValueHint names the options of the referenced object that hold no plain value.
func withMissingValueHint(err error, obj client.Object) error {
	var optionValues []greenhousev1alpha1.PluginOptionValue
	switch o := obj.(type) {
	case *greenhousev1alpha1.Plugin:
		optionValues = o.Spec.OptionValues
	case *greenhousev1alpha1.PluginPreset:
		optionValues = util.ConvertToPluginOptionValues(o.Spec.Plugin.OptionValues)
	}

	var missing []string
	for _, ov := range optionValues {
		if ov.Value == nil {
			missing = append(missing, ov.Name)
		}
	}
	if len(missing) == 0 {
		return err
	}
	return fmt.Errorf("%w (options with no plain value to read: %s)", err, strings.Join(missing, ", "))
}

// mergeSelfOptionValue puts the option's own value, one entry or a list, in front of the resolved one.
func mergeSelfOptionValue(selfValue *apiextensionsv1.JSON, resolvedValue any) (any, error) {
	if selfValue == nil || len(selfValue.Raw) == 0 {
		return resolvedValue, nil
	}

	var value any
	if err := json.Unmarshal(selfValue.Raw, &value); err != nil {
		return nil, err
	}
	return appendToResults(appendToResults(nil, value), resolvedValue), nil
}

// dropDuplicates keeps the first occurrence of each list entry, comparing entries as JSON.
func dropDuplicates(value any) any {
	list, ok := value.([]any)
	if !ok {
		return value
	}
	seen := make(map[string]bool, len(list))
	return slices.DeleteFunc(list, func(entry any) bool {
		key, _ := json.Marshal(entry)
		duplicate := seen[string(key)]
		seen[string(key)] = true
		return duplicate
	})
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

// refCELEnv is shared by every reference expression. ext.Lists adds sort(), a list built from a map
// comes back in random order and would rewrite the Plugin on every reconcile.
var refCELEnv = sync.OnceValues(func() (*celgo.Env, error) {
	return celgo.NewEnv(
		celgo.Variable("object", celgo.DynType),
		celgo.Variable("spec", celgo.DynType),
		celgo.Variable("metadata", celgo.DynType),
		celgo.Variable("status", celgo.DynType),
		ext.Lists(),
	)
})

// compileRefExpression compiles a valueFrom.ref expression, with or without the ${...} wrapper.
func compileRefExpression(expression string) (celgo.Program, error) {
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		expr = expr[2 : len(expr)-1]
		if strings.Contains(expr, "${") {
			return nil, fmt.Errorf("expression %q holds more than one ${...}, wrap the whole expression once instead", expression)
		}
	}

	env, err := refCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	return cel.CompileExpressionWithEnv(expr, env)
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
