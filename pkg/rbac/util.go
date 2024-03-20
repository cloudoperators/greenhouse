// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"fmt"
)

// OrganizationAdminRoleName returns the name of the admin role for an organization.
func OrganizationAdminRoleName(orgName string) string {
	return fmt.Sprintf("role:%s:admin", orgName)
}

// OrganizationClusterAdminRoleName returns the name of the cluster admin role for an organization.
func OrganizationClusterAdminRoleName(orgName string) string {
	return fmt.Sprintf("role:%s:cluster-admin", orgName)
}

// GetOrganizationPluginAdminRoleName returns the name of the plugin admin role for an organization.
func OrganizationPluginAdminRoleName(orgName string) string {
	return fmt.Sprintf("role:%s:plugin-admin", orgName)
}

// OrganizationRoleName returns the name of the role for an organization.
func OrganizationRoleName(orgName string) string {
	return fmt.Sprintf("organization:%s", orgName)
}
