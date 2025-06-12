// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// WithRules overrides the default rules of a TeamRole
func WithRules(rules []rbacv1.PolicyRule) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.Rules = rules
	}
}

// WithAggregationRule sets the AggregationRule on a TeamRole
func WithAggregationRule(aggregationRule *rbacv1.AggregationRule) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.AggregationRule = aggregationRule
	}
}

// WithLabels sets the .spec.Labels on a TeamRole
func WithLabels(labels map[string]string) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.Labels = labels
	}
}

// NewTeamRole returns a greenhousev1alpha1.TeamRole object. Opts can be used to set the desired state of the TeamRole.
func NewTeamRole(name, namespace string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
	tr := &greenhousev1alpha1.TeamRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRole",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.TeamRoleSpec{
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(tr)
	}
	return tr
}

func WithTeamRoleRef(roleRef string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.TeamRoleRef = roleRef
	}
}

func WithTeamRef(teamRef string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.TeamRef = teamRef
	}
}

func WithClusterName(clusterName string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.ClusterName = clusterName
	}
}

func WithClusterSelector(selector metav1.LabelSelector) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.ClusterSelector = selector
	}
}

func WithNamespaces(namespaces ...string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.Namespaces = namespaces
	}
}

func WithCreateNamespace(createNamespaces bool) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.CreateNamespaces = createNamespaces
	}
}

func WithUsernames(usernames []string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.Usernames = usernames
	}
}

// NewTeamRoleBinding returns a greenhousev1alpha1.TeamRoleBinding object. Opts can be used to set the desired state of the TeamRoleBinding.
func NewTeamRoleBinding(name, namespace string, opts ...func(*greenhousev1alpha1.TeamRoleBinding)) *greenhousev1alpha1.TeamRoleBinding {
	trb := &greenhousev1alpha1.TeamRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRoleBinding",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, o := range opts {
		o(trb)
	}
	return trb
}

func WithMappedIDPGroup(group string) func(*greenhousev1alpha1.Team) {
	return func(t *greenhousev1alpha1.Team) {
		t.Spec.MappedIDPGroup = group
	}
}

// NewTeam returns a greenhousev1alpha1.Team object. Opts can be used to set the desired state of the Team.
func NewTeam(name, namespace string, opts ...func(*greenhousev1alpha1.Team)) *greenhousev1alpha1.Team {
	team := &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(team)
	}
	return team
}
