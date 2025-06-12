// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/test/mocks"
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
		team = setup.CreateTeam(test.Ctx, "test-team")
		cluster = setup.CreateCluster(test.Ctx, "test-cluster")

		teamRole = setup.CreateTeamRole(test.Ctx, "test-teamrole", mocks.WithRules(rules))
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRole)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cluster)
	})

	Context("deny create if referenced resources do not exist", func() {
		It("should return an error if the role does not exist", func() {
			rb := mocks.NewTeamRoleBinding("testBinding", setup.Namespace(),
				mocks.WithTeamRoleRef("non-existent-role"),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("role does not exist")))
		})
		It("should return an error if the team does not exist", func() {
			rb := mocks.NewTeamRoleBinding("testBinding", setup.Namespace(),
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef("non-existent-team"),
				mocks.WithClusterName(cluster.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("team does not exist")))
		})
		It("should return an error if both clusterName and clusterSelector not specified", func() {
			rb := mocks.NewTeamRoleBinding("testBinding", setup.Namespace(),
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("must specify either spec.clusterName or spec.clusterSelector")))
		})
		It("should return an error if both clusterName and clusterSelector are specified", func() {
			rb := mocks.NewTeamRoleBinding("testBinding", setup.Namespace(),
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}}),
			)

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot specify both spec.clusterName and spec.clusterSelector")))
		})
	})

	Context("Validate Update Rolebinding", func() {
		It("Should deny changes to the empty Namespaces", func() {
			oldRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces(),
			)

			curRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change existing TeamRoleBinding from cluster-scoped to namespace-scoped")))
		})

		It("Should deny removing all Namespaces", func() {
			oldRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace1", "demoNamespace2"),
			)

			curRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot remove all namespaces in existing TeamRoleBinding")))
		})

		It("Should allow changing Namespaces", func() {
			oldRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace1", "demoNamespace2"),
			)

			curRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace1", "demoNamespace2", "demoNamespace3"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")

			curRB.Spec.Namespaces = []string{"demoNamespace1"}
			warns, err = ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")

			curRB.Spec.Namespaces = []string{"differentNamespace1", "differentNamespace2"}
			warns, err = ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).ToNot(HaveOccurred(), "expected no error")
		})

		It("Should deny changing the TeamRoleRef", func() {
			oldRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace"),
			)

			curRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef("differentTeamRole"),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
				mocks.WithNamespaces("demoNamespace"),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change TeamRoleRef of an existing TeamRoleBinding")))
		})

		It("Should deny changing the TeamRef", func() {
			oldRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef(team.Name),
				mocks.WithClusterName(cluster.Name),
			)

			curRB := mocks.NewTeamRoleBinding("testBinding", "greenhouse",
				mocks.WithTeamRoleRef(teamRole.Name),
				mocks.WithTeamRef("differentTeam"),
				mocks.WithClusterName(cluster.Name),
			)

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change TeamRef of an existing TeamRoleBinding")))
		})
	})
})
