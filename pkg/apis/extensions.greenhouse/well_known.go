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

	// RolebindingRoleRefField is the field in the RoleBinding spec that references the Role.
	RolebindingRoleRefField = ".spec.roleRef"

	// RolebindingTeamRefField is the field in the RoleBinding spec that references the Team.
	RolebindingTeamRefField = ".spec.teamRef"
)
