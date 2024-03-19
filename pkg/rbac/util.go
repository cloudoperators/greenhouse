// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"fmt"
)

// GetAdminRoleNameForOrganization returns the name of the admin role for an organization.
func GetAdminRoleNameForOrganization(orgName string) string {
	return fmt.Sprintf("role:%s:admin", orgName)
}

// GetOrganizationRoleName returns the name of the role for an organization.
func GetOrganizationRoleName(orgName string) string {
	return fmt.Sprintf("organization:%s", orgName)
}
