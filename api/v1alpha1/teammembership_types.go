// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SCIMAccessReadyCondition reflects if there is a connection to SCIM.
	SCIMAccessReadyCondition greenhouseapis.ConditionType = "SCIMAccessReady"
	// SCIMAPIUnavailableReason is set when the organization has set SCIMAPIAvailableCondition to false.
	SCIMAPIUnavailableReason greenhouseapis.ConditionReason = "SCIMAPIUnavailable"
	// SCIMAllMembersValidCondition reflects if all members are valid. It is set to false if there are invalid or inactive members.
	SCIMAllMembersValidCondition greenhouseapis.ConditionType = "SCIMAllMembersValid"
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

// TeamMembershipSpec defines the desired state of TeamMembership
type TeamMembershipSpec struct {
	// Members list users that are part of a team.
	// +optional
	Members []User `json:"members,omitempty"`
}

// TeamMembershipStatus defines the observed state of TeamMembership
type TeamMembershipStatus struct {
	// LastSyncedTime is the information when was the last time the membership was synced
	// +optional
	LastSyncedTime *metav1.Time `json:"lastSyncedTime,omitempty"`
	// LastChangedTime is the information when was the last time the membership was actually changed
	// +optional
	LastChangedTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	// StatusConditions contain the different conditions that constitute the status of the TeamMembership.
	greenhouseapis.StatusConditions `json:"statusConditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TeamMembership is the Schema for the teammemberships API
type TeamMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamMembershipSpec   `json:"spec,omitempty"`
	Status TeamMembershipStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamMembershipList contains a list of TeamMembership
type TeamMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamMembership `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TeamMembership{}, &TeamMembershipList{})
}
