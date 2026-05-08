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
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ExecuteSingleTeamRefScenario verifies that a TeamRoleBinding with a single entry in
// teamRefs creates a ClusterRoleBinding whose subjects include that team's IDP group.
func (s *scenario) ExecuteSingleTeamRefScenario(ctx context.Context) {
	GinkgoHelper()
	var trb *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() { s.cleanup(ctx, trb) })

	By("creating a TeamRoleBinding with a single team")
	trb = s.createTRB(ctx, "trb-single",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name),
		test.WithClusterName(s.clusterName),
	)

	By("verifying the ClusterRoleBinding is created on the remote cluster with the correct subject")
	remoteCRB := &rbacv1.ClusterRoleBinding{}
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)).
			To(Succeed(), "ClusterRoleBinding should exist on the remote cluster")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeTrue(), "subjects should contain teamAlpha's IDP group")
	}).Should(Succeed(), "ClusterRoleBinding should be created with the correct subject")

	By("verifying the TeamRoleBinding RBACReady status is True")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trb), trb)).To(Succeed())
		cond := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	}).Should(Succeed(), "RBACReady should be True")
}
