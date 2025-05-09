// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// Common condition types used across Greenhouse resources.
const (
	// ReadyCondition reflects the overall readiness status of a resource.
	ReadyCondition ConditionType = "Ready"

	// DeleteCondition reflects that the resource has finished its cleanup process.
	DeleteCondition ConditionType = "Delete"

	// ClusterListEmpty is set when the resources ClusterSelector results in an empty ClusterList.
	ClusterListEmpty ConditionType = "ClusterListEmpty"
)
