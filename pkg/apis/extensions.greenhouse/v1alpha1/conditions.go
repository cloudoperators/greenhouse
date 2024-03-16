// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

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
