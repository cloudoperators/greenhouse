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

	Context("Validate Update Rolebinding", func() {
		It("Should deny changes to the empty Namespaces", func() {
			emptyNamespaces := []string{}
			oldRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  emptyNamespaces,
				},
			}

			editedNamespaces := []string{"demoNamespace"}
			curRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  editedNamespaces,
				},
			}

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot change existing TeamRoleBinding from cluster-scoped to namespace-scoped")))
		})

		It("Should deny removing all Namespaces", func() {
			filledNamespaces := []string{"demoNamespace1", "demoNamespace2"}
			oldRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  filledNamespaces,
				},
			}

			emptyNamespaces := []string{}
			curRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  emptyNamespaces,
				},
			}

			warns, err := ValidateUpdateRoleBinding(test.Ctx, test.K8sClient, oldRB, curRB)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cannot remove all namespaces in existing TeamRoleBinding")))
		})

		It("Should allow changing Namespaces", func() {
			filledNamespaces := []string{"demoNamespace1", "demoNamespace2"}
			oldRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  filledNamespaces,
				},
			}

			addedNamespaces := []string{"demoNamespace1", "demoNamespace2", "demoNamespace3"}
			curRB := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: teamRole.Name,
					TeamRef:     team.Name,
					ClusterName: cluster.Name,
					Namespaces:  addedNamespaces,
				},
			}

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
	})
})
