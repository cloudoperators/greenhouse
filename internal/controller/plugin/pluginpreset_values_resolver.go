// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/pkg/cel"
)

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
