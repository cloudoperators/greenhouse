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
	const (
		testrolename = "test-role-create-admission"
	)
	BeforeAll(func() {
		testrole := &greenhousev1alpha1.TeamRole{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: test.TestNamespace,
				Name:      testrolename,
			},
			Spec: greenhousev1alpha1.TeamRoleSpec{
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get"},
						APIGroups: []string{""},
						Resources: []string{"pods"},
					},
				},
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, testrole)).To(Succeed())
	})

	Context("deny create if referenced resources do not exist", func() {
		It("should return an error if the role does not exist", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: test.TestNamespace,
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: "non-existent-role",
					TeamRef:     testteamname,
					ClusterName: testclustername,
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
					Namespace: test.TestNamespace,
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: testrolename,
					TeamRef:     "non-existent-team",
					ClusterName: testclustername,
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
					Namespace: test.TestNamespace,
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: testrolename,
					TeamRef:     testteamname,
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
					Namespace: test.TestNamespace,
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: testrolename,
					TeamRef:     testteamname,
					ClusterName: testclustername,
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
		It("should return an error if the cluster does not exist", func() {
			rb := &greenhousev1alpha1.TeamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: test.TestNamespace,
					Name:      "testBinding",
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef: testrolename,
					TeamRef:     testteamname,
					ClusterName: "non-existent-cluster",
				},
			}
			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
			Expect(err).To(MatchError(ContainSubstring("cluster does not exist")))
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
