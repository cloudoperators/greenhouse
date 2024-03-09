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
