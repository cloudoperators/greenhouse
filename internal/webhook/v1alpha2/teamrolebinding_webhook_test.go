// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate Create RoleBinding", Ordered, func() {
	var (
		setup    *test.TestSetup
		teamRole *greenhousev1alpha1.TeamRole

		team    *greenhousev1alpha1.Team
		cluster *greenhousev1alpha1.Cluster
	)
	rules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		},
	}

	BeforeAll(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "rolebinding-create")
		team = setup.CreateTeam(test.Ctx, "test-setup-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		cluster = setup.CreateCluster(test.Ctx, "test-cluster", test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

		teamRole = setup.CreateTeamRole(test.Ctx, "test-teamrole", test.WithRules(rules))
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRole)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cluster)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
	})

	Context("deny create if referenced resources do not exist", func() {
		It("should return an error if the role does not exist", func() {
			rb := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef("non-existent-role"),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("role does not exist")))
		})
		It("should return an error if the team does not exist", func() {
			rb := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef("non-existent-team"),
				test.WithClusterName(cluster.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("team does not exist")))
		})
		It("should return an error if both clusterName and clusterSelector not specified", func() {
			rb := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("must specify either spec.clusterSelector.name or spec.clusterSelector.labelSelector")))
		})
		It("should return an error if both clusterName and clusterSelector are specified", func() {
			rb := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}}),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot specify both spec.clusterSelector.Name and spec.clusterSelector.labelSelector")))
		})
	})

	Context("Validate Update Rolebinding", func() {
		It("Should deny changes to the empty Namespaces", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces(),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change existing TeamRoleBinding from cluster-scoped to namespace-scoped")))
		})

		It("Should deny removing all Namespaces", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace1", "demoNamespace2"),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot remove all namespaces in existing TeamRoleBinding")))
		})

		It("Should allow changing Namespaces", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace1", "demoNamespace2"),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace1", "demoNamespace2", "demoNamespace3"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")

			removedNamespaces := []string{"demoNamespace1"}
			curRB.Spec.Namespaces = removedNamespaces

			warns, err = ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")

			differentNamespaces := []string{"differentNamespace1", "differentNamespace2"}
			curRB.Spec.Namespaces = differentNamespaces

			warns, err = ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")
		})

		It("Should deny changing the TeamRoleRef", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace"),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef("differentTeamRole"),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
				test.WithNamespaces("demoNamespace"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change TeamRoleRef of an existing TeamRoleBinding")))
		})

		It("Should deny changing the TeamRef", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef("differentTeam"),
				test.WithClusterName(cluster.Name),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change TeamRef of an existing TeamRoleBinding")))
		})

		It("Should return a warning when the owner Team is in another namespace", func() {
			oldRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", setup.Namespace(),
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
			)

			curRB := test.NewTeamRoleBinding(test.Ctx, "testBinding", "greenhouse",
				test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
				test.WithTeamRoleRef(teamRole.Name),
				test.WithTeamRef(team.Name),
				test.WithClusterName(cluster.Name),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(err).ToNot(HaveOccurred(), "expected no error")
			Expect(warns).Should(ContainElement(
				ContainSubstring("TeamRoleBinding should have a support-group Team set as its owner"),
			), "expected a warning")
		})
	})
})
