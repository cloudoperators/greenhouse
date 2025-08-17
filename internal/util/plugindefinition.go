// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

// EffectivePluginDefinitionSpec returns the Spec of a PluginDefinition/ClusterPluginDefinition referenced by the PluginPreset. To be removed with the deprecated .spec.plugin.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionSpec(ctx context.Context, c client.Client, pp *greenhousev1alpha2.PluginPreset) (*greenhousemetav1alpha1.PluginDefinitionTemplateSpec, error) {
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" {
		switch pp.Spec.Plugin.PluginDefinitionRef.Kind {
		case "PluginDefinition":
			pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{
				Namespace: pp.Spec.Plugin.PluginDefinitionRef.Namespace,
				Name:      pp.Spec.Plugin.PluginDefinitionRef.Name,
			}, pluginDefinition)
			if err != nil {
				return nil, err
			}
			return &pluginDefinition.Spec, nil
		case "ClusterPluginDefinition":
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.Plugin.PluginDefinitionRef.Name}, clusterPluginDefinition)
			if err != nil {
				return nil, err
			}
			return &clusterPluginDefinition.Spec, nil
		}
	}
	// For already existing PluginPresets get the value from the deprecated field.
	if pp.Spec.Plugin.PluginDefinition != "" {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.Plugin.PluginDefinition}, clusterPluginDefinition)
		if err != nil {
			return nil, err
		}
		return &clusterPluginDefinition.Spec, nil
	}
	return nil, errors.New("no PluginDefinition reference found")
}

// EffectivePluginDefinitionName returns the name of the PluginDefinition/ClusterPluginDefinition referenced by the PluginPreset. To be removed with the deprecated .spec.plugin.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionName(pp *greenhousev1alpha2.PluginPreset) string {
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" {
		return pp.Spec.Plugin.PluginDefinitionRef.Name
	}
	return pp.Spec.Plugin.PluginDefinition
}
