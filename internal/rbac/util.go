// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
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
	return "organization:" + orgName
}

// OrganizationPluginDefinitionCatalogServiceAccountName returns the name of the ServiceAccount for PluginDefinitionCatalog operations.
func OrganizationPluginDefinitionCatalogServiceAccountName(orgName string) string {
	return orgName + "-plugin-definition-catalog"
}

// OrganizationPluginDefinitionCatalogRoleName returns the name of the Role for PluginDefinitionCatalog operations.
func OrganizationPluginDefinitionCatalogRoleName(orgName string) string {
	return orgName + "-plugin-definition-catalog"
}
