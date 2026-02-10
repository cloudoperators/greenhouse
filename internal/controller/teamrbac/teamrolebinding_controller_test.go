// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate ClusterRole & RoleBinding on Remote Cluster", Ordered, func() {
	var (
		setup      *test.TestSetup
		clusterA   *greenhousev1alpha1.Cluster
		clusterB   *greenhousev1alpha1.Cluster
		teamUT     *greenhousev1alpha1.Team
		teamRoleUT *greenhousev1alpha1.TeamRole
	)

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
		clusterA = setup.OnboardCluster(test.Ctx, "test-cluster-a", clusterAKubeConfig, test.WithClusterLabel("cluster", "a"), test.WithClusterLabel("rbac", "true"))
		Expect(test.SetClusterReadyCondition(test.Ctx, test.K8sClient, clusterA, metav1.ConditionTrue)).To(Succeed(), "there should be no error setting the cluster to ready")
		clusterB = setup.OnboardCluster(test.Ctx, "test-cluster-b", clusterBKubeConfig, test.WithClusterLabel("cluster", "b"), test.WithClusterLabel("rbac", "true"))
		Expect(test.SetClusterReadyCondition(test.Ctx, test.K8sClient, clusterB, metav1.ConditionTrue)).To(Succeed(), "there should be no error setting the cluster to ready")

		teamUT = setup.CreateTeam(test.Ctx, "test-team", test.WithMappedIDPGroup(testTeamIDPGroup))

		By("creating a TeamRole on the central cluster")
		teamRoleUT = setup.CreateTeamRole(test.Ctx, "test-role", test.WithLabels(map[string]string{"aggregate": "true"}))
		test.EventuallyCreated(test.Ctx, setup.Client, teamRoleUT)
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

	Context("When editing clusterName or clusterSelector on a TeamRoleBinding", func() {
		It("should remove the RoleBinding on the cluster that is no longer referenced by clusterName and reconcile the clusters referenced by clusterSelector", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithUsernames([]string{"test-user-1"}))
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

				rbacReadyCondition := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
				g.Expect(rbacReadyCondition).ToNot(BeNil(), "RBACReady condition on TeamRoleBinding should not be nil")
				g.Expect(rbacReadyCondition.Status).To(Equal(metav1.ConditionTrue), "RBACReady condition on TeamRoleBinding should be True")
				g.Expect(rbacReadyCondition.Reason).To(Equal(greenhousev1alpha2.RBACReconciled), "RBACReady condition reason on TeamRoleBinding should be RBACReconciled")
				readyCondition := trb.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "Ready condition on TeamRoleBinding should not be nil")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition on TeamRoleBinding should be True")
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("updating the TeamRoleBinding with a selector matching clusterB")
			trb.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, trb, func() error {
				trb.Spec.ClusterSelector.Name = ""
				trb.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
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
			Expect(remoteRoleBinding.Subjects).To(HaveLen(2))

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
			trb.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, trb, func() error {
				trb.Spec.ClusterSelector.Name = ""
				trb.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "b"}}
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
				test.WithNamespaces(setup.Namespace()),
				test.WithUsernames([]string{"test-user-1"}))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the RoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteRoleBinding.Subjects).To(HaveLen(2))

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

		It("Should propagate the error correctly when a non-existing Namespace was specified", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces("non-existing-namespace"))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func() error {
				return clusterAKubeClient.Get(test.Ctx, remoteRoleBindingName, remoteRoleBinding)
			}).ShouldNot(Succeed(), "there should be an error getting the RoleBinding from the Remote Cluster")

			By("ensuring the TeamRoleBinding propagates the error")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the TeamRoleBinding from central Cluster")

				g.Expect(trb.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				statusMessage := "Failed to reconcile RoleBindings: namespaces \"non-existing-namespace\" not found"
				g.Expect(trb.Status.PropagationStatus).To(ContainElement(And(
					HaveField("ClusterName", clusterA.Name),
					HaveField("Condition.Status", metav1.ConditionFalse),
					HaveField("Condition.Reason", greenhousev1alpha2.RoleBindingFailed),
					HaveField("Condition.Message", statusMessage),
				)), "there should be a correct PropagationStatus")

				rbacReadyCondition := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
				g.Expect(rbacReadyCondition).ToNot(BeNil(), "RBACReady condition should not be nil on the TeamRoleBinding")
				g.Expect(rbacReadyCondition.Status).To(Equal(metav1.ConditionFalse), "RBACReady condition should be False on the TeamRoleBinding")
				g.Expect(rbacReadyCondition.Reason).To(Equal(greenhousev1alpha2.RBACReconcileFailed), "RBACReady condition should have the correct Reason")
				g.Expect(rbacReadyCondition.Message).To(Equal("Error reconciling TeamRoleBinding for clusters: test-cluster-a"), "RBACReady condition should have the correct Message")
			}).Should(Succeed(), "TeamRoleBinding should propagate the error")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})

		It("Should create namespaces when flag is set to true", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterB.Name),
				test.WithNamespaces("non-existing-namespace-1", "non-existing-namespace-2"),
				test.WithCreateNamespace(true))

			By("checking that the Namespace is created")
			namespace := &corev1.Namespace{}
			Eventually(func(g Gomega) {
				err := clusterBKubeClient.Get(test.Ctx, types.NamespacedName{Name: "non-existing-namespace-1"}, namespace)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the non-existing namespace")
			}).Should(Succeed())
			Eventually(func(g Gomega) {
				err := clusterBKubeClient.Get(test.Ctx, types.NamespacedName{Name: "non-existing-namespace-2"}, namespace)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the non-existing namespace")
			}).Should(Succeed())

			By("ensuring the Team Role Binding has been reconciled")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the TeamRoleBinding from central Cluster")

				rbacReadyCondition := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
				g.Expect(rbacReadyCondition).ToNot(BeNil(), "RBACReady condition should not be nil on the TeamRoleBinding")
				g.Expect(rbacReadyCondition.Status).To(Equal(metav1.ConditionTrue), "RBACReady condition should be True on the TeamRoleBinding")
			}).Should(Succeed(), "TeamRoleBinding should propagate the error")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding with non-existing namespaces on the central cluster", func() {
		It("Should fail to create ClusterRole and RoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces("non-existing-namespace", setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(clusterAKubeClient.Get(test.Ctx, remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
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

			By("validating the TeamRoleBinding PropagationStatus for the remote cluster is false")
			actTRB := &greenhousev1alpha2.TeamRoleBinding{}
			actTRBKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, actTRBKey, actTRB)).To(Succeed(), "there should be no error getting the TeamRoleBinding from the Central Cluster")
				g.Expect(actTRB.Status.PropagationStatus).To(HaveLen(1), "the TeamRoleBinding should be propagated to one cluster")
				g.Expect(actTRB.Status.PropagationStatus[0].Message).To(ContainSubstring("Failed to reconcile RoleBindings"))
			}).Should(Succeed(), "there should be no error validating the PropagationStatus")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding that does not match any clusters", func() {
		It("Should be in a not-Ready state", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}))

			By("validating the TeamRoleBinding Ready Status is false")
			actTRB := &greenhousev1alpha2.TeamRoleBinding{}
			actTRBKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, actTRBKey, actTRB)).To(Succeed(), "there should be no error getting the TeamRoleBinding from the Central Cluster")
				rbacReadyCondition := actTRB.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
				g.Expect(rbacReadyCondition).ToNot(BeNil(), "RBACReady condition on TeamRoleBinding should not be nil")
				g.Expect(rbacReadyCondition.Status).To(Equal(metav1.ConditionFalse), "RBACReady condition on TeamRoleBinding should be False")
				g.Expect(rbacReadyCondition.Reason).To(Equal(greenhousev1alpha2.EmptyClusterList), "RBACReady condition reason on TeamRoleBinding should be EmptyClusterList")
				readyCondition := actTRB.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "Ready condition on TeamRoleBinding should not be nil")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition on TeamRoleBinding should be True")
			}).Should(Succeed(), "there should be no error validating the PropagationStatus")

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
				// The dev-env does not start the Kubernetes ControllerManager, thus the ClusterRoles are not reconciled and we can only check that it was
				// created with the correct AggregationRule.
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

	Context("Changing Namespaces in a Greenhouse TeamRoleBinding", func() {
		It("Should create and delete RoleBindings based on .Spec.Namespaces in the remote cluster", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-trb-1",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(setup.Namespace()))
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

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

			By("Creating additional namespaces on the remote cluster")
			firstAdditionalNamespace := "test-namespace-1"
			secondAdditionalNamespace := "test-namespace-2"
			var namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: firstAdditionalNamespace,
				}}
			Expect(clusterAKubeClient.Create(test.Ctx, namespace)).To(Succeed(), "there should be no error creating the first additional namespace on the remote cluster")
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: secondAdditionalNamespace,
				}}
			Expect(clusterAKubeClient.Create(test.Ctx, namespace)).To(Succeed(), "there should be no error creating the second additional namespace on the remote cluster")

			By("Adding namespaces to TRB")
			err := setup.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)
			Expect(err).ToNot(HaveOccurred(), "There should be no error getting the TeamRoleBinding")
			trb.Spec.Namespaces = append(trb.Spec.Namespaces, firstAdditionalNamespace, secondAdditionalNamespace)
			Expect(setup.Update(test.Ctx, trb)).To(Succeed(), "There should be no error updating the Namespaces in TeamRoleBinding")

			By("Checking that additional RoleBindings have been deployed")
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(3), "There should be exactly three RoleBindings deployed to the remote cluster")

				g.Expect(slices.ContainsFunc(remoteRoleBindings.Items, func(roleBinding rbacv1.RoleBinding) bool {
					return roleBinding.Namespace == firstAdditionalNamespace
				})).To(BeTrue(), "There should be a RoleBinding for the first added namespace")
				g.Expect(slices.ContainsFunc(remoteRoleBindings.Items, func(roleBinding rbacv1.RoleBinding) bool {
					return roleBinding.Namespace == secondAdditionalNamespace
				})).To(BeTrue(), "There should be a RoleBinding for the second added namespace")
			}).Should(Succeed(), "Two additional RoleBindings should be deployed to remote cluster")

			By("Removing some namespaces from TRB")
			err = setup.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)
			Expect(err).ToNot(HaveOccurred(), "There should be no error getting the TeamRoleBinding")
			trb.Spec.Namespaces = []string{firstAdditionalNamespace}
			Expect(setup.Update(test.Ctx, trb)).To(Succeed(), "There should be no error updating the Namespaces in TeamRoleBinding")

			By("Checking that RoleBindings have been removed from remote cluster")
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding deployed to the remote cluster")

				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(firstAdditionalNamespace))
			}).Should(Succeed(), "Two RoleBindings should be removed from remote cluster")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})

		It("Should remove all deployed RoleBindings when remote cluster is no longer matching cluster selector", func() {
			By("counting all RoleBindings on the remote cluster")
			var allRemoteRoleBindings = new(rbacv1.RoleBindingList)
			Expect(clusterAKubeClient.List(test.Ctx, allRemoteRoleBindings)).To(Succeed(), "There should be no error listing remote RoleBindings")
			previousRemoteRoleBindingsTotalCount := len(allRemoteRoleBindings.Items)

			// These namespaces must be created in one of the previous tests.
			firstAdditionalNamespace := "test-namespace-1"
			secondAdditionalNamespace := "test-namespace-2"

			By("creating a TeamRoleBinding with ClusterSelector and Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "a"}}),
				test.WithNamespaces(firstAdditionalNamespace, secondAdditionalNamespace))

			trbKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}

			By("validating the RoleBindings created on the remote clusterA")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(2), "There should be exactly two RoleBindings deployed to the remote cluster")

				g.Expect(slices.ContainsFunc(remoteRoleBindings.Items, func(roleBinding rbacv1.RoleBinding) bool {
					return roleBinding.Namespace == firstAdditionalNamespace
				})).To(BeTrue(), "There should be a RoleBinding for the first added namespace")
				g.Expect(slices.ContainsFunc(remoteRoleBindings.Items, func(roleBinding rbacv1.RoleBinding) bool {
					return roleBinding.Namespace == secondAdditionalNamespace
				})).To(BeTrue(), "There should be a RoleBinding for the second added namespace")
			}).Should(Succeed(), "Two RoleBindings should be deployed to remote clusterA")

			By("updating the TeamRoleBinding with a different selector and changed namespaces")
			_, err := clientutil.CreateOrPatch(test.Ctx, k8sClient, trb, func() error {
				trb.Spec.ClusterSelector.Name = ""
				trb.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"not": "matching"}}
				// Replace with a single different namespace.
				trb.Spec.Namespaces = []string{setup.Namespace()}
				return nil
			})
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the TeamRoleBinding")

			By("validating that all the deployed RoleBindings are removed from the remote clusterA")
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(BeEmpty(), "There should be no RoleBindings deployed to the remote clusterA")
			}).Should(Succeed(), "Both deployed RoleBindings should be removed from remote clusterA")

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(BeEmpty(), "the TeamRoleBinding should not be propagated to any cluster")
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("checking that no other RoleBindings have been deleted")
			Expect(clusterAKubeClient.List(test.Ctx, allRemoteRoleBindings)).To(Succeed(), "There should be no error listing all remote RoleBindings")
			currentRemoteRoleBindingsTotalCount := len(allRemoteRoleBindings.Items)
			Expect(currentRemoteRoleBindingsTotalCount).To(Equal(previousRemoteRoleBindingsTotalCount), "There should be the same total number of RoleBindings on the remote cluster")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})

		It("Should delete only deployed RoleBindings from the remote cluster when the TRB is deleted", func() {
			By("counting all RoleBindings on the remote cluster")
			var allRemoteRoleBindings = new(rbacv1.RoleBindingList)
			Expect(clusterAKubeClient.List(test.Ctx, allRemoteRoleBindings)).To(Succeed(), "There should be no error listing remote RoleBindings")
			previousRemoteRoleBindingsTotalCount := len(allRemoteRoleBindings.Items)

			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-trb-1",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			Eventually(func(g Gomega) {
				var deployedRoleBindings = new(rbacv1.RoleBindingList)
				err := clusterAKubeClient.List(test.Ctx, deployedRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(deployedRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

			By("removing the TeamRoleBinding from the central cluster")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)

			By("checking that there is the same total number of RoleBindings as before")
			Eventually(func(g Gomega) {
				var deployedRoleBindings = new(rbacv1.RoleBindingList)
				err := clusterAKubeClient.List(test.Ctx, deployedRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trb.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(deployedRoleBindings.Items).To(BeEmpty(), "There should be no deployed RoleBindings on the remote cluster")

				g.Expect(clusterAKubeClient.List(test.Ctx, allRemoteRoleBindings)).To(Succeed(), "There should be no error listing all remote RoleBindings")
				currentRemoteRoleBindingsTotalCount := len(allRemoteRoleBindings.Items)
				g.Expect(currentRemoteRoleBindingsTotalCount).To(Equal(previousRemoteRoleBindingsTotalCount), "There should be the same total number of RoleBindings on the remote cluster")
			}).Should(Succeed(), "there should be the same total number of RoleBindings as before the TRB creation")
		})
	})

	Context("Ensure not-Ready Clusters are handled correctly", func() {
		It("should skip over the non-Ready cluster", func() {
			By("creating a non-Ready cluster")
			Expect(test.SetClusterReadyCondition(test.Ctx, k8sClient, clusterA, metav1.ConditionFalse)).To(Succeed(), "there should be no error setting clusterA to not ready")

			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"rbac": "true"}}))
			trbKey := types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}

			By("validating the TeamRoleBinding's status is updated")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(test.Ctx, trbKey, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding")
				g.Expect(trb.Status.PropagationStatus).To(HaveLen(2), "the TeamRoleBinding should contain 2 clusters")
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterA.Name), HaveField("Condition.Status", Equal(metav1.ConditionFalse)))))
				g.Expect(trb.Status.PropagationStatus).Should(ContainElement(And(HaveField("ClusterName", clusterB.Name), HaveField("Condition.Status", Equal(metav1.ConditionTrue)))))

				rbacReadyCondition := trb.Status.GetConditionByType(greenhousev1alpha2.RBACReady)
				g.Expect(rbacReadyCondition).ToNot(BeNil(), "RBACReady condition on TeamRoleBinding should not be nil")
				g.Expect(rbacReadyCondition.Status).To(Equal(metav1.ConditionTrue), "RBACReady condition on TeamRoleBinding should be True")
				readyCondition := trb.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "Ready condition on TeamRoleBinding should not be nil")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse), "Ready condition on TeamRoleBinding should be False")
			}).Should(Succeed(), "the TeamRoleBindings status should reflect the current status")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})
})
