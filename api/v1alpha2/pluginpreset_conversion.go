// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	"github.com/cloudoperators/greenhouse/api/v1alpha3"
)

// ConvertTo converts this version to the Hub version.
func (pp *PluginPreset) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha3.PluginPreset) //nolint:errcheck

	// Convert PluginSpec to PluginTemplateSpec.
	dst.Spec.Plugin = v1alpha3.PluginTemplateSpec{
		PluginDefinitionRef: greenhousemetav1alpha1.PluginDefinitionReference{
			Name:      pp.Spec.Plugin.PluginDefinition,
			Namespace: pp.Spec.Plugin.PluginDefinitionNamespace,
		},
		DisplayName:      pp.Spec.Plugin.DisplayName,
		OptionValues:     pp.Spec.Plugin.OptionValues,
		ClusterName:      pp.Spec.Plugin.ClusterName,
		ReleaseNamespace: pp.Spec.Plugin.ReleaseNamespace,
		ReleaseName:      pp.Spec.Plugin.ReleaseName,
	}

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = pp.ObjectMeta

	// Spec
	dst.Spec.ClusterSelector = pp.Spec.ClusterSelector
	dstClusterOptionOverrides := make([]v1alpha3.ClusterOptionOverride, len(pp.Spec.ClusterOptionOverrides))
	for i, v := range pp.Spec.ClusterOptionOverrides {
		dstClusterOptionOverrides[i] = v1alpha3.ClusterOptionOverride{
			ClusterName: v.ClusterName,
			Overrides:   v.Overrides,
		}
	}
	dst.Spec.ClusterOptionOverrides = dstClusterOptionOverrides

	// Status
	dst.Status.StatusConditions = pp.Status.StatusConditions
	dstPluginStatuses := make([]v1alpha3.ManagedPluginStatus, len(pp.Status.PluginStatuses))
	for i, v := range pp.Status.PluginStatuses {
		dstPluginStatuses[i] = v1alpha3.ManagedPluginStatus{
			PluginName:     v.PluginName,
			ReadyCondition: v.ReadyCondition,
		}
	}
	dst.Status.PluginStatuses = dstPluginStatuses
	dst.Status.AvailablePlugins = pp.Status.AvailablePlugins
	dst.Status.ReadyPlugins = pp.Status.ReadyPlugins
	dst.Status.FailedPlugins = pp.Status.FailedPlugins

	return nil
}

// ConvertFrom converts from the Hub version to this version.
func (pp *PluginPreset) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha3.PluginPreset) //nolint:errcheck

	// Convert PluginTemplateSpec.
	pp.Spec.Plugin = PluginTemplateSpec{
		PluginDefinition:          src.Spec.Plugin.PluginDefinitionRef.Name,
		PluginDefinitionNamespace: src.Spec.Plugin.PluginDefinitionRef.Namespace,
		DisplayName:               src.Spec.Plugin.DisplayName,
		OptionValues:              src.Spec.Plugin.OptionValues,
		ClusterName:               src.Spec.Plugin.ClusterName,
		ReleaseNamespace:          src.Spec.Plugin.ReleaseNamespace,
		ReleaseName:               src.Spec.Plugin.ReleaseName,
	}

	// Rote conversion.

	// ObjectMeta
	pp.ObjectMeta = src.ObjectMeta

	// Spec
	pp.Spec.ClusterSelector = src.Spec.ClusterSelector
	dstClusterOptionOverrides := make([]ClusterOptionOverride, len(src.Spec.ClusterOptionOverrides))
	for i, v := range src.Spec.ClusterOptionOverrides {
		dstClusterOptionOverrides[i] = ClusterOptionOverride{
			ClusterName: v.ClusterName,
			Overrides:   v.Overrides,
		}
	}
	pp.Spec.ClusterOptionOverrides = dstClusterOptionOverrides

	// Status
	pp.Status.StatusConditions = src.Status.StatusConditions
	dstPluginStatuses := make([]ManagedPluginStatus, len(src.Status.PluginStatuses))
	for i, v := range src.Status.PluginStatuses {
		dstPluginStatuses[i] = ManagedPluginStatus{
			PluginName:     v.PluginName,
			ReadyCondition: v.ReadyCondition,
		}
	}
	pp.Status.PluginStatuses = dstPluginStatuses
	pp.Status.AvailablePlugins = src.Status.AvailablePlugins
	pp.Status.ReadyPlugins = src.Status.ReadyPlugins
	pp.Status.FailedPlugins = src.Status.FailedPlugins

	return nil
}
