// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

const (
	// PluginPresetKind is the kind of the PluginPreset resource
	PluginPresetKind = "PluginPreset"

	// PluginReconcileFailed is set when Plugin creation or update failed.
	PluginReconcileFailed greenhousemetav1alpha1.ConditionReason = "PluginReconcileFailed"

	// PreventDeletionAnnotation is the annotation used to prevent deletion of a PluginPreset.
	// If the annotation is set the PluginPreset cannot be deleted.
	PreventDeletionAnnotation = "greenhouse.sap/prevent-deletion"
)

// PluginPresetSpec defines the desired state of PluginPreset
type PluginPresetSpec struct {
	// Plugin is a template of the spec of the plugin to be deployed by the PluginPreset.
	Plugin PluginTemplateSpec `json:"plugin"`

	// ClusterSelector is used to select a Cluster or Clusters the plugin bundle should be deployed to.
	ClusterSelector ClusterSelector `json:"clusterSelector"`

	// ClusterOptionOverrides define plugin option values to override by the PluginPreset
	// +kubebuilder:validation:Optional
	ClusterOptionOverrides []ClusterOptionOverride `json:"clusterOptionOverrides,omitempty"`
}

// PluginTemplateSpec defines the template state of Plugin to be used by PluginPreset.
type PluginTemplateSpec struct {
	// PluginDefinition is the name of the PluginDefinition this instance is for.
	PluginDefinition string `json:"pluginDefinition"`

	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Values are the values for a PluginDefinition instance.
	OptionValues []greenhousemetav1alpha1.PluginOptionValue `json:"optionValues,omitempty"`

	// ClusterName is the name of the cluster the plugin is deployed to. If not set, the plugin is deployed to the greenhouse cluster.
	ClusterName string `json:"clusterName,omitempty"`

	// ReleaseNamespace is the namespace in the remote cluster to which the backend is deployed.
	// Defaults to the Greenhouse managed namespace if not set.
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`

	// ReleaseName is the name of the helm release in the remote cluster to which the backend is deployed.
	// If the Plugin was already deployed, the Plugin's name is used as the release name.
	// If this Plugin is newly created, the releaseName is defaulted to the PluginDefinitions HelmChart name.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ReleaseName is immutable"
	// +kubebuilder:validation:MaxLength=53
	ReleaseName string `json:"releaseName,omitempty"`
}

// ClusterOptionOverride defines which plugin option should be override in which cluster
// +kubebuilder:validation:Optional
type ClusterOptionOverride struct {
	ClusterName string                                     `json:"clusterName"`
	Overrides   []greenhousemetav1alpha1.PluginOptionValue `json:"overrides"`
}

const (
	// PluginSkippedCondition is set when the pluginPreset encounters a non-managed plugin.
	PluginSkippedCondition greenhousemetav1alpha1.ConditionType = "PluginSkipped"
	// PluginFailedCondition is set when the pluginPreset encounters a failure during the reconciliation of a plugin.
	PluginFailedCondition greenhousemetav1alpha1.ConditionType = "PluginFailed"
)

// PluginPresetStatus defines the observed state of PluginPreset
type PluginPresetStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the PluginPreset.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`

	// PluginStatuses contains statuses of Plugins managed by the PluginPreset.
	PluginStatuses []ManagedPluginStatus `json:"pluginStatuses,omitempty"`
	// AvailablePlugins is the number of available Plugins managed by the PluginPreset.
	AvailablePlugins int `json:"availablePlugins,omitempty"`
	// ReadyPlugins is the number of ready Plugins managed by the PluginPreset.
	ReadyPlugins int `json:"readyPlugins,omitempty"`
	// FailedPlugins is the number of failed Plugins managed by the PluginPreset.
	FailedPlugins int `json:"failedPlugins,omitempty"`
}

// ManagedPluginStatus defines the Ready condition of a managed Plugin identified by its name.
type ManagedPluginStatus struct {
	PluginName     string                           `json:"pluginName,omitempty"`
	ReadyCondition greenhousemetav1alpha1.Condition `json:"readyCondition,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Plugin Definition",type=string,JSONPath=`.spec.plugin.pluginDefinition`
//+kubebuilder:printcolumn:name="Release Namespace",type=string,JSONPath=`.spec.plugin.releaseNamespace`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`

// PluginPreset is the Schema for the PluginPresets API
type PluginPreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginPresetSpec   `json:"spec,omitempty"`
	Status PluginPresetStatus `json:"status,omitempty"`
}

func (c *PluginPreset) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return c.Status.StatusConditions
}

func (c *PluginPreset) SetCondition(condition greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(condition)
}

//+kubebuilder:object:root=true

// PluginPresetList contains a list of PluginPresets
type PluginPresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginPreset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PluginPreset{}, &PluginPresetList{})
}
