// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ExecutePartialFailureScenario covers two sub-cases:
//  1. One valid team + one missing team → RBAC is applied for the valid team and the status
//     message mentions the missing team.
//  2. All referenced teams missing → RBACReady=False with TeamNotFound reason.
func (s *scenario) ExecutePartialFailureScenario(ctx context.Context) {
	GinkgoHelper()

	// Unique suffix per run to avoid collisions with leftover resources.
	suffix := rand.String(6)

	// ── Sub-case 1: one valid, one non-existent ──────────────────────────────
	By("Sub-case 1: one valid team and one non-existent team")
	var trbPartial *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() { s.cleanup(ctx, trbPartial) })

	trbPartial = s.createTRB(ctx, "trb-partial-"+suffix,
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name, "non-existent-team"),
		test.WithClusterName(s.clusterName),
	)

	By("verifying the ClusterRoleBinding is still created for the valid team with the correct subject")
	remoteCRB := &rbacv1.ClusterRoleBinding{}
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trbPartial.GetRBACName()}, remoteCRB)).
			To(Succeed(), "ClusterRoleBinding should be created despite the missing team")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeTrue(), "subjects should contain teamAlpha's IDP group")
	}).Should(Succeed(), "ClusterRoleBinding should exist with the valid team's subject")

	By("verifying the status reports the partial failure for the missing team")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trbPartial), trbPartial)).To(Succeed())
		cond := trbPartial.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		g.Expect(cond.Message).To(ContainSubstring("non-existent-team"),
			"status message should mention the missing team")
	}).Should(Succeed(), "status should reflect partial success with missing team information")

	// ── Sub-case 2: all teams missing ────────────────────────────────────────
	By("Sub-case 2: all referenced teams are missing")
	var trbAllMissing *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() { s.cleanup(ctx, trbAllMissing) })

	trbAllMissing = s.createTRB(ctx, "trb-all-missing-"+suffix,
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs("ghost-team-1", "ghost-team-2"),
		test.WithClusterName(s.clusterName),
	)

	By("verifying RBACReady=False with TeamNotFound reason")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trbAllMissing), trbAllMissing)).To(Succeed())
		cond := trbAllMissing.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		g.Expect(cond.Reason).To(Equal(greenhousev1alpha2.TeamNotFound))
	}).Should(Succeed(), "RBACReady should be False when all referenced teams are missing")

	By("verifying no ClusterRoleBinding is created on the remote cluster")
	missingCRB := &rbacv1.ClusterRoleBinding{}
	Consistently(func() bool {
		return apierrors.IsNotFound(
			s.remoteClient.Get(ctx, client.ObjectKey{Name: trbAllMissing.GetRBACName()}, missingCRB),
		)
	}, "5s", "200ms").Should(BeTrue(), "ClusterRoleBinding should not be created when all teams are missing")
}
