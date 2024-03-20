// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Validate Create RoleBinding", func() {
	Context("deny create if the role does not exist", func() {
		It("should return an error if the role does not exist", func() {
			rb := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef: "nonExistentRole",
				},
			}

			warns, err := ValidateCreateRoleBinding(test.Ctx, test.K8sClient, rb)
			Expect(warns).To(BeNil(), "expected no warnings")
			Expect(err).To(HaveOccurred(), "expected an error")
		})
	})
})

var _ = Describe("Validate Update Rolebinding", func() {
	Context("ensures that changes to the immutable ClusterSelector are detected", func() {
		defaultClusterName := "test-cluster"
		emptyClusterName := ""
		editedClusterName := "edited-cluster"

		DescribeTable("Validate that adding, removing, or editing ClusterSelector is detected", func(oldCluster, curName string, expChange bool) {
			oldRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:     "testRole",
					Namespaces:  []string{"testNamespace"},
					ClusterName: oldCluster,
				},
			}

			curRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:     "testRole",
					Namespaces:  []string{"testNamespace"},
					ClusterName: curName,
				},
			}

			switch hasChanged := hasClusterChanged(oldRB, curRB); hasChanged {
			case true:
				Expect(expChange).To(Equal(hasChanged), "expected Cluster changes, but none found")
			default:
				Expect(expChange).To(Equal(hasChanged), "unexpected Cluster change detected")
			}
		},
			Entry("No Changes, all good", defaultClusterName, defaultClusterName, false),
			Entry("New selector added", emptyClusterName, defaultClusterName, true),
			Entry("ClusterSelector removed", defaultClusterName, emptyClusterName, true),
			Entry("Label Selector edited", defaultClusterName, editedClusterName, true),
		)
	})

	Context("ensures that changes to the immutable Namespaces are detected", func() {
		defaultNamespaces := []string{"testNamespace", "demoNamespace"}
		emptyNamespaces := []string{}
		editedNamespaces := []string{"editedNamespace", "demoNamespace"}
		deletedNamespaces := []string{"demoNamespace"}

		DescribeTable("Validate that adding, removing, or editing Namespaces is detected", func(oldNamespaces, curNamespaces []string, expChange bool) {
			oldRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:    "testRole",
					Namespaces: oldNamespaces,
				},
			}

			curRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:    "testRole",
					Namespaces: curNamespaces,
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
