// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	setup      *test.TestSetup
	clusterUT  *greenhousev1alpha1.Cluster
	teamUT     *greenhousev1alpha1.Team
	teamRoleUT *greenhousev1alpha1.TeamRole
)

var _ = Describe("Validate ClusterRole & RoleBinding on Remote Cluster", Ordered, func() {
	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
		clusterUT = setup.OnboardCluster(test.Ctx, "test-cluster", remoteKubeConfig)

		teamUT = setup.CreateTeam(test.Ctx, "test-team", test.WithMappedIDPGroup(testTeamIDPGroup))

		By("creating a TeamRole on the central cluster")
		teamRoleUT = setup.CreateTeamRole(test.Ctx, "test-role")
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
			err := remoteK8sClient.List(test.Ctx, remoteCRBList, listOpts...)
			if err != nil || len(remoteCRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoleBindings left to list on the remote cluster")

		// check that all RoleBindings are eventually deleted on the remote cluster
		remoteRBList := &rbacv1.RoleBindingList{}
		Eventually(func() bool {
			err := remoteK8sClient.List(test.Ctx, remoteRBList, listOpts...)
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
			err := remoteK8sClient.List(test.Ctx, remoteList, listOpts...)
			if err != nil || len(remoteList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoles left to list on the remote cluster")
	})

	Context("When creating a Greenhouse TeamRoleBinding with namespaces on the central cluster", func() {
		It("Should create a ClusterRole and RoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-rolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterUT.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
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
				test.WithClusterName(clusterUT.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      greenhouseapis.RBACPrefix + trb.Name,
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})

	Context("When creating a Greenhouse TeamRoleBinding with and without namespaces on the central cluster", func() {
		It("Should create a ClusterRole, ClusterRoleBinding and TeamRoleBinding on the remote cluster", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterUT.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      greenhouseapis.RBACPrefix + trb.Name,
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("creating a TeamRoleBinding on the central cluster")
			trbNoNamespaces := setup.CreateTeamRoleBinding(test.Ctx, "teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterUT.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the RoleBinding created on the remote cluster")
			remoteRoleBinding := &rbacv1.RoleBinding{}
			remoteRoleBindingName := types.NamespacedName{
				Name:      trbNoNamespaces.GetRBACName(),
				Namespace: trbNoNamespaces.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
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
				test.WithClusterName(clusterUT.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      trb.GetRBACName(),
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return !remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("updating the Greenhouse TeamRole on the central cluster")
			teamRoleUT.Spec.Rules[0].Verbs = []string{"get"}
			Expect(k8sClient.Update(test.Ctx, teamRoleUT)).To(Succeed(), "there should be no error updating the TeamRole on the central cluster")
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the remote cluster")
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
				test.WithClusterName(clusterUT.Name),
				test.WithNamespaces(setup.Namespace()))

			By("validating the ClusterRole created on the remote cluster")
			remoteClusterRole := &rbacv1.ClusterRole{}
			remoteClusterRoleName := types.NamespacedName{
				Name: teamRoleUT.GetRBACName(),
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteRoleBindingName, remoteRoleBinding)).To(Succeed(), "there should be no error getting the RoleBinding from the Remote Cluster")
				return !remoteRoleBinding.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the TeamRoleBinding")
			Expect(remoteRoleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteRoleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))

			By("updating the Greenhouse TeamRole on the central cluster")
			teamRoleUT.Spec.Rules[0].Verbs = []string{"get"}
			err := k8sClient.Update(test.Ctx, teamRoleUT)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the remote cluster")
				return g.Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))
			}).Should(BeTrue(), "there should be no error getting the ClusterRole from the remote cluster")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})
	Context("When tampering with a RoleBinding on the Remote Cluster", func() {
		It("should reconcile the remote RoleBinding", func() {
			By("creating a TeamRoleBinding without Namespaces on the central cluster")
			trb := setup.CreateTeamRoleBinding(test.Ctx, "test-teamrolebinding",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterUT.Name))

			By("validating the ClusterRoleBinding created on the remote cluster")
			remoteClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			remoteClusterRoleBindingName := types.NamespacedName{
				Name:      greenhouseapis.RBACPrefix + trb.Name,
				Namespace: trb.Namespace,
			}
			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(context.TODO(), remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the Remote Cluster")
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
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleName, remoteClusterRole)).To(Succeed(), "there should be no error getting the ClusterRole from the Remote Cluster")
				return g.Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules)) &&
					!remoteClusterRole.CreationTimestamp.IsZero()
			}).Should(BeTrue(), "there should be no error getting the ClusterRole")
			Expect(remoteClusterRole.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			Expect(remoteClusterRole.Name).To(ContainSubstring(teamRoleUT.Name))
			Expect(remoteClusterRole.Rules).To(Equal(teamRoleUT.Spec.Rules))

			By("altering the RoleBinding on the remote cluster")
			Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the remote cluster")
			expected := remoteClusterRoleBinding.DeepCopy().Subjects
			remoteClusterRoleBinding.Subjects = append(remoteClusterRoleBinding.Subjects, rbacv1.Subject{Kind: "User", Name: "foobar", APIGroup: "rbac.authorization.k8s.io"})
			Expect(remoteK8sClient.Update(test.Ctx, remoteClusterRoleBinding)).To(Succeed(), "there should be no error updating the ClusterRoleBinding on the remote cluster")

			By("triggering the reconcile of the central cluster TeamRoleBinding with a noop update")
			Expect(k8sClient.Get(test.Ctx, types.NamespacedName{Name: trb.Name, Namespace: trb.Namespace}, trb)).To(Succeed(), "there should be no error getting the TeamRoleBinding from the central cluster")
			// changing the labels to trigger the reconciliation in this test.
			trb.SetLabels(map[string]string{"foo": "bar"})
			Expect(k8sClient.Update(test.Ctx, trb)).To(Succeed(), "there should be no error updating the TeamRoleBinding on the central cluster")

			Eventually(func(g Gomega) bool {
				g.Expect(remoteK8sClient.Get(test.Ctx, remoteClusterRoleBindingName, remoteClusterRoleBinding)).To(Succeed(), "there should be no error getting the ClusterRoleBinding from the remote cluster")
				return g.Expect(remoteClusterRoleBinding.Subjects).To(Equal(expected))
			}).Should(BeTrue(), "the remote RoleBinding should eventually be reconciled")

			By("cleaning up the test")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, trb)
		})
	})
})
