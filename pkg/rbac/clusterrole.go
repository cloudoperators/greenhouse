// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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

package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// MakePolicyRulesForOrganizationAdminClusterRole returns the cluster-scoped PolicyRules for an organization admin.
func MakePolicyRulesForOrganizationAdminClusterRole(organizationName string) []rbacv1.PolicyRule {
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
	return append(MakePolicyRulesForOrganizationMemberClusterRole(organizationName), orgAdminPolicyRules...)
}

// MakePolicyRulesForOrganizationMemberClusterRole returns the cluster-scoped PolicyRules for an organization member.
func MakePolicyRulesForOrganizationMemberClusterRole(organizationName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		// Grant read permissions for this Organization to its members.
		{
			Verbs:         []string{"get", "list", "watch"},
			APIGroups:     []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources:     []string{"organizations"},
			ResourceNames: []string{organizationName},
		},
		// Grant read permissions for Plugins.
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{greenhouseapisv1alpha1.GroupVersion.Group},
			Resources: []string{"plugins"},
		},
	}
}
