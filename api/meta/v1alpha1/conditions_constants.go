// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// Common condition types and reasons used across Greenhouse resources.
const (
	// ReadyCondition reflects the overall readiness status of a resource.
	ReadyCondition ConditionType = "Ready"

	// DeleteCondition reflects that the resource has finished its cleanup process.
	DeleteCondition ConditionType = "Delete"

	// ClusterListEmpty is set when the resources ClusterSelector results in an empty ClusterList.
	ClusterListEmpty ConditionType = "ClusterListEmpty"

	// OwnerLabelSetCondition reflects that the resource has the owned-by label set to an existing support-group Team.
	OwnerLabelSetCondition ConditionType = "OwnerLabelSet"
	// OwnerLabelMissingReason is set when the resource is missing the owned-by label.
	OwnerLabelMissingReason ConditionReason = "OwnerLabelMissing"
	// OwnerLabelSetToNotExistingTeamReason is set when the resource has the owned-by label set to a non-existing Team.
	OwnerLabelSetToNotExistingTeamReason ConditionReason = "OwnerLabelNotExistingTeam"
	// OwnerLabelSetToNonSupportGroupTeamReason is set when the resource has the owned-by label set to a non-support-group Team.
	OwnerLabelSetToNonSupportGroupTeamReason ConditionReason = "OwnerLabelSetToNonSupportGroupTeam"

	// SuspendedCondition reflects that the resource is suspended.
	SuspendedCondition ConditionType = "Suspended"
	// ResourceSuspendedReason is set when the resource is successfully suspended.
	ResourceSuspendedReason ConditionReason = "ResourceSuspended"
	// ResourceSuspensionFailedReason is set when the resource suspension failed.
	ResourceSuspensionFailedReason ConditionReason = "ResourceSuspensionFailed"
)
