// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

var _ = Describe("resolveTeamRefs", func() {
	It("should return teamRefs when set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRefs: []string{"team-a", "team-b"},
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-a", "team-b"}))
	})

	It("should fall back to teamRef when teamRefs is empty", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "pre-migration-trb"},
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRef: "team-legacy",
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-legacy"}))
	})

	It("should prefer teamRefs over teamRef when both are set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRef:  "team-old",
				TeamRefs: []string{"team-new"},
			},
		}
		Expect(resolveTeamRefs(trb)).To(Equal([]string{"team-new"}))
	})

	It("should return nil when neither is set", func() {
		trb := &greenhousev1alpha2.TeamRoleBinding{
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{},
		}
		Expect(resolveTeamRefs(trb)).To(BeNil())
	})
})

var _ = Describe("EnsureCreated", func() {
	It("should not deploy RBAC to a cluster that is being deleted", func() {
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(greenhousev1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(greenhousev1alpha2.AddToScheme(scheme)).To(Succeed())
		cluster := &greenhousev1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cluster",
			Namespace:         "test-org",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"greenhouse.sap/test"},
		}}
		teamRole := &greenhousev1alpha1.TeamRole{ObjectMeta: metav1.ObjectMeta{Name: "test-role", Namespace: "test-org"}}
		team := &greenhousev1alpha1.Team{ObjectMeta: metav1.ObjectMeta{Name: "test-team", Namespace: "test-org"}}
		trb := &greenhousev1alpha2.TeamRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "test-rolebinding", Namespace: "test-org"},
			Spec: greenhousev1alpha2.TeamRoleBindingSpec{
				TeamRoleRef:     teamRole.Name,
				TeamRefs:        []string{team.Name},
				ClusterSelector: greenhousev1alpha2.ClusterSelector{Name: cluster.Name},
			},
		}
		r := &TeamRoleBindingReconciler{
			Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, teamRole, team).Build(),
			recorder: events.NewFakeRecorder(10),
		}

		_, result, err := r.EnsureCreated(context.Background(), trb)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(lifecycle.Success))
		Expect(trb.Status.PropagationStatus).To(BeEmpty())
	})
})
