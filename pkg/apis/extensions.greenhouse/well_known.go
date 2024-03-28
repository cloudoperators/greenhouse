// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package extensionsgreenhouse

const (
	// FinalizerCleanupRoleBinding is used to invoke the RoleBinding release cleanup logic.
	FinalizerCleanupRoleBinding = "extensions.greenhouse.sap/rolebinding"

	// FinalizerCleanupRole is used to invoke the Role release cleanup logic.
	FinalizerCleanupRole = "extensions.greenhouse.sap/role"

	// LabelKeyRoleBinding is the key of the label that is used to identify the RoleBinding.
	LabelKeyRoleBinding = "extensions.greenhouse.sap/rolebinding"

	// LabelKeyRole is the key of the label that is used to identify the Role.
	LabelKeyRole = "extensionsgreenhouse.sap/role"

	// RoleAndBindingNamePrefix is the prefix for the Role and RoleBinding names.
	RoleAndBindingNamePrefix = "greenhouse:"

	// RolebindingRoleRefField is the field in the RoleBinding spec that references the Role.
	RolebindingRoleRefField = ".spec.roleRef"

	// RolebindingTeamRefField is the field in the RoleBinding spec that references the Team.
	RolebindingTeamRefField = ".spec.teamRef"
)
