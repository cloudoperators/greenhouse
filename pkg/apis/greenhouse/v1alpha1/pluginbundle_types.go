// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginBundleSpec defines the desired state of PluginBundle
type PluginBundleSpec struct {
	// ClusterSelector is a label selector to select the clusters the plugin bundle should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`

	// Plugins is a list of plugins that should be deployed.
	Plugins []PluginSpec `json:"plugins"`
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
