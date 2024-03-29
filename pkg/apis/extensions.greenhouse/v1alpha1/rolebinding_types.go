// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBindingSpec defines the desired state of RoleBinding
type RoleBindingSpec struct {
	// RoleRef references a Greenhouse Role by name
	RoleRef string `json:"roleRef,omitempty"`
	// TeamRef references a Greenhouse Team by name
	TeamRef string `json:"teamRef,omitempty"`
	// ClusterName is the name of the cluster the pluginConfig is deployed to.
	ClusterName string `json:"clusterName,omitempty"`
	// Namespaces is the immutable list of namespaces in the Greenhouse Clusters to apply the RoleBinding to
	Namespaces []string `json:"namespaces,omitempty"`
}

// RoleBindingStatus defines the observed state of RoleBinding
type RoleBindingStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RoleBinding is the Schema for the rolebindings API
type RoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleBindingSpec   `json:"spec,omitempty"`
	Status RoleBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleBindingList contains a list of RoleBinding
type RoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RoleBinding{}, &RoleBindingList{})
}
