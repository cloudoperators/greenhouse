// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	// GreenhouseOperation is the annotation used to trigger specific operations on a Greenhouse resource
	GreenhouseOperation string = "greenhouse.sap/operation"

	// GreenhouseOperationReconcile is the value used to trigger a reconcile operation on a Greenhouse resource
	GreenhouseOperationReconcile string = "reconcile"
)
