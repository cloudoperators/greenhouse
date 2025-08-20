// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/cloudoperators/greenhouse/api/v1alpha2"
)

// ConvertTo converts this version to the Hub version. See: https://book.kubebuilder.io/multiversion-tutorial/conversion
func (pp *PluginPreset) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.PluginPreset) //nolint:errcheck

	// Convert old selectors to the new ClusterSelector.
	dst.Spec.ClusterSelector = v1alpha2.ClusterSelector{
		Name:          pp.Spec.ClusterName,
		LabelSelector: pp.Spec.ClusterSelector,
	}

	// Rote conversion.

	// ObjectMeta
	dst.ObjectMeta = pp.ObjectMeta

	// Spec
	dst.Spec.Plugin = v1alpha2.PluginTemplateSpec(pp.Spec.Plugin)
	dstClusterOptionOverrides := make([]v1alpha2.ClusterOptionOverride, len(pp.Spec.ClusterOptionOverrides))
	for i, v := range pp.Spec.ClusterOptionOverrides {
		dstClusterOptionOverrides[i] = v1alpha2.ClusterOptionOverride{
			ClusterName: v.ClusterName,
			Overrides:   v.Overrides,
		}
	}
	dst.Spec.ClusterOptionOverrides = dstClusterOptionOverrides

	// Status
	dst.Status.StatusConditions = pp.Status.StatusConditions
	dstPluginStatuses := make([]v1alpha2.ManagedPluginStatus, len(pp.Status.PluginStatuses))
	for i, v := range pp.Status.PluginStatuses {
		dstPluginStatuses[i] = v1alpha2.ManagedPluginStatus{
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

// ConvertFrom converts from the Hub version (v1alpha2) to this version.
func (pp *PluginPreset) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.PluginPreset) //nolint:errcheck

	// Convert the new ClusterSelector to the old selector.
	pp.Spec.ClusterName = src.Spec.ClusterSelector.Name
	pp.Spec.ClusterSelector = src.Spec.ClusterSelector.LabelSelector

	// Rote conversion.

	// ObjectMeta
	pp.ObjectMeta = src.ObjectMeta

	// Spec
	pp.Spec.Plugin = PluginSpec(src.Spec.Plugin)
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
