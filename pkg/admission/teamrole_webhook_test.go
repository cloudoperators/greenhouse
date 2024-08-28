// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate Role Admission", func() {
	var (
		setup           *test.TestSetup
		teamRole        *greenhousev1alpha1.TeamRole
		teamRoleBinding *greenhousev1alpha1.TeamRoleBinding

		team    *greenhousev1alpha1.Team
		cluster *greenhousev1alpha1.Cluster
	)
	rules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		},
	}
	aggregationRule := &rbacv1.AggregationRule{
		ClusterRoleSelectors: []metav1.LabelSelector{
			{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "role-admission")
		cluster = setup.CreateCluster(test.Ctx, "test-cluster")
		team = setup.CreateTeam(test.Ctx, "test-team")
	})

	AfterEach(func() {
		if teamRoleBinding != nil {
			test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRoleBinding)
			teamRoleBinding = nil
		}
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRole)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cluster)
	})

	It("should not allow creating a TeamRole with both Rules and AggregationRule set", func() {
		teamRole = setup.NewTeamRole(test.Ctx, "test-role", test.WithRules(rules), test.WithAggregationRule(aggregationRule), test.WithRules(rules))

		err := test.K8sClient.Create(test.Ctx, teamRole)
		Expect(err).To(HaveOccurred(), "there should be an error creating the role with both rules and aggregation rule set")
		Expect(err.Error()).To(ContainSubstring(errAggregationRuleAndRulesExclusive), "unexpected error message")
	})

	It("should not allow to add Rules to a TeamRole with AggregationRule set", func() {
		teamRole = setup.CreateTeamRole(test.Ctx, "test-role", test.WithAggregationRule(aggregationRule), test.WithRules(nil))

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, teamRole, func() error {
			teamRole.Spec.Rules = rules
			return nil
		})

		Expect(err).To(HaveOccurred(), "there should be an error adding Rules if the TeamRole has an AggregationRule set")
	})

	It("should not allow to add an AggregationRule to a TeamRole with Rules set", func() {
		teamRole = setup.CreateTeamRole(test.Ctx, "test-role", test.WithRules(rules))

		_, err := clientutil.CreateOrPatch(test.Ctx, test.K8sClient, teamRole, func() error {
			teamRole.Spec.AggregationRule = aggregationRule
			return nil
		})

		Expect(err).To(HaveOccurred(), "there should be an error adding an AggregationRule if the TeamRole has Rules set")
	})

	It("should not allow deleting a role with references", func() {
		teamRole = setup.CreateTeamRole(test.Ctx, "test-delete-role", test.WithRules(rules))

		teamRoleBinding = setup.CreateTeamRoleBinding(test.Ctx, "test-delete-rolebinding",
			test.WithClusterName(cluster.Name),
			test.WithTeamRef(team.Name),
			test.WithTeamRoleRef(teamRole.Name),
		)

		err := test.K8sClient.Delete(test.Ctx, teamRole)
		Expect(err).To(HaveOccurred(), "there should be an error deleting the role with references")
	})
})

// setupRoleBindingWebhookForTest adds an indexField for '.spec.roleRef', additionally to setting up the webhook for the RoleBinding resource. It is used in the webhook tests.
// we can't add this to the webhook setup because it's already indexed by the controller and indexing the field twice is not possible.
// This is to have the webhook tests run independently of the controller.
func setupRoleBindingWebhookForTest(mgr manager.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingRoleRefField, func(rawObj client.Object) []string {
		// Extract the Role name from the RoleBinding Spec, if one is provided
		roleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if roleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{roleBinding.Spec.TeamRoleRef}
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error indexing the rolebindings by roleRef")
	return SetupTeamRoleBindingWebhookWithManager(mgr)
}
