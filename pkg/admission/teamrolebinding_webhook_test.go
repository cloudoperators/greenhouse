// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
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

		teamRole = setup.CreateTeamRole(test.Ctx, "test-teamrole", test.WithRules(rules))
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRole)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, cluster)
	})

	Context("deny create if referenced resources do not exist", func() {
		It("should return an error if the role does not exist", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: setup.Namespace(),
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: "non-existent-role",
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
				},
			}

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("role does not exist")))
		})
		It("should return an error if the team does not exist", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: setup.Namespace(),
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     "non-existent-team",
					ClusterName: cluster.Name,
				},
			}
			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("team does not exist")))
		})
		It("should return an error if both clusterName and clusterSelector not specified", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: setup.Namespace(),
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
				},
			}
			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("must specify either spec.clusterName or spec.clusterSelector")))
		})
		It("should return an error if both clusterName and clusterSelector are specified", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: setup.Namespace(),
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{"test": "test"},
					},
				},
			}
			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot specify both spec.clusterName and spec.clusterSelector")))
		})
	})
})

var _ = Describe("Validate Update Rolebinding", func() {
	Context("ensures that changes to the immutable Namespaces are detected", func() {
		defaultNamespaces := []string{"testNamespace", "demoNamespace"}
		emptyNamespaces := []string{}
		editedNamespaces := []string{"editedNamespace", "demoNamespace"}
		deletedNamespaces := []string{"demoNamespace"}

		DescribeTable("Validate that adding, removing, or editing Namespaces is detected", func(oldNamespaces, curNamespaces []string, expChange bool) {
			oldRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: "testRole",
					Namespaces:  oldNamespaces,
				},
			}

			curRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: "testRole",
					Namespaces:  curNamespaces,
				},
			}

			switch hasChanged := hasNamespacesChanged(oldRB, curRB); hasChanged {
			case true:
				Expect(expChange).To(BeTrue(), "expected Namespaces changes, but none found")
			default:
				Expect(expChange).To(BeFalse(), "unexpected Namespaces change detected")
			}
		},
			Entry("No Changes, all good", defaultNamespaces, defaultNamespaces, false),
			Entry("Namespaces added", emptyNamespaces, defaultNamespaces, true),
			Entry("Namespaces removed", defaultNamespaces, emptyNamespaces, true),
			Entry("Namespaces edited", defaultNamespaces, editedNamespaces, true),
			Entry("Namespaces deleted", defaultNamespaces, deletedNamespaces, true),
		)
	})
})
