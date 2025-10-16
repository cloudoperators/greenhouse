// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

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

	// PluginSpec is the spec of the plugin to be deployed by the PluginPreset.
	Plugin PluginPresetPluginSpec `json:"plugin"`

	// ClusterSelector is a label selector to select the clusters the plugin bundle should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`

	// ClusterOptionOverrides define plugin option values to override by the PluginPreset
	// +Optional
	ClusterOptionOverrides []ClusterOptionOverride `json:"clusterOptionOverrides,omitempty"`

	// WaitFor defines other Plugins to wait for before creating the Plugin.
	WaitFor []WaitForItem `json:"waitFor,omitempty"`

	// DeletionPolicy defines how Plugins owned by a PluginPreset are handled on deletion of the PluginPreset.
	// Supported values are "Delete" and "Retain". If not set, defaults to "Delete".
	// +Optional
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// ClusterOptionOverride defines which plugin option should be override in which cluster
// +Optional
type ClusterOptionOverride struct {
	ClusterName string              `json:"clusterName"`
	Overrides   []PluginOptionValue `json:"overrides"`
}

// PluginPresetPluginSpec defines the desired state of PluginPreset's PluginSpec
type PluginPresetPluginSpec struct {
	// PluginDefinition is the name of the PluginDefinition this instance is for.
	//
	// Deprecated: Use PluginDefinitionRef instead. Future releases of greenhouse will remove this field.
	PluginDefinition string `json:"pluginDefinition"`

	// PluginDefinitionRef is the reference to the (Cluster-)PluginDefinition.
	PluginDefinitionRef PluginDefinitionReference `json:"pluginDefinitionRef"`

	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Values are the values for a PluginDefinition instance.
	OptionValues []PluginOptionValue `json:"optionValues,omitempty"`

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

const (
	// PluginSkippedCondition is set when the pluginPreset encounters a non-managed plugin.
	PluginSkippedCondition greenhousemetav1alpha1.ConditionType = "PluginSkipped"
	// PluginFailedCondition is set when the pluginPreset encounters a failure during the reconciliation of a plugin.
	PluginFailedCondition greenhousemetav1alpha1.ConditionType = "PluginFailed"
	// AllPluginsReadyCondition is set when all Plugins managed by the PluginPreset are created and ready.
	AllPluginsReadyCondition greenhousemetav1alpha1.ConditionType = "AllPluginsReady"
)

// PluginPresetStatus defines the observed state of PluginPreset
type PluginPresetStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the PluginPreset.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`

	// PluginStatuses contains statuses of Plugins managed by the PluginPreset.
	PluginStatuses []ManagedPluginStatus `json:"pluginStatuses,omitempty"`
	// TotalPlugins is the number of Plugins in total managed by the PluginPreset.
	TotalPlugins int `json:"totalPlugins,omitempty"`
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
//+kubebuilder:resource:shortName=pp
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
