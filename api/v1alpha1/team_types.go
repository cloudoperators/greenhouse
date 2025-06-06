// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

const (
	// SCIMAccessReadyCondition reflects if there is a connection to SCIM.
	SCIMAccessReadyCondition greenhousemetav1alpha1.ConditionType = "SCIMAccessReady"
	// SCIMAPIUnavailableReason is set when the organization has set SCIMAPIAvailableCondition to false.
	SCIMAPIUnavailableReason greenhousemetav1alpha1.ConditionReason = "SCIMAPIUnavailable"
	// SCIMAllMembersValidCondition reflects if all members are valid. It is set to false if there are invalid or inactive members.
	SCIMAllMembersValidCondition greenhousemetav1alpha1.ConditionType = "SCIMAllMembersValid"
)

// User specifies a human person.
type User struct {
	// ID is the unique identifier of the user.
	ID string `json:"id"`
	// FirstName of the user.
	FirstName string `json:"firstName"`
	// LastName of the user.
	LastName string `json:"lastName"`
	// Email of the user.
	Email string `json:"email"`
}

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
	StatusConditions greenhousemetav1alpha1.StatusConditions `json:"statusConditions"`
	Members          []User                                  `json:"members,omitempty"`
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

func (o *Team) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return o.Status.StatusConditions
}

func (o *Team) SetCondition(condition greenhousemetav1alpha1.Condition) {
	o.Status.StatusConditions.SetConditions(condition)
}
