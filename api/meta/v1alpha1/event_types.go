// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	// Success is used if the resource was successfully reconciled
	SuccessEvent = "Success"
	// FailedEvent is used if the resource reconciliation failed
	FailedEvent = "Failed"
	// SuccessfulDeletedEvent is used if the resource was deleted successfully
	SuccessfulDeletedEvent = "SuccessfulDeleted"
	// FailedDeleteFailedReason is used if the delete failed
	FailedDeleteEvent = "FailedDelete"
)
