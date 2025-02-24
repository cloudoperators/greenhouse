// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// OrganizationAdminPolicyRules returns the namespace-scoped PolicyRules for an organization admin.
func OrganizationAdminPolicyRules() []rbacv1.PolicyRule {
	orgAdminPolicyRules := []rbacv1.PolicyRule{
		// Grant read permissions for Clusters, Plugins to organization admins.
		{
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete", "create"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"teams", "teammemberships"},
		},
		// Grant permissions for secrets referenced by other resources, e.g. Plugins for storing sensitive values.
		// Retrieving these secrets is not permitted to the user.
		{
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"secrets"},
		},
		// Grant permission to create RoleBindings
		{
			Verbs:     []string{"create"},
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"rolebindings"},
		},
		// Grant permission to view Alertmanager and AlertmanagerConfig resources
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{"monitoring.coreos.com"},
			Resources: []string{"alertmanagers", "alertmanagerconfigs"},
		},
		// Grant permission to view Pods, ReplicaSets, Deployments, StatefulSets, DaemonSets, CronJobs, Jobs, ConfigMaps
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{""},
			Resources: []string{"pods", "replicasets", "deployments", "statefulsets", "daemonsets", "cronjobs", "jobs", "configmaps"},
		},
	}
	orgAdminPolicyRules = append(orgAdminPolicyRules,
		OrganizationClusterAdminPolicyRules()...)
	return append(orgAdminPolicyRules, OrganizationPluginAdminPolicyRules()...)
}

// OrganizationClusterAdminPolicyRules returns the namespace-scoped PolicyRules for an organization cluster admin.
func OrganizationClusterAdminPolicyRules() []rbacv1.PolicyRule {
	policyRules := []rbacv1.PolicyRule{
		// Grant CRUD Permissions for Clusters, TeamRoles and TeamRoleBindings
		{
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete", "create"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"clusters", "teamroles", "teamrolebindings"},
		},
		// Grant permissions for secrets referenced by other resources, e.g. Plugins for storing sensitive values.
		// Retrieving these secrets is not permitted to the user.
		{
			Verbs:     []string{"create", "update", "patch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"secrets"},
		},
	}
	return append(OrganizationMemberPolicyRules(), policyRules...)
}

// OrganizationPluginAdminPolicyRules returns the namespace-scoped PolicyRules for an organization plugin admin.
func OrganizationPluginAdminPolicyRules() []rbacv1.PolicyRule {
	policyRules := []rbacv1.PolicyRule{
		// Grant read, create, update and delete permissions for PluginPresets to organization plugin admins.
		{
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete", "create"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"pluginpresets"},
		},
		// Grant read, update and delete permissions for Plugins to organization plugin admins. No create
		{
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"plugins"},
		},
		// Grant permissions for secrets referenced by other resources, e.g. Plugins for storing sensitive values.
		// Retrieving these secrets is not permitted to the user.
		{
			Verbs:     []string{"create", "update", "patch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"secrets"},
		},
	}
	return append(OrganizationMemberPolicyRules(), policyRules...)
}

// OrganizationMemberPolicyRules returns the namespace-scoped PolicyRules for an organization member.
func OrganizationMemberPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		// Grant read permissions for Clusters, Plugins, Teams, TeamMemberships to organization members.
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"clusters", "plugins", "pluginpresets", "teams", "teammemberships", "teamroles", "teamrolebindings"},
		},
	}
}

// GetTeamRoleName returns the name of the role for a team.
func GetTeamRoleName(teamName string) string {
	return "team:" + teamName
}
