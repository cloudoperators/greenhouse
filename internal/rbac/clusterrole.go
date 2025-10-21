// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// OrganizationAdminClusterRolePolicyRules returns the cluster-scoped PolicyRules for an organization admin.
func OrganizationAdminClusterRolePolicyRules(organizationName string) []rbacv1.PolicyRule {
	orgAdminPolicyRules := []rbacv1.PolicyRule{
		// Grant extensive permissions for this Organization to its administrators.
		// Creation and deletion is only permitted for Greenhouse administrators though.
		{
			Verbs:         []string{"get", "list", "watch", "update", "patch"},
			APIGroups:     []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources:     []string{"organizations"},
			ResourceNames: []string{organizationName},
		},
	}
	return append(OrganizationMemberClusterRolePolicyRules(organizationName), orgAdminPolicyRules...)
}

// OrganizationMemberClusterRolePolicyRules returns the cluster-scoped PolicyRules for an organization member.
func OrganizationMemberClusterRolePolicyRules(organizationName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		// Grant read permissions for this Organization to its members.
		{
			Verbs:         []string{"get", "list", "watch"},
			APIGroups:     []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources:     []string{"organizations"},
			ResourceNames: []string{organizationName},
		},
		// Grant read permissions for PluginDefinitions.
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"clusterplugindefinitions"},
		},
	}
}
