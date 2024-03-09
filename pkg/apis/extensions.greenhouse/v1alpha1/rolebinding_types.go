// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// ClusterSelector is the immutable selector to deterimine the Greenhouse Clusters to apply the RoleBinding to
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`
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
