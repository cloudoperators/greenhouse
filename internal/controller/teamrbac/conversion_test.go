// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("TeamRoleBinding Conversion", Ordered, func() {
	var (
		setup      *test.TestSetup
		clusterA   *greenhousev1alpha1.Cluster
		teamUT     *greenhousev1alpha1.Team
		teamRoleUT *greenhousev1alpha1.TeamRole
	)

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac-conversion")
		clusterA = setup.OnboardCluster(test.Ctx, "test-cluster-a", clusterAKubeConfig, test.WithClusterLabel("cluster", "a"), test.WithClusterLabel("rbac", "true"))
		Expect(test.SetClusterReadyCondition(test.Ctx, test.K8sClient, clusterA, metav1.ConditionTrue)).To(Succeed(), "there should be no error setting the cluster to ready")

		teamUT = setup.CreateTeam(test.Ctx, "test-team-conversion", test.WithMappedIDPGroup(testTeamIDPGroup))

		By("creating a TeamRole on the central cluster")
		teamRoleUT = setup.CreateTeamRole(test.Ctx, "test-role-conversion", test.WithLabels(map[string]string{"aggregate": "true"}))
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

	Context("Validate Conversion of TeamRoleBinding resource", func() {
		It("should correctly convert the TRB with ClusterName from v1alpha1 to the hub version (v1alpha2)", func() {
			By("creating a TeamRoleBinding with v1alpha1 version on the central cluster")
			trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "TeamRoleBinding",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      setup.RandomizeName("test-trb-1"),
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef:      teamRoleUT.Name,
					TeamRef:          teamUT.Name,
					ClusterName:      clusterA.Name,
					Namespaces:       []string{setup.Namespace()},
					CreateNamespaces: true,
				},
			}
			Expect(setup.Create(test.Ctx, trbV1alpha1)).To(Succeed(), "TeamRoleBinding in v1alpha1 version should be created successfully")

			By("validating the conversion to v1alpha2 version")
			trbV1alpha2 := &greenhousev1alpha2.TeamRoleBinding{}
			trbKey := types.NamespacedName{Name: trbV1alpha1.Name, Namespace: trbV1alpha1.Namespace}
			Expect(setup.Get(test.Ctx, trbKey, trbV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 TeamRoleBinding")

			Expect(trbV1alpha2.Spec.ClusterSelector.Name).To(Equal(trbV1alpha1.Spec.ClusterName), ".Spec.ClusterSelector.Name in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(trbV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector.LabelSelector in TRB should be correctly converted between versions")

			Expect(trbV1alpha2.Spec.TeamRoleRef).To(Equal(trbV1alpha1.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.TeamRef).To(Equal(trbV1alpha1.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.Namespaces).To(Equal(trbV1alpha1.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.CreateNamespaces).To(Equal(trbV1alpha1.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")

			By("validating the RoleBinding created on the remote cluster")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(setup.Namespace()))
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

			By("cleaning up the created TeamRoleBinding")
			test.EventuallyDeleted(test.Ctx, setup.Client, trbV1alpha2)
		})

		It("should correctly convert the TRB with ClusterName from v1alpha2 to v1alpha1", func() {
			By("creating a TeamRoleBinding with v1alpha2 on the central cluster")
			trbV1alpha2 := setup.CreateTeamRoleBinding(test.Ctx, "test-trb-1",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterName(clusterA.Name),
				test.WithNamespaces(setup.Namespace()),
				test.WithCreateNamespace(true))

			By("validating the conversion to v1alpha1 version")
			trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{}
			trbKey := types.NamespacedName{Name: trbV1alpha2.Name, Namespace: trbV1alpha2.Namespace}
			Expect(setup.Get(test.Ctx, trbKey, trbV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 TeamRoleBinding")

			Expect(trbV1alpha1.Spec.ClusterName).To(Equal(trbV1alpha2.Spec.ClusterSelector.Name), ".Spec.ClusterName in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.ClusterSelector).To(Equal(trbV1alpha2.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector in TRB should be correctly converted between versions")

			Expect(trbV1alpha1.Spec.TeamRoleRef).To(Equal(trbV1alpha2.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.TeamRef).To(Equal(trbV1alpha2.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.Namespaces).To(Equal(trbV1alpha2.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.CreateNamespaces).To(Equal(trbV1alpha2.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")

			By("validating the RoleBinding created on the remote cluster")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(setup.Namespace()))
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

			By("cleaning up the created TeamRoleBinding")
			test.EventuallyDeleted(test.Ctx, setup.Client, trbV1alpha1)
		})

		It("should correctly convert the TRB with LabelSelector from v1alpha1 to the hub version (v1alpha2)", func() {
			By("creating a TeamRoleBinding with v1alpha1 version on the central cluster")
			trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "TeamRoleBinding",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      setup.RandomizeName("test-trb-1"),
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.TeamRoleBindingSpec{
					TeamRoleRef:      teamRoleUT.Name,
					TeamRef:          teamUT.Name,
					ClusterSelector:  metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "a"}},
					Namespaces:       []string{setup.Namespace()},
					CreateNamespaces: true,
				},
			}
			Expect(setup.Create(test.Ctx, trbV1alpha1)).To(Succeed(), "TeamRoleBinding in v1alpha1 version should be created successfully")

			By("validating the conversion to v1alpha2 version")
			trbV1alpha2 := &greenhousev1alpha2.TeamRoleBinding{}
			trbKey := types.NamespacedName{Name: trbV1alpha1.Name, Namespace: trbV1alpha1.Namespace}
			Expect(setup.Get(test.Ctx, trbKey, trbV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 TeamRoleBinding")

			Expect(trbV1alpha2.Spec.ClusterSelector.Name).To(Equal(trbV1alpha1.Spec.ClusterName), ".Spec.ClusterSelector.Name in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(trbV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector.LabelSelector in TRB should be correctly converted between versions")

			Expect(trbV1alpha2.Spec.TeamRoleRef).To(Equal(trbV1alpha1.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.TeamRef).To(Equal(trbV1alpha1.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.Namespaces).To(Equal(trbV1alpha1.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
			Expect(trbV1alpha2.Spec.CreateNamespaces).To(Equal(trbV1alpha1.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")

			By("validating the RoleBinding created on the remote cluster")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(setup.Namespace()))
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

			By("cleaning up the created TeamRoleBinding")
			test.EventuallyDeleted(test.Ctx, setup.Client, trbV1alpha2)
		})

		It("should correctly convert the TRB with LabelSelector from v1alpha2 to v1alpha1", func() {
			By("creating a TeamRoleBinding with v1alpha2 on the central cluster")
			trbV1alpha2 := setup.CreateTeamRoleBinding(test.Ctx, "test-trb-1",
				test.WithTeamRoleRef(teamRoleUT.Name),
				test.WithTeamRef(teamUT.Name),
				test.WithClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"cluster": "a"}}),
				test.WithNamespaces(setup.Namespace()),
				test.WithCreateNamespace(true))

			By("validating the conversion to v1alpha1 version")
			trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{}
			trbKey := types.NamespacedName{Name: trbV1alpha2.Name, Namespace: trbV1alpha2.Namespace}
			Expect(setup.Get(test.Ctx, trbKey, trbV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 TeamRoleBinding")

			Expect(trbV1alpha1.Spec.ClusterName).To(Equal(trbV1alpha2.Spec.ClusterSelector.Name), ".Spec.ClusterName in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.ClusterSelector).To(Equal(trbV1alpha2.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector in TRB should be correctly converted between versions")

			Expect(trbV1alpha1.Spec.TeamRoleRef).To(Equal(trbV1alpha2.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.TeamRef).To(Equal(trbV1alpha2.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.Namespaces).To(Equal(trbV1alpha2.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
			Expect(trbV1alpha1.Spec.CreateNamespaces).To(Equal(trbV1alpha2.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")

			By("validating the RoleBinding created on the remote cluster")
			var remoteRoleBindings = new(rbacv1.RoleBindingList)
			Eventually(func(g Gomega) {
				err := clusterAKubeClient.List(test.Ctx, remoteRoleBindings, &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
				})
				g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
				g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
				roleBinding := remoteRoleBindings.Items[0]
				g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
				g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
				g.Expect(roleBinding.Namespace).To(Equal(setup.Namespace()))
			}).Should(Succeed(), "there should be no error getting the RoleBindings")

			By("cleaning up the created TeamRoleBinding")
			test.EventuallyDeleted(test.Ctx, setup.Client, trbV1alpha1)
		})
	})
})
