// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PluginPresetKind is the kind of the PluginPreset resource
	PluginPresetKind = "PluginPreset"
)

// PluginPresetSpec defines the desired state of PluginPreset
type PluginPresetSpec struct {

	// PluginSpec is the spec of the plugin to be deployed by the PluginPreset.
	Plugin PluginSpec `json:"plugin"`

	// ClusterSelector is a label selector to select the clusters the plugin bundle should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`
}

const (
	// PluginSkippedCondition is set when the pluginPreset encounters a non-managed plugin.
	PluginSkippedCondition ConditionType = "PluginSkipped"
	// PluginFailedCondition is set when the pluginPreset encounters a failure during the reconciliation of a plugin.
	PluginFailedCondition ConditionType = "PluginFailed"
	// ClusterListEmpty is set when the PluginPreset's selector results in an empty ClusterList.
	ClusterListEmpty ConditionType = "ClusterListEmpty"
)

// PluginPresetStatus defines the observed state of PluginPreset
type PluginPresetStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the PluginPreset.
	StatusConditions `json:"statusConditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PluginPreset is the Schema for the PluginPresets API
type PluginPreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginPresetSpec   `json:"spec,omitempty"`
	Status PluginPresetStatus `json:"status,omitempty"`
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
