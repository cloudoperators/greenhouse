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
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ExecuteTeamRefsMutationScenario verifies that updating teamRefs on an existing TeamRoleBinding
// causes the ClusterRoleBinding subjects on the remote cluster to be updated accordingly.
func (s *scenario) ExecuteTeamRefsMutationScenario(ctx context.Context) {
	GinkgoHelper()
	var trb *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() { s.cleanup(ctx, trb) })

	By("creating a TeamRoleBinding with a single team")
	trb = s.createTRB(ctx, "trb-mutate",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name),
		test.WithClusterName(s.clusterName),
	)

	remoteCRB := &rbacv1.ClusterRoleBinding{}
	By("verifying the initial ClusterRoleBinding has only teamAlpha subjects")
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)).To(Succeed())
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeTrue(), "initially teamAlpha's IDP group should be in subjects")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
		})).To(BeFalse(), "teamBeta's IDP group should not yet be in subjects")
	}).Should(Succeed(), "initial subjects should only contain teamAlpha")

	By("adding teamBeta to teamRefs")
	_, err := clientutil.CreateOrPatch(ctx, s.adminClient, trb, func() error {
		trb.Spec.TeamRefs = []string{s.teamAlpha.Name, s.teamBeta.Name}
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error adding teamBeta to teamRefs")

	By("verifying ClusterRoleBinding subjects now include both teams")
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)).To(Succeed())
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeTrue(), "teamAlpha's IDP group should remain in subjects")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
		})).To(BeTrue(), "teamBeta's IDP group should now appear in subjects")
	}).Should(Succeed(), "subjects should contain both teams after adding teamBeta")

	By("removing teamAlpha from teamRefs")
	_, err = clientutil.CreateOrPatch(ctx, s.adminClient, trb, func() error {
		trb.Spec.TeamRefs = []string{s.teamBeta.Name}
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error removing teamAlpha from teamRefs")

	By("verifying ClusterRoleBinding subjects now only contain teamBeta")
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)).To(Succeed())
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeFalse(), "teamAlpha's IDP group should have been removed from subjects")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
		})).To(BeTrue(), "teamBeta's IDP group should remain in subjects")
	}).Should(Succeed(), "subjects should only contain teamBeta after removing teamAlpha")

	By("verifying TeamRoleBinding status remains ready after mutation")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trb), trb)).To(Succeed())
		cond := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	}).Should(Succeed(), "RBACReady should stay True after mutation")
}
