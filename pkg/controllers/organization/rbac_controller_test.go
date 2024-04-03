// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/rbac"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	orgName = "test-org"
)

var ownerRef metav1.OwnerReference

var _ = Describe("Test RBAC reconciliation", func() {
	When("reconciling an organization", func() {
		It("should create the Organization successfully", func() {
			testOrg := &greenhousev1alpha1.Organization{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Organization",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      orgName,
					Namespace: test.TestNamespace,
				},
			}
			Expect(test.K8sClient.Create(test.Ctx, testOrg)).Should(Succeed(), "must be able to create test organization")
			orgID := types.NamespacedName{Name: orgName, Namespace: ""}
			actOrg := &greenhousev1alpha1.Organization{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, orgID, actOrg) == nil
			}).Should(BeTrue(), "test organization must be created")

			ownerRef = metav1.OwnerReference{
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
				Kind:       "Organization",
				UID:        testOrg.UID,
				Name:       testOrg.Name,
			}
		})

		It("must create a ClusterRole for the Org Admin", func() {
			clusterRoleID := types.NamespacedName{Name: rbac.OrganizationAdminRoleName(orgName), Namespace: ""}
			actClusterRole := &rbacv1.ClusterRole{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, clusterRoleID, actClusterRole) == nil
			}).Should(BeTrue(), "ClusterRole for the Admin must be created")
			Expect(actClusterRole.Rules).To(ContainElements(rbac.OrganizationAdminClusterRolePolicyRules(orgName)), "ClusterRole for the Admin must have the correct rules")
			Expect(actClusterRole.OwnerReferences).To(ContainElement(ownerRef), "ClusterRole for the Admin must have the correct owner reference")
		})

		It("must create a ClusterRoleBinding for Org Admin ", func() {
			clusterRoleBindingID := types.NamespacedName{Name: rbac.OrganizationAdminRoleName(orgName), Namespace: ""}
			actClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, clusterRoleBindingID, actClusterRoleBinding) == nil
			}).Should(BeTrue(), "ClusterRoleBinding for the Admin must be created")
			Expect(actClusterRoleBinding.Subjects).To(ContainElement(rbacv1.Subject{APIGroup: rbacv1.GroupName, Kind: rbacv1.GroupKind, Name: rbac.OrganizationAdminRoleName(orgName)}), "ClusterRoleBinding for org admin must have the correct subject")
			Expect(actClusterRoleBinding.OwnerReferences).To(ContainElement(ownerRef), "ClusterRoleBinding for the Admin must have the correct owner reference")
		})

		It("must create a Role for Org Admins", func() {
			roleID := types.NamespacedName{Name: rbac.OrganizationAdminRoleName(orgName), Namespace: test.TestNamespace}
			actRole := &rbacv1.Role{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleID, actRole) == nil
			}).Should(BeTrue(), "Role for the org admins must be created")
			Expect(actRole.Rules).To(ContainElements(rbac.OrganizationAdminPolicyRules()), "Role for org admins must have the correct rules")
			Expect(actRole.OwnerReferences).To(ContainElement(ownerRef), "Role for org admins must have the correct owner reference")
		})

		It("must create a RoleBinding for org admins", func() {
			roleBindingID := types.NamespacedName{Name: rbac.OrganizationAdminRoleName(orgName), Namespace: test.TestNamespace}
			actRoleBinding := &rbacv1.RoleBinding{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleBindingID, actRoleBinding) == nil
			}).Should(BeTrue(), "RoleBinding for org admins must be created")
			Expect(actRoleBinding.RoleRef.Name).To(Equal(rbac.OrganizationAdminRoleName(orgName)), "RoleBinding for org admins must have the correct role reference")
			Expect(actRoleBinding.Subjects).To(ContainElement(rbacv1.Subject{APIGroup: rbacv1.GroupName, Kind: rbacv1.GroupKind, Name: rbac.OrganizationAdminRoleName(orgName)}), "RoleBinding for org admins must have the correct subject")
			Expect(actRoleBinding.OwnerReferences).To(ContainElement(ownerRef), "RoleBinding for org admins must have the correct owner reference")
		})

		It("must create a ClusterRole for Org Member", func() {
			clusterRoleID := types.NamespacedName{Name: rbac.OrganizationRoleName(orgName), Namespace: ""}
			actClusterRole := &rbacv1.ClusterRole{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, clusterRoleID, actClusterRole) == nil
			}).Should(BeTrue(), "ClusterRole for the Admin must be created")
			Expect(actClusterRole.Rules).To(ContainElements(rbac.OrganizationMemberClusterRolePolicyRules(orgName)), "ClusterRole for Member must have the correct rules")
		})

		It("must create a ClusterRoleBinding for Org Member ", func() {
			clusterRoleBindingID := types.NamespacedName{Name: rbac.OrganizationRoleName(orgName), Namespace: ""}
			actClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, clusterRoleBindingID, actClusterRoleBinding) == nil
			}).Should(BeTrue(), "ClusterRoleBinding for org member must be created")
			Expect(actClusterRoleBinding.Subjects).To(ContainElement(rbacv1.Subject{APIGroup: rbacv1.GroupName, Kind: rbacv1.GroupKind, Name: rbac.OrganizationRoleName(orgName)}), "ClusterRoleBinding for org member must have the correct subject")
			Expect(actClusterRoleBinding.OwnerReferences).To(ContainElement(ownerRef), "ClusterRoleBinding for org member must have the correct owner reference")
		})

		It("must create a Role for Org Cluster Admins", func() {
			roleID := types.NamespacedName{Name: rbac.OrganizationClusterAdminRoleName(orgName), Namespace: test.TestNamespace}
			actRole := &rbacv1.Role{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleID, actRole) == nil
			}).Should(BeTrue(), "Role for org cluster admin must be created")
			Expect(actRole.Rules).To(ContainElements(rbac.OrganizationClusterAdminPolicyRules()), "Role for org cluster admin must have the correct rules")
			Expect(actRole.OwnerReferences).To(ContainElement(ownerRef), "Role for org cluster admin must have the correct owner reference")
		})

		It("must create a Role for Org Plugin Admins", func() {
			roleID := types.NamespacedName{Name: rbac.OrganizationPluginAdminRoleName(orgName), Namespace: test.TestNamespace}
			actRole := &rbacv1.Role{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleID, actRole) == nil
			}).Should(BeTrue(), "Role for org plugin admin must be created")
			Expect(actRole.Rules).To(ContainElements(rbac.OrganizationPluginAdminPolicyRules()), "Role for org plugin admin must have the correct rules")
			Expect(actRole.OwnerReferences).To(ContainElement(ownerRef), "Role for org plugin admin must have the correct owner reference")
		})

		It("must create a Role for Org Member", func() {
			roleID := types.NamespacedName{Name: rbac.OrganizationRoleName(orgName), Namespace: test.TestNamespace}
			actRole := &rbacv1.Role{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleID, actRole) == nil
			}).Should(BeTrue(), "Role for org member must be created")
			Expect(actRole.Rules).To(ContainElements(rbac.OrganizationMemberPolicyRules()), "Role for org member must have the correct rules")
			Expect(actRole.OwnerReferences).To(ContainElement(ownerRef), "Role for org member must have the correct owner reference")
		})

		It("must create a RoleBinding for org member", func() {
			roleBindingID := types.NamespacedName{Name: rbac.OrganizationRoleName(orgName), Namespace: test.TestNamespace}
			actRoleBinding := &rbacv1.RoleBinding{}
			Eventually(func() bool {
				return test.K8sClient.Get(test.Ctx, roleBindingID, actRoleBinding) == nil
			}).Should(BeTrue(), "RoleBinding for org member must be created")
			Expect(actRoleBinding.RoleRef.Name).To(Equal(rbac.OrganizationRoleName(orgName)), "RoleBinding for org member must have the correct role reference")
			Expect(actRoleBinding.Subjects).To(ContainElement(rbacv1.Subject{APIGroup: rbacv1.GroupName, Kind: rbacv1.GroupKind, Name: rbac.OrganizationRoleName(orgName)}), "RoleBinding for org member must have the correct subject")
			Expect(actRoleBinding.OwnerReferences).To(ContainElement(ownerRef), "RoleBinding for org member must have the correct owner reference")
		})
	})
})
