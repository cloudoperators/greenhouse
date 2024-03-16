// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// MakePolicyRulesForOrganizationAdminRole returns the namespace-scoped PolicyRules for an organization admin.
func MakePolicyRulesForOrganizationAdminRole() []rbacv1.PolicyRule {
	orgAdminPolicyRules := []rbacv1.PolicyRule{
		// Grant read permissions for Clusters, PluginConfigs to organization admins.
		{
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete", "create"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"clusters", "pluginconfigs", "teams", "teammemberships"},
		},
		// Grant permissions for secrets referenced by other resources, e.g. PluginConfigs for storing sensitive values.
		// Retrieving these secrets is not permitted to the user.
		{
			Verbs:     []string{"create", "update", "patch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"secrets"},
		},
	}
	return append(MakePolicyRulesForOrganizationMemberRole(), orgAdminPolicyRules...)
}

// MakePolicyRulesForOrganizationMemberRole returns the namespace-scoped PolicyRules for an organization member.
func MakePolicyRulesForOrganizationMemberRole() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		// Grant read permissions for Clusters, PluginConfigs, Teams, TeamMemberships to organization members.
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"clusters", "pluginconfigs", "teams", "teammemberships"},
		},
	}
}

// GetTeamRoleName returns the name of the role for a team.
func GetTeamRoleName(teamName string) string {
	return fmt.Sprintf("team:%s", teamName)
}
