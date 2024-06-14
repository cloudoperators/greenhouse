// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
)

// TeamRoleBindingSpec defines the desired state of a TeamRoleBinding
type TeamRoleBindingSpec struct {
	// TeamRoleRef references a Greenhouse TeamRole by name
	TeamRoleRef string `json:"teamRoleRef,omitempty"`
	// TeamRef references a Greenhouse Team by name
	TeamRef string `json:"teamRef,omitempty"`
	// ClusterName is the name of the cluster the rbacv1 resources are created on.
	ClusterName string `json:"clusterName,omitempty"`
	// ClusterSelector is a label selector to select the Clusters the TeamRoleBinding should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`
	// Namespaces is the immutable list of namespaces in the Greenhouse Clusters to apply the RoleBinding to.
	// If empty, a ClusterRoleBinding will be created on the remote cluster, otherwise a RoleBinding per namespace.
	Namespaces []string `json:"namespaces,omitempty"`
}

// TeamRoleBindingStatus defines the observed state of the TeamRoleBinding
type TeamRoleBindingStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the TeamRoleBinding.
	StatusConditions `json:"statusConditions,omitempty"`
	// PropagationStatus is the list of clusters the TeamRoleBinding is applied to
	// +listType="map"
	// +listMapKey=clusterName
	PropagationStatus []PropagationStatus `json:"clusters,omitempty"`
}

// PropagationStatus defines the observed state of the TeamRoleBinding's associated rbacv1 resources  on a Cluster
type PropagationStatus struct {
	// ClusterName is the name of the cluster the rbacv1 resources are created on.
	ClusterName string `json:"clusterName"`
	// Condition is the overall Status of the rbacv1 resources created on the cluster
	Condition `json:"condition,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Team Role",type=string,JSONPath=`.spec.teamRoleRef`
//+kubebuilder:printcolumn:name="Team",type=string,JSONPath=`.spec.teamRef`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TeamRoleBinding is the Schema for the rolebindings API
type TeamRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamRoleBindingSpec   `json:"spec,omitempty"`
	Status TeamRoleBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamRoleBindingList contains a list of RoleBinding
type TeamRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TeamRoleBinding{}, &TeamRoleBindingList{})
}

// GetRBACName returns the name of the rbacv1.RoleBinding or rbacv1.ClusterRoleBinding that will be created on the remote cluster
func (trb *TeamRoleBinding) GetRBACName() string {
	return greenhouseapis.RBACPrefix + trb.GetName()
}

const (
	// RBACReady is the condition type for the TeamRoleBinding when the rbacv1 resources are ready
	RBACReady ConditionType = "RBACReady"

	// RBACReconciled is the condition reason for the TeamRoleBinding when the rbacv1 resources are successfully reconciled
	RBACReconciled ConditionReason = "RBACReconciled"

	// RBACReconcileFailed is the condition reason for the TeamRoleBinding when not all of the rbacv1 resources have been successfully reconciled
	RBACReconcileFailed ConditionReason = "RBACReconcileFailed"

	// ClusterConnectionFailed is the condition reason for the TeamRoleBinding when the connection to the cluster failed
	ClusterConnectionFailed ConditionReason = "ClusterConnectionFailed"

	// ClusterRoleFailed is the condition reason for the TeamRoleBinding when the ClusterRole could not be created
	ClusterRoleFailed ConditionReason = "ClusterRoleFailed"

	// RoleBindingFailed is the condition reason for the TeamRoleBinding when the RoleBinding could not be created
	RoleBindingFailed ConditionReason = "RoleBindingFailed"
)

const (
	// ReconcileFailedReason is used if the reconcile failed
	ReconcileFailedReason = "FailedReconcile"
	// DeletedReason is used if the resource was deleted
	DeletedReason = "Deleted"
	// FailedDeleteFailedReason is used if the delete failed
	DeleteFailedReason = "FailedDelete"
)
