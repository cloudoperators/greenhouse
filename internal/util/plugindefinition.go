// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

// EffectivePluginDefinitionSpecFromPluginPreset returns the Spec of a PluginDefinition/ClusterPluginDefinition referenced by the PluginPreset. To be removed with the deprecated .spec.plugin.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionSpecFromPluginPreset(ctx context.Context, c client.Client, pp *greenhousev1alpha2.PluginPreset) (*greenhousemetav1alpha1.PluginDefinitionTemplateSpec, error) {
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" {
		switch pp.Spec.Plugin.PluginDefinitionRef.Kind {
		case "PluginDefinition":
			pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{
				Namespace: pp.Spec.Plugin.PluginDefinitionRef.Namespace,
				Name:      pp.Spec.Plugin.PluginDefinitionRef.Name,
			}, pluginDefinition)
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("PluginDefinition %s does not exist in namespace %s",
					pp.Spec.Plugin.PluginDefinitionRef.Name, pp.Spec.Plugin.PluginDefinitionRef.Name)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get PluginDefinition %s in namespace %s: %s",
					pp.Spec.Plugin.PluginDefinitionRef.Name, pp.Spec.Plugin.PluginDefinitionRef.Namespace, err.Error())
			}
			return &pluginDefinition.Spec, nil
		case "ClusterPluginDefinition":
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.Plugin.PluginDefinitionRef.Name}, clusterPluginDefinition)
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist",
					pp.Spec.Plugin.PluginDefinitionRef.Name)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s",
					pp.Spec.Plugin.PluginDefinitionRef.Name, err.Error())
			}
			return &clusterPluginDefinition.Spec, nil
		}
	}
	// For already existing PluginPresets get the value from the deprecated field.
	if pp.Spec.Plugin.PluginDefinition != "" {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.Plugin.PluginDefinition}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist",
				pp.Spec.Plugin.PluginDefinition)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s",
				pp.Spec.Plugin.PluginDefinition, err.Error())
		}
		return &clusterPluginDefinition.Spec, nil
	}
	return nil, errors.New("PluginDefinition not found")
}

// EffectivePluginDefinitionNameFromPluginPreset returns the name of the PluginDefinition/ClusterPluginDefinition referenced by the PluginPreset. To be removed with the deprecated .spec.plugin.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionNameFromPluginPreset(pp *greenhousev1alpha2.PluginPreset) string {
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" {
		return pp.Spec.Plugin.PluginDefinitionRef.Name
	}
	return pp.Spec.Plugin.PluginDefinition
}

// EffectivePluginDefinitionSpecFromPlugin returns the Spec of a PluginDefinition/ClusterPluginDefinition referenced by the Plugin. To be removed with the deprecated .spec.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionSpecFromPlugin(ctx context.Context, c client.Client, pp *greenhousev1alpha1.Plugin) (*greenhousemetav1alpha1.PluginDefinitionTemplateSpec, error) {
	if pp.Spec.PluginDefinitionRef.Name != "" {
		switch pp.Spec.PluginDefinitionRef.Kind {
		case "PluginDefinition":
			pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{
				Namespace: pp.Spec.PluginDefinitionRef.Namespace,
				Name:      pp.Spec.PluginDefinitionRef.Name,
			}, pluginDefinition)
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("PluginDefinition %s does not exist in namespace %s",
					pp.Spec.PluginDefinitionRef.Name, pp.Spec.PluginDefinitionRef.Name)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get PluginDefinition %s in namespace %s: %s",
					pp.Spec.PluginDefinitionRef.Name, pp.Spec.PluginDefinitionRef.Namespace, err.Error())
			}
			return &pluginDefinition.Spec, nil
		case "ClusterPluginDefinition":
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.PluginDefinitionRef.Name}, clusterPluginDefinition)
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist",
					pp.Spec.PluginDefinitionRef.Name)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s",
					pp.Spec.PluginDefinitionRef.Name, err.Error())
			}
			return &clusterPluginDefinition.Spec, nil
		}
	}
	// For already existing PluginPresets get the value from the deprecated field.
	if pp.Spec.PluginDefinition != "" {
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{Name: pp.Spec.PluginDefinition}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("ClusterPluginDefinition %s does not exist",
				pp.Spec.PluginDefinition)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get ClusterPluginDefinition %s: %s",
				pp.Spec.PluginDefinition, err.Error())
		}
		return &clusterPluginDefinition.Spec, nil
	}
	return nil, errors.New("no PluginDefinition reference found")
}

// EffectivePluginDefinitionNameFromPlugin returns the name of the PluginDefinition/ClusterPluginDefinition referenced by the Plugin. To be removed with the deprecated .spec.pluginDefinition field.
//
//nolint:staticcheck
func EffectivePluginDefinitionNameFromPlugin(pp *greenhousev1alpha1.Plugin) string {
	if pp.Spec.PluginDefinitionRef.Name != "" {
		return pp.Spec.PluginDefinitionRef.Name
	}
	return pp.Spec.PluginDefinition
}
