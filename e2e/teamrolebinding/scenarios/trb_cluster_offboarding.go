// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func (s *scenario) ExecuteClusterOffboardingScenario(ctx context.Context) {
	GinkgoHelper()
	namespace := "trb-e2e-ns-offboarding-" + rand.String(6)

	var clusterTRB, namespaceTRB *greenhousev1alpha2.TeamRoleBinding
	DeferCleanup(func() {
		s.cleanup(ctx, clusterTRB)
		s.cleanup(ctx, namespaceTRB)
		test.EventuallyDeleted(ctx, s.remoteClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
	})

	By("creating a cluster scoped and a namespace scoped TeamRoleBinding")
	clusterTRB = s.createTRB(ctx, "trb-offboarding-cluster",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name),
		test.WithClusterName(s.clusterName),
	)
	namespaceTRB = s.createTRB(ctx, "trb-offboarding-namespace",
		test.WithTeamRoleRef(s.teamRole.Name),
		test.WithTeamRefs(s.teamAlpha.Name),
		test.WithClusterName(s.clusterName),
		test.WithNamespaces(namespace),
		test.WithCreateNamespace(true),
	)
	teamRBAC := []client.Object{
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: s.teamRole.GetRBACName()}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterTRB.GetRBACName()}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: namespaceTRB.GetRBACName(), Namespace: namespace}},
	}

	By("verifying the team RBAC is created on the remote cluster")
	Eventually(func(g Gomega) {
		for _, obj := range teamRBAC {
			g.Expect(s.remoteClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).
				To(Succeed(), "%T %s should exist on the remote cluster", obj, obj.GetName())
		}
	}).Should(Succeed(), "team RBAC should be created on the remote cluster")

	By("off-boarding the remote cluster")
	shared.OffBoardRemoteCluster(ctx, s.adminClient, s.remoteClient, time.Now().UTC(), s.clusterName, s.namespace)

	By("verifying the team RBAC is removed from the remote cluster")
	for _, obj := range teamRBAC {
		err := s.remoteClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "%T %s should be removed from the remote cluster", obj, obj.GetName())
	}
}
