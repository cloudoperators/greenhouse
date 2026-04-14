// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ExecuteNamespaceCreationScenario verifies that, when createNamespaces=true is set on a
// TeamRoleBinding that references multiple teams, namespaces are created on the remote cluster
// and each namespace receives a RoleBinding whose subjects include all teams' IDP groups.
func (s *scenario) ExecuteNamespaceCreationScenario(ctx context.Context) {
	GinkgoHelper()

	const (
		nsOne = "trb-e2e-ns-one"
		nsTwo = "trb-e2e-ns-two"
	)

	var trb *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() { s.cleanup(ctx, trb) })

	By("creating a TeamRoleBinding with createNamespaces=true referencing two teams")
	trb = s.createTRB(ctx, "trb-ns-create",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name, s.teamBeta.Name),
		test.WithClusterName(s.clusterName),
		test.WithNamespaces(nsOne, nsTwo),
		test.WithCreateNamespace(true),
	)

	By("verifying RoleBinding in namespace one is created on the remote cluster")
	rb1 := &rbacv1.RoleBinding{}
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx,
			types.NamespacedName{Name: trb.GetRBACName(), Namespace: nsOne},
			rb1)).To(Succeed(), "RoleBinding in nsOne should exist on remote cluster")
	}).Should(Succeed(), "RoleBinding should be created in nsOne")

	By("verifying RoleBinding in namespace two is created on the remote cluster")
	rb2 := &rbacv1.RoleBinding{}
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx,
			types.NamespacedName{Name: trb.GetRBACName(), Namespace: nsTwo},
			rb2)).To(Succeed(), "RoleBinding in nsTwo should exist on remote cluster")
	}).Should(Succeed(), "RoleBinding should be created in nsTwo")

	By("verifying RoleBinding subjects in nsOne include both teams' IDP groups")
	Expect(slices.ContainsFunc(rb1.Subjects, func(sub rbacv1.Subject) bool {
		return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
	})).To(BeTrue(), "RoleBinding in nsOne should contain teamAlpha's IDP group")
	Expect(slices.ContainsFunc(rb1.Subjects, func(sub rbacv1.Subject) bool {
		return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
	})).To(BeTrue(), "RoleBinding in nsOne should contain teamBeta's IDP group")

	By("verifying RoleBinding subjects in nsTwo include both teams' IDP groups")
	Expect(slices.ContainsFunc(rb2.Subjects, func(sub rbacv1.Subject) bool {
		return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
	})).To(BeTrue(), "RoleBinding in nsTwo should contain teamAlpha's IDP group")
	Expect(slices.ContainsFunc(rb2.Subjects, func(sub rbacv1.Subject) bool {
		return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
	})).To(BeTrue(), "RoleBinding in nsTwo should contain teamBeta's IDP group")

	By("verifying the TeamRoleBinding RBACReady status is True")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trb), trb)).To(Succeed())
		cond := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	}).Should(Succeed(), "RBACReady should be True after namespace creation with multiple teams")
}
