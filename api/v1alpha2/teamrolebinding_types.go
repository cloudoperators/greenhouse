// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// TeamRoleBindingSpec defines the desired state of a TeamRoleBinding
type TeamRoleBindingSpec struct {
	// TeamRoleRef references a Greenhouse TeamRole by name
	TeamRoleRef string `json:"teamRoleRef,omitempty"`
	// TeamRef references a Greenhouse Team by name
	TeamRef string `json:"teamRef,omitempty"`
	// Usernames defines list of users to add to the (Cluster-)RoleBindings
	Usernames []string `json:"usernames,omitempty"`
	// ClusterSelector is used to select a Cluster or Clusters the TeamRoleBinding should be deployed to.
	ClusterSelector ClusterSelector `json:"clusterSelector,omitempty"`
	// Namespaces is a list of namespaces in the Greenhouse Clusters to apply the RoleBinding to.
	// If empty, a ClusterRoleBinding will be created on the remote cluster, otherwise a RoleBinding per namespace.
	Namespaces []string `json:"namespaces,omitempty"`
	// CreateNamespaces when enabled the controller will create namespaces for RoleBindings if they do not exist.
	// +kubebuilder:default:=false
	CreateNamespaces bool `json:"createNamespaces,omitempty"`
}

// TeamRoleBindingStatus defines the observed state of the TeamRoleBinding
type TeamRoleBindingStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the TeamRoleBinding.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
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
	greenhousemetav1alpha1.Condition `json:"condition,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
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

func (trb *TeamRoleBinding) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return trb.Status.StatusConditions
}

func (trb *TeamRoleBinding) SetCondition(condition greenhousemetav1alpha1.Condition) {
	trb.Status.SetConditions(condition)
}

// SetPropagationStatus updates the TeamRoleBinding's PropagationStatus for the Cluster
func (trb *TeamRoleBinding) SetPropagationStatus(cluster string, rbacReady metav1.ConditionStatus, reason greenhousemetav1alpha1.ConditionReason, message string) {
	condition := greenhousemetav1alpha1.NewCondition(RBACReady, rbacReady, reason, message)
	for i, ps := range trb.Status.PropagationStatus {
		if ps.ClusterName != cluster {
			continue
		}
		if ps.Status == rbacReady {
			// Set the LastTransitionTime to its previous value if the status did not change.
			condition.LastTransitionTime = ps.LastTransitionTime
		}
		trb.Status.PropagationStatus[i].Condition = condition
		return
	}
	condition.LastTransitionTime = metav1.Now()
	trb.Status.PropagationStatus = append(trb.Status.PropagationStatus, PropagationStatus{
		ClusterName: cluster,
		Condition:   condition,
	})
}

// RemovePropagationStatus removes a condition for the Cluster from TeamRoleBinding's PropagationStatus
func (trb *TeamRoleBinding) RemovePropagationStatus(cluster string) {
	updatedStatus := slices.DeleteFunc(trb.Status.PropagationStatus, func(ps PropagationStatus) bool {
		return ps.ClusterName == cluster
	})
	trb.Status.PropagationStatus = updatedStatus
}

// GetRBACName returns the name of the rbacv1.RoleBinding or rbacv1.ClusterRoleBinding that will be created on the remote cluster
func (trb *TeamRoleBinding) GetRBACName() string {
	return greenhouseapis.RBACPrefix + trb.GetName()
}

const (
	// RBACReady is the condition type for the TeamRoleBinding when the rbacv1 resources are ready
	RBACReady greenhousemetav1alpha1.ConditionType = "RBACReady"

	// RBACReconciled is the condition reason for the TeamRoleBinding when the rbacv1 resources are successfully reconciled
	RBACReconciled greenhousemetav1alpha1.ConditionReason = "RBACReconciled"

	// RBACReconcileFailed is the condition reason for the TeamRoleBinding when not all of the rbacv1 resources have been successfully reconciled
	RBACReconcileFailed greenhousemetav1alpha1.ConditionReason = "RBACReconcileFailed"

	// EmptyClusterList is the condition reason for a resource when the clusterSelector and clusterName do not provide any existing cluster
	EmptyClusterList greenhousemetav1alpha1.ConditionReason = "EmptyClusterList"

	// TeamNotFound is the condition reason when the resources refers to a non-existing Team
	TeamNotFound greenhousemetav1alpha1.ConditionReason = "TeamNotFound"

	// ClusterConnectionFailed is the condition reason for the TeamRoleBinding when the connection to the cluster failed
	ClusterConnectionFailed greenhousemetav1alpha1.ConditionReason = "ClusterConnectionFailed"

	// ClusterRoleFailed is the condition reason for the TeamRoleBinding when the ClusterRole could not be created
	ClusterRoleFailed greenhousemetav1alpha1.ConditionReason = "ClusterRoleFailed"

	// RoleBindingFailed is the condition reason for the TeamRoleBinding when the RoleBinding could not be created
	RoleBindingFailed greenhousemetav1alpha1.ConditionReason = "RoleBindingFailed"

	// CreateNamespacesFailed is the condition reason for the TeamRoleBinding when the namespaces could not be created
	CreateNamespacesFailed greenhousemetav1alpha1.ConditionReason = "CreateNamespacesFailed"
)
