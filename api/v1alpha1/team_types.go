// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	// Description provides additional details of the team.
	Description string `json:"description,omitempty"`
	// IdP group id matching team.
	MappedIDPGroup string `json:"mappedIdPGroup,omitempty"`
	// URL to join the IdP group.
	JoinURL string `json:"joinUrl,omitempty"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	StatusConditions greenhouseapis.StatusConditions `json:"statusConditions"`
	Members          []User                          `json:"members,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
//+kubebuilder:printcolumn:name="IDP Group",type=string,JSONPath=`.spec.mappedIdPGroup`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="SCIM Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "SCIMAccessReady")].status`

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}

func (o *Team) GetConditions() greenhouseapis.StatusConditions {
	return o.Status.StatusConditions
}

func (o *Team) SetCondition(condition greenhouseapis.Condition) {
	o.Status.StatusConditions.SetConditions(condition)
}
