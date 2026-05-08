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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// ExecuteClusterSelectorScenario verifies that a TeamRoleBinding with a label-based
// clusterSelector applies RBAC only to clusters whose labels match the selector, and that RBAC
// is removed from a cluster when it no longer matches (label change).
func (s *scenario) ExecuteClusterSelectorScenario(ctx context.Context) {
	GinkgoHelper()
	var trb *greenhousev1alpha2.TeamRoleBinding

	// A unique label key used only by this test run to avoid interference.
	labelKey := "trb-selector-test-" + rand.String(6)
	const labelMatch = "match"
	const labelNoMatch = "no-match"

	By("adding a unique label to the remote cluster for the selector test")
	cluster := &greenhousev1alpha1.Cluster{}
	Expect(s.adminClient.Get(ctx, client.ObjectKey{Name: s.clusterName, Namespace: s.namespace}, cluster)).
		To(Succeed(), "remote cluster should exist")

	_, err := clientutil.CreateOrPatch(ctx, s.adminClient, cluster, func() error {
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		cluster.Labels[labelKey] = labelMatch
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error labeling the cluster")

	// Restore the cluster label on cleanup regardless of test outcome.
	DeferCleanup(func() {
		_, err := clientutil.CreateOrPatch(ctx, s.adminClient, cluster, func() error {
			delete(cluster.Labels, labelKey)
			return nil
		})
		Expect(err).ToNot(HaveOccurred(), "there should be no error restoring the cluster label during cleanup")
		s.cleanup(ctx, trb)
	})

	By("creating a TeamRoleBinding with a label selector matching the remote cluster")
	trb = s.createTRB(ctx, "trb-selector",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name, s.teamBeta.Name),
		test.WithClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{labelKey: labelMatch},
		}),
	)

	By("verifying the ClusterRoleBinding is created on the matching remote cluster with subjects for both teams")
	remoteCRB := &rbacv1.ClusterRoleBinding{}
	Eventually(func(g Gomega) {
		g.Expect(s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)).
			To(Succeed(), "ClusterRoleBinding should be created on the matching cluster")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamAlpha.Spec.MappedIDPGroup
		})).To(BeTrue(), "subjects should contain teamAlpha's IDP group")
		g.Expect(slices.ContainsFunc(remoteCRB.Subjects, func(sub rbacv1.Subject) bool {
			return sub.Kind == rbacv1.GroupKind && sub.Name == s.teamBeta.Spec.MappedIDPGroup
		})).To(BeTrue(), "subjects should contain teamBeta's IDP group")
	}).Should(Succeed(), "ClusterRoleBinding should exist on the matching cluster with subjects for both teams")

	By("verifying the TeamRoleBinding PropagationStatus references only the matching cluster")
	Eventually(func(g Gomega) {
		g.Expect(s.adminClient.Get(ctx, client.ObjectKeyFromObject(trb), trb)).To(Succeed())
		g.Expect(trb.Status.PropagationStatus).To(HaveLen(1),
			"only one cluster should appear in PropagationStatus")
		g.Expect(trb.Status.PropagationStatus[0].ClusterName).To(Equal(s.clusterName))
		g.Expect(trb.Status.PropagationStatus[0].Condition.Status).To(Equal(metav1.ConditionTrue))
	}).Should(Succeed(), "PropagationStatus should only show the matching cluster")

	By("changing the cluster label so it no longer matches the selector")
	_, err = clientutil.CreateOrPatch(ctx, s.adminClient, cluster, func() error {
		cluster.Labels[labelKey] = labelNoMatch
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error updating cluster label")

	By("verifying the ClusterRoleBinding is removed after the cluster label no longer matches")
	Eventually(func(g Gomega) {
		err := s.remoteClient.Get(ctx, client.ObjectKey{Name: trb.GetRBACName()}, remoteCRB)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(),
			"ClusterRoleBinding should be removed after cluster label no longer matches the selector")
	}).Should(Succeed(), "ClusterRoleBinding should be cleaned up when cluster no longer matches selector")
}
