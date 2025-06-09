// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initially inline PluginDefinitionSpec to avoid duplication of fields

// ClusterPluginDefinitionStatus defines the observed state of ClusterPluginDefinition.
type ClusterPluginDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterPluginDefinition is the Schema for the clusterplugindefinitions API.
type ClusterPluginDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginDefinitionSpec          `json:"spec,omitempty"`
	Status ClusterPluginDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterPluginDefinitionList contains a list of ClusterPluginDefinition.
type ClusterPluginDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPluginDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterPluginDefinition{}, &ClusterPluginDefinitionList{})
}
