// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// initially inline PluginDefinitionSpec to avoid duplication of fields

const (
	// ClusterPluginDefinitionKind is the kind of the ClusterPluginDefinition resource
	ClusterPluginDefinitionKind = "ClusterPluginDefinition"
)

// ClusterPluginDefinitionStatus defines the observed state of ClusterPluginDefinition.
type ClusterPluginDefinitionStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the Plugin.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=cpd
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type=="Ready")].message`

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

func (c *ClusterPluginDefinition) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return c.Status.StatusConditions
}

func (c *ClusterPluginDefinition) SetCondition(condition greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(condition)
}

func init() {
	SchemeBuilder.Register(&ClusterPluginDefinition{}, &ClusterPluginDefinitionList{})
}
