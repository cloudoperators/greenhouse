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
	// Namespaces is the immutable list of namespaces in the Greenhouse Clusters to apply the RoleBinding to
	Namespaces []string `json:"namespaces,omitempty"`
}

// TeamRoleBindingStatus defines the observed state of the TeamRoleBinding
type TeamRoleBindingStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
	// ClusterNotFoundReason is the event type if the cluster for a RoleBinding was not found
	ClusterNotFoundReason = "ClusterNotFound"

	// TeamNotFoundReason is the event type if the team for a RoleBinding was not found
	TeamNotFoundReason = "TeamNotFound"

	// FailedDeleteRoleBindingReason is the event type if the deletion of a RoleBinding in the remote cluster failed
	FailedDeleteRoleBindingReason = "FailedDeleteRoleBinding"

	// FailedDeleteClusterRoleBindingReason is the event type if the deletion of a RoleBinding in the remote cluster failed
	FailedDeleteClusterRoleBindingReason = "FailedDeleteClusterRoleBinding"

	// FailedReconcileRoleReason is the event type if the reconciliation of a ClusterRole in the remote cluster failed
	FailedReconcileClusterRoleReason = "FailedReconcileClusterRole"

	// FailedReconcileRoleReason is the event type if the reconciliation of a Role in the remote cluster failed
	FailedReconcileRoleReason = "FailedReconcileRole"

	// FailedReconcileRoleBindingReason is the event type if the reconciliation of a RoleBinding in the remote cluster failed
	FailedReconcileRoleBindingReason = "FailedReconcileRoleBinding"

	// FailedReconcileClusterRoleBindingReason is the event type if the reconciliation of a ClusterRoleBinding in the remote cluster failed
	FailedReconcileClusterRoleBindingReason = "FailedReconcileClusterRoleBinding"
)
