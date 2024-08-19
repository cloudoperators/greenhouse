// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	setup      *test.TestSetup
	clusterA   *greenhousev1alpha1.Cluster
	clusterB   *greenhousev1alpha1.Cluster
	teamUT     *greenhousev1alpha1.Team
	teamRoleUT *greenhousev1alpha1.TeamRole
)

var _ = Describe("Validate ClusterRole & RoleBinding on Remote Cluster", Ordered, func() {
	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
		clusterA = setup.OnboardCluster(test.Ctx, "test-cluster-a", clusterAKubeConfig, test.WithLabel("cluster", "a"), test.WithLabel("rbac", "true"))
		clusterB = setup.OnboardCluster(test.Ctx, "test-cluster-b", clusterBKubeConfig, test.WithLabel("cluster", "b"), test.WithLabel("rbac", "true"))

		teamUT = setup.CreateTeam(test.Ctx, "test-team", test.WithMappedIDPGroup(testTeamIDPGroup))

		By("creating a TeamRole on the central cluster")
		teamRoleUT = setup.CreateTeamRole(test.Ctx, "test-role", test.WithLabels(map[string]string{"aggregate": "true"}))
	})

	AfterEach(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamRoleUT)
	})

	// After all tests are run ensure there are no resources left behind on the remote cluster
	// This ensures the deletion of the Remote Resources is working correctly.
	AfterAll(func() {
		// check that all ClusterRoleBindings are eventually deleted on the remote cluster
		remoteCRBList := &rbacv1.ClusterRoleBindingList{}
		listOpts := []client.ListOption{
			client.HasLabels{greenhouseapis.LabelKeyRoleBinding},
		}
		Eventually(func() bool {
			err := clusterAKubeClient.List(test.Ctx, remoteCRBList, listOpts...)
			if err != nil || len(remoteCRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoleBindings left to list on the remote cluster")

		// check that all RoleBindings are eventually deleted on the remote cluster
		remoteRBList := &rbacv1.RoleBindingList{}
		Eventually(func() bool {
			err := clusterAKubeClient.List(test.Ctx, remoteRBList, listOpts...)
			if err != nil || len(remoteRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no RoleBindings left to list on the remote cluster")

		// check that all ClusterRoles are eventually deleted on the remote cluster
		remoteList := &rbacv1.ClusterRoleList{}
		listOpts = []client.ListOption{
			client.HasLabels{greenhouseapis.LabelKeyRole},
		}
		Eventually(func() bool {
			err := clusterAKubeClient.List(test.Ctx, remoteList, listOpts...)
			if err != nil || len(remoteList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoles left to list on the remote cluster")
	})

	Context("When editing clusterName or clusterSelctor on a TeamRoleBinding", func() {
		It("should remove the RoleBinding on the cluster that is no longer referenced by clusterName and reconcile the clusters referenced by clusterSelector", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name))
			trbKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}
			By("validating the RoleBinding created on the remote clusterA")
			remoteRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name: trb.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterA.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("updating the TeamRoleBinding with to a selector matching clusterB")
			trb.Spec.ClusterSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, trb, func() error {
				trb.Spec.ClusterName = ""
				trb.Spec.ClusterSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
				return nil
			})
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the TeamRoleBinding")
			By("validating the RoleBinding is removed from the remote clusterA and created on the remote clusterB")
			Eventually(func() bool {
				err := clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "the RoleBinding should be removed from the remote clusterA")
			remoteRoleBinding = &rbacv1.ClusterRoleBinding{}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterBKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote clusterB")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding from ClusterB")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).To(ContainElement(And(HaveField("ClusterName", clusterB.Name), HaveField("Condition.Status", metav1.ConditionTrue))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})

		It("should remove the RoleBinding on the cluster that is no longer referenced by the clusterSelector", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"rbac": "true"}}))
			trbKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}
			By("validating the RoleBinding created on the remote clusterA")
			remoteRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name: trb.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the RoleBinding created on the remote clusterB")
			Eventually(func(g Gomega) bool {
				g.Expect(clusterBKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(2), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterA.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterB.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("updating the TeamRoleBinding with to a selector matching clusterB")
			trb.Spec.ClusterSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, trb, func() error {
				trb.Spec.ClusterName = ""
				trb.Spec.ClusterSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
				return nil
			})
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the TeamRoleBinding")
			By("validating the RoleBinding is removed from the remote clusterA and created on the remote clusterB")
			Eventually(func() bool {
				err := clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "the RoleBinding should be removed from the remote clusterA")
			remoteRoleBinding = &rbacv1.ClusterRoleBinding{}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterBKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote clusterB")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding from ClusterB")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).To(ContainElement(And(HaveField("ClusterName", clusterB.Name), HaveField("Condition.Status", metav1.ConditionTrue))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})

		It("should remove the RoleBinding on the cluster when the label matching the clusterSelector is removed from the cluster", func() {
			By("labelling the clusters")
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, clusterA, func() error {
				clusterA.Labels["foo"] = "bar"
				return nil
			})
			Expect(err).NotTo(HaveOccurred(), "there should be no error labelling clusterA")

			_, err = clientutil.CreateOrPatch(test.Ctx, k8sClient, clusterB, func() error {
				clusterB.Labels["foo"] = "bar"
				return nil
			})
			Expect(err).NotTo(HaveOccurred(), "there should be no error labelling clusterB")

			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}))
			trbKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}
			By("validating the RoleBinding created on the remote clusterA")
			remoteRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name: trb.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the RoleBinding created on the remote clusterB")
			Eventually(func(g Gomega) bool {
				g.Expect(clusterBKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(2), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterA.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterB.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("updating the ClusterB to remove the label that matches the clusterSelector")
			_, err = clientutil.CreateOrPatch(test.Ctx, k8sClient, clusterB, func() error {
				delete(clusterB.Labels, "foo")
				return nil
			})
			Expect(err).ToNot(HaveOccurred(), "there should be no error removing the label from clusterB")

			By("validating the RoleBinding is removed from the remote clusterB")
			Eventually(func() bool {
				err := clusterBKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "the RoleBinding should be removed from the remote clusterB")
			remoteRoleBinding = &rbacv1.ClusterRoleBinding{}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote clusterA")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding from ClusterB")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(trb.Status.PropagationStatus).To(ContainElement(And(HaveField("ClusterName", clusterA.Name), HaveField("Condition.Status", metav1.ConditionTrue))))
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding with namespaces on the central cluster", func() {
		It("Should create a ClusterRole and RoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding without namespaces on the central cluster", func() {
		It("Should create a ClusterRole and ClusterRoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      greenhouseapis.RBACPrefix + trb.Name,
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating Greenhouse TeamRoleBindings with and without namespaces on the central cluster", func() {
		It("Should create a ClusterRole, ClusterRoleBinding and TeamRoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      greenhouseapis.RBACPrefix + trb.Name,
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("creating a TeamRoleBinding on the central cluster")
			trbNoNamespaces := setup.CreateTeamRoleBinding(test.Ctx, "teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trbNoNamespaces.GetRBACName(),
				Namespace: trbNoNamespaces.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trbNoNamespaces)
		})
	})

	Context("When updating Greenhouse TeamRole & TeamRoleBinding w/wo Namespaces on the central cluster", func() {
		It("Should reconcile the ClusterRole, ClusterRoleBinding for a TeamRoleBinding without Namespaces on the remote cluster", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("updating the Greenhouse TeamRole on the central cluster")
			teamRoleUT.Spec.Rules[0].Verbs = []string{"get"}
			Expect(k8sClient.Update(test.Ctx, teamRoleUT)).To(Succeed(), "there should be no error updating the TeamRole on the central cluster")
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the remote cluster")
				return g.Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))
			}).Should(BeTrue(), "there should be no error getting the ClusterRole from the remote cluster")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
		It("Should reconcile the ClusterRole, RoleBindings for a TeamRoleBinding with Namespaces on the remote cluster", func() {
			By("creating a TeamRoleBinding with Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("validating the RoleBindings are created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the TeamRoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("updating the Greenhouse TeamRole on the central cluster")
			teamRoleUT.Spec.Rules[0].Verbs = []string{"get"}
			err := k8sClient.Update(test.Ctx, teamRoleUT)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the remote cluster")
				return g.Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))
			}).Should(BeTrue(), "there should be no error getting the ClusterRole from the remote cluster")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding with AggregationRule", func() {
		It("Should create a ClusterRole with Aggregation Rules and RoleBinding on the remote cluster", func() {
			trbBase := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRef(teamUT.Name),
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithClusterName(clusterA.Name),
			)
			By("creating a TeamRoleBinding with AggregationRule on the central cluster")
			trAggregate := setup.CreateTeamRole(test.Ctx, "test-aggregation-teamrole",
				test.WithRules(nil),
				test.WithAggregationRule(&rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{MatchLabels: map[string]string{"aggregate": "true"}}},
				}))
			trbAggregate := setup.CreateTeamRoleBinding(test.Ctx, "test-aggregation-teamrolebinding",
				test.WithTeamRef(teamUT.Name),
				test.WithTeamRoleRef(trAggregate.Name),
				test.WithClusterName(clusterA.Name),
			)
			By("validating the Base ClusterRole created on the remote cluster")
			baseClusterRole := &rbacv1.ClusterRole{}
			baseClusterRoleName := types.NamespacedName{Name: teamRoleUT.GetRBACName()}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, baseClusterRoleName, baseClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				g.Expect(baseClusterRole.Labels).To(HaveKeyWithValue("aggregate", "true"), "the Base ClusterRole should have the aggregate label")
				return true
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")

			By("validating the Aggregate ClusterRole created on the remote cluster")
			aggregateClusterRole := &rbacv1.ClusterRole{}
			aggregateClusterRoleName := types.NamespacedName{Name: trAggregate.GetRBACName()}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, aggregateClusterRoleName, aggregateClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				g.Expect(aggregateClusterRole.AggregationRule).To(Equal(trAggregate.Spec.AggregationRule), "the Aggregate ClusterRole should have the same AggregationRule as the Base ClusterRole")
				return true
			}).Should(BeTrue(), "the ClusterRole should exists and have the correct rules")
			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trbBase)
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trbAggregate)
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trAggregate)
		})
	})

	Context("When tampering with a RoleBinding on the Remote Cluster", func() {
		It("should reconcile the remote RoleBinding", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
				return !remoteClusterRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRoleBinding")
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return g.Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules)) &&
					!remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("altering the RoleBinding on the remote cluster")
			Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the remote cluster")
			expected := remoteClusterRoleBinding.DeepCopy().Subjects
			remoteClusterRoleBinding.Subjects = append(remoteClusterRoleBinding.Subjects, rbacv1.Subject{Kind: "User", Name: "foobar", APIGroup: "rbac.authorization.k8s.io"})
			Expect(clusterAKubeClient.Update(test.Ctx, remoteClusterRoleBinding)).To(Succeed(), "there should be no error updating the ClusterRoleBinding on the remote cluster")

			By("triggering the reconcile of the central cluster TeamRoleBinding with a noop update")
			Expect(k8sClient.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding from the central cluster")
			// changing the labels to trigger the reconciliation in this test.
			trb.SetLabels(map[string]string{"foo": "bar"})
			Expect(k8sClient.Update(test.Ctx, trb)).To(Succeed(), "there should be no error updating the TeamRoleBinding on the central cluster")

			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the remote cluster")
				return g.Expect(remoteClusterRoleBinding.Subjects).To(Equal(expected))
			}).Should(BeTrue(), "the remote RoleBinding should eventually be reconciled")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})
})
