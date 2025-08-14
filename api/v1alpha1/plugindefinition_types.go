// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// PluginDefinitionStatus defines the observed state of PluginDefinition
type PluginDefinitionStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the Plugin.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PluginDefinition is the Schema for the PluginDefinitions API
type PluginDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   greenhousemetav1alpha1.PluginDefinitionTemplateSpec `json:"spec,omitempty"`
	Status PluginDefinitionStatus                              `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PluginDefinitionList contains a list of PluginDefinition
type PluginDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginDefinition `json:"items"`
}

func (p *PluginDefinition) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return p.Status.StatusConditions
}

func (p *PluginDefinition) SetCondition(condition greenhousemetav1alpha1.Condition) {
	p.Status.SetConditions(condition)
}

func init() {
	SchemeBuilder.Register(&PluginDefinition{}, &PluginDefinitionList{})
}
