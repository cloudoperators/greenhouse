// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PluginBundleKind is the kind of the PluginBundle resource
	PluginBundleKind = "PluginBundle"
)

// PluginBundleSpec defines the desired state of PluginBundle
type PluginBundleSpec struct {
	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// PluginDefinition is the name of the PluginDefinition this instance is for.
	PluginDefinition string `json:"pluginDefinition"`

	// OptionValues are the defaults for the Plugins deployed for this PluginPreset.
	OptionValues []PluginOptionValue `json:"optionValues,omitempty"`

	// ClusterSelector is a label selector to select the clusters the plugin bundle should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`

	// ReleaseNamespace is the namespace in the remote cluster to which the plugin is deployed.
	// Defaults to the Greenhouse managed namespace if not set.
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`
}

// PluginBundleStatus defines the observed state of PluginBundle
type PluginBundleStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PluginBundle is the Schema for the pluginbundles API
type PluginBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginBundleSpec   `json:"spec,omitempty"`
	Status PluginBundleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PluginBundleList contains a list of PluginBundle
type PluginBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PluginBundle{}, &PluginBundleList{})
}
