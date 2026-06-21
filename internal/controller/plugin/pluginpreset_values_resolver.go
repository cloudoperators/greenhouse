// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/pkg/cel"
)

// resolvePluginOptionValuesForPreset resolves expressions in a PluginPreset's
// option values before writing to Plugin.
func (r *PluginPresetReconciler) resolvePluginOptionValuesForPreset(
	ctx context.Context,
	preset *greenhousev1alpha1.PluginPreset,
	cluster *greenhousev1alpha1.Cluster,
) ([]greenhousev1alpha1.PluginOptionValue, error) {

	if r.ExpressionEvaluationEnabled {
		return r.resolveExpressionsForPreset(ctx, preset, cluster)
	}
	return ConvertToPluginOptionValues(preset.Spec.Plugin.OptionValues), nil
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
		return ConvertToPluginOptionValues(preset.Spec.Plugin.OptionValues), nil
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
					Ref:    optionValue.ValueFrom.Ref,
				}
			}
			result = append(result, ov)
		}
	}
	return result, nil
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

func ConvertToPluginOptionValues(presetValues []greenhousev1alpha1.PluginPresetPluginOptionValue) []greenhousev1alpha1.PluginOptionValue {
	result := make([]greenhousev1alpha1.PluginOptionValue, 0, len(presetValues))
	for _, pv := range presetValues {
		ov := greenhousev1alpha1.PluginOptionValue{
			Name:  pv.Name,
			Value: pv.Value,
		}

		if pv.ValueFrom != nil {
			ov.ValueFrom = &greenhousev1alpha1.PluginValueFromSource{
				Secret: pv.ValueFrom.Secret,
			}
		}
		result = append(result, ov)
	}
	return result
}
