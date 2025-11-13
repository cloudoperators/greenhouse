// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build conversionE2E

package conversion

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	remoteClusterName = "remote-plugin-cluster"
	testTeamIDPGroup  = "test-idp-group"
)

var (
	env              *shared.TestEnv
	ctx              context.Context
	adminClient      client.Client
	remoteClient     client.Client
	remoteRestClient *clientutil.RestClientGetter
	testStartTime    time.Time
	teamUT           *greenhousev1alpha1.Team
	teamRoleUT       *greenhousev1alpha1.TeamRole
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conversion E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv()

	var err error
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")
	remoteRestClient = env.RemoteRestClientGetter
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	testStartTime = time.Now().UTC()

	By("creating a Team on the admin cluster")
	teamUT = test.NewTeam(ctx, "test-team", env.TestNamespace, test.WithMappedIDPGroup(testTeamIDPGroup), test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	Expect(adminClient.Create(ctx, teamUT)).To(Succeed(), "there should be no error creating a Team")

	By("onboarding remote cluster")
	shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, teamUT.Name)
	By("verifying if the cluster resource is created")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
		g.Expect(err).ToNot(HaveOccurred())
	}).Should(Succeed(), "cluster resource should be created")

	By("verifying the cluster status is ready")
	shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	By("cleaning the Team")
	test.EventuallyDeleted(ctx, adminClient, teamUT)
	env.GenerateGreenhouseControllerLogs(ctx, testStartTime)
})

var _ = Describe("Conversion E2E", Ordered, func() {
	BeforeEach(func() {
		By("creating a TeamRole on the admin cluster")
		teamRoleUT = test.NewTeamRole(ctx, "test-role"+"-"+rand.String(8), env.TestNamespace,
			test.WithLabels(map[string]string{"aggregate": "true"}))
		Expect(adminClient.Create(ctx, teamRoleUT)).To(Succeed(), "there should be no error creating a TeamRole")
	})

	AfterEach(func() {
		By("cleaning the TeamRole")
		test.EventuallyDeleted(ctx, adminClient, teamRoleUT)
	})

	// After all tests are run ensure there are no resources left behind on the remote cluster
	// This ensures the deletion of the Remote Resources is working correctly.
	AfterAll(func() {
		Eventually(func() bool {
			remoteCRBList := &rbacv1.ClusterRoleBindingList{}
			err := remoteClient.List(ctx, remoteCRBList, client.HasLabels{greenhouseapis.LabelKeyRoleBinding})
			if err != nil || len(remoteCRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoleBindings left to list on the remote cluster")

		// check that all RoleBindings are eventually deleted on the remote cluster
		remoteRBList := &rbacv1.RoleBindingList{}
		Eventually(func() bool {
			err := remoteClient.List(ctx, remoteRBList, client.HasLabels{greenhouseapis.LabelKeyRoleBinding})
			if err != nil || len(remoteRBList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no RoleBindings left to list on the remote cluster")

		// check that all ClusterRoles are eventually deleted on the remote cluster
		remoteList := &rbacv1.ClusterRoleList{}
		Eventually(func() bool {
			err := remoteClient.List(ctx, remoteList, client.HasLabels{greenhouseapis.LabelKeyRole})
			if err != nil || len(remoteList.Items) > 0 {
				return false
			}
			return true
		}).Should(BeTrue(), "there should be no ClusterRoles left to list on the remote cluster")
	})

	It("should correctly convert the TRB with ClusterName from v1alpha1 to the hub version (v1alpha2)", func() {
		By("creating a TeamRoleBinding with v1alpha1 version on the central cluster")
		trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "TeamRoleBinding",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-trb-1",
				Namespace: env.TestNamespace,
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: teamUT.Name},
			},
			Spec: greenhousev1alpha1.TeamRoleBindingSpec{
				TeamRoleRef:      teamRoleUT.Name,
				TeamRef:          teamUT.Name,
				ClusterName:      remoteClusterName,
				Namespaces:       []string{env.TestNamespace},
				CreateNamespaces: true,
			},
		}
		Expect(adminClient.Create(ctx, trbV1alpha1)).To(Succeed(), "TeamRoleBinding in v1alpha1 version should be created successfully")

		By("validating the conversion to v1alpha2 version")
		trbV1alpha2 := &greenhousev1alpha2.TeamRoleBinding{}
		trbKey := types.NamespacedName{Name: trbV1alpha1.Name, Namespace: trbV1alpha1.Namespace}
		Expect(adminClient.Get(ctx, trbKey, trbV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 TeamRoleBinding")

		Expect(trbV1alpha2.Spec.ClusterSelector.Name).To(Equal(trbV1alpha1.Spec.ClusterName), ".Spec.ClusterSelector.Name in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(trbV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector.LabelSelector in TRB should be correctly converted between versions")

		Expect(trbV1alpha2.Spec.TeamRoleRef).To(Equal(trbV1alpha1.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.TeamRef).To(Equal(trbV1alpha1.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.Namespaces).To(Equal(trbV1alpha1.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.CreateNamespaces).To(Equal(trbV1alpha1.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, teamUT.Name), "owned-by label should be correctly converted between versions")

		By("validating the RoleBinding created on the remote cluster")
		var remoteRoleBindings = new(rbacv1.RoleBindingList)
		Eventually(func(g Gomega) {
			err := remoteClient.List(ctx, remoteRoleBindings, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
			})
			g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
			g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
			roleBinding := remoteRoleBindings.Items[0]
			g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			g.Expect(roleBinding.Namespace).To(Equal(env.TestNamespace))
		}).Should(Succeed(), "there should be no error getting the RoleBindings")

		By("cleaning up the created TeamRoleBinding")
		test.EventuallyDeleted(ctx, adminClient, trbV1alpha2)
	})

	It("should correctly convert the TRB with ClusterName from v1alpha2 to v1alpha1", func() {
		By("creating a TeamRoleBinding with v1alpha2 on the central cluster")
		trbV1alpha2 := test.NewTeamRoleBinding(ctx, "test-trb-2", env.TestNamespace,
			test.WithTeamRoleRef(teamRoleUT.Name),
			test.WithTeamRef(teamUT.Name),
			test.WithClusterName(remoteClusterName),
			test.WithNamespaces(env.TestNamespace),
			test.WithCreateNamespace(true),
			test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, teamUT.Name),
		)
		Expect(adminClient.Create(ctx, trbV1alpha2)).To(Succeed(), "there should be no error creating the TeamRoleBinding")

		By("validating the conversion to v1alpha1 version")
		trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{}
		trbKey := types.NamespacedName{Name: trbV1alpha2.Name, Namespace: trbV1alpha2.Namespace}
		Expect(adminClient.Get(ctx, trbKey, trbV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 TeamRoleBinding")

		Expect(trbV1alpha1.Spec.ClusterName).To(Equal(trbV1alpha2.Spec.ClusterSelector.Name), ".Spec.ClusterName in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.ClusterSelector).To(Equal(trbV1alpha2.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector in TRB should be correctly converted between versions")

		Expect(trbV1alpha1.Spec.TeamRoleRef).To(Equal(trbV1alpha2.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.TeamRef).To(Equal(trbV1alpha2.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.Namespaces).To(Equal(trbV1alpha2.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.CreateNamespaces).To(Equal(trbV1alpha2.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, teamUT.Name), "owned-by label should be correctly converted between versions")

		By("validating the RoleBinding created on the remote cluster")
		var remoteRoleBindings = new(rbacv1.RoleBindingList)
		Eventually(func(g Gomega) {
			err := remoteClient.List(ctx, remoteRoleBindings, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
			})
			g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
			g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
			roleBinding := remoteRoleBindings.Items[0]
			g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			g.Expect(roleBinding.Namespace).To(Equal(env.TestNamespace))
		}).Should(Succeed(), "there should be no error getting the RoleBindings")

		By("cleaning up the created TeamRoleBinding")
		test.EventuallyDeleted(ctx, adminClient, trbV1alpha1)
	})

	It("should correctly convert the TRB with LabelSelector from v1alpha1 to the hub version (v1alpha2)", func() {
		By("Add labels to remote cluster")
		remoteCluster := &greenhousev1alpha1.Cluster{}
		err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
		Expect(err).ToNot(HaveOccurred())
		if remoteCluster.Labels == nil {
			remoteCluster.Labels = make(map[string]string, 1)
		}
		remoteCluster.Labels["app"] = "test-cluster"
		err = adminClient.Update(ctx, remoteCluster)
		Expect(err).ToNot(HaveOccurred())

		By("creating a TeamRoleBinding with v1alpha1 version on the central cluster")
		trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "TeamRoleBinding",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-trb-3",
				Namespace: env.TestNamespace,
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: teamUT.Name},
			},
			Spec: greenhousev1alpha1.TeamRoleBindingSpec{
				TeamRoleRef:      teamRoleUT.Name,
				TeamRef:          teamUT.Name,
				ClusterSelector:  metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-cluster"}},
				Namespaces:       []string{env.TestNamespace},
				CreateNamespaces: true,
			},
		}
		Expect(adminClient.Create(ctx, trbV1alpha1)).To(Succeed(), "TeamRoleBinding in v1alpha1 version should be created successfully")

		By("validating the conversion to v1alpha2 version")
		trbV1alpha2 := &greenhousev1alpha2.TeamRoleBinding{}
		trbKey := types.NamespacedName{Name: trbV1alpha1.Name, Namespace: trbV1alpha1.Namespace}
		Expect(adminClient.Get(ctx, trbKey, trbV1alpha2)).To(Succeed(), "There should be no error getting the v1alpha2 TeamRoleBinding")

		Expect(trbV1alpha2.Spec.ClusterSelector.Name).To(Equal(trbV1alpha1.Spec.ClusterName), ".Spec.ClusterSelector.Name in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.ClusterSelector.LabelSelector).To(Equal(trbV1alpha1.Spec.ClusterSelector), ".Spec.ClusterSelector.LabelSelector in TRB should be correctly converted between versions")

		Expect(trbV1alpha2.Spec.TeamRoleRef).To(Equal(trbV1alpha1.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.TeamRef).To(Equal(trbV1alpha1.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.Namespaces).To(Equal(trbV1alpha1.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Spec.CreateNamespaces).To(Equal(trbV1alpha1.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha2.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, teamUT.Name), "owned-by label should be correctly converted between versions")

		By("validating the RoleBinding created on the remote cluster")
		var remoteRoleBindings = new(rbacv1.RoleBindingList)
		Eventually(func(g Gomega) {
			err := remoteClient.List(ctx, remoteRoleBindings, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
			})
			g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
			g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
			roleBinding := remoteRoleBindings.Items[0]
			g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			g.Expect(roleBinding.Namespace).To(Equal(env.TestNamespace))
		}).Should(Succeed(), "there should be no error getting the RoleBindings")

		By("cleaning up the created TeamRoleBinding")
		test.EventuallyDeleted(ctx, adminClient, trbV1alpha2)
	})

	It("should correctly convert the TRB with LabelSelector from v1alpha2 to v1alpha1", func() {
		By("creating a TeamRoleBinding with v1alpha2 on the central cluster")
		trbV1alpha2 := test.NewTeamRoleBinding(ctx, "test-trb-4", env.TestNamespace,
			test.WithTeamRoleRef(teamRoleUT.Name),
			test.WithTeamRef(teamUT.Name),
			test.WithClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-cluster"},
			}),
			test.WithNamespaces(env.TestNamespace),
			test.WithCreateNamespace(true),
			test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, teamUT.Name),
		)
		Expect(adminClient.Create(ctx, trbV1alpha2)).To(Succeed(), "there should be no error creating the TeamRoleBinding")

		By("validating the conversion to v1alpha1 version")
		trbV1alpha1 := &greenhousev1alpha1.TeamRoleBinding{}
		trbKey := types.NamespacedName{Name: trbV1alpha2.Name, Namespace: trbV1alpha2.Namespace}
		Expect(adminClient.Get(ctx, trbKey, trbV1alpha1)).To(Succeed(), "There should be no error getting the v1alpha1 TeamRoleBinding")

		Expect(trbV1alpha1.Spec.ClusterName).To(Equal(trbV1alpha2.Spec.ClusterSelector.Name), ".Spec.ClusterName in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.ClusterSelector).To(Equal(trbV1alpha2.Spec.ClusterSelector.LabelSelector), ".Spec.ClusterSelector in TRB should be correctly converted between versions")

		Expect(trbV1alpha1.Spec.TeamRoleRef).To(Equal(trbV1alpha2.Spec.TeamRoleRef), ".Spec.TeamRoleRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.TeamRef).To(Equal(trbV1alpha2.Spec.TeamRef), ".Spec.TeamRef in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.Namespaces).To(Equal(trbV1alpha2.Spec.Namespaces), ".Spec.Namespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Spec.CreateNamespaces).To(Equal(trbV1alpha2.Spec.CreateNamespaces), ".Spec.CreateNamespaces in TRB should be correctly converted between versions")
		Expect(trbV1alpha1.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, teamUT.Name), "owned-by label should be correctly converted between versions")

		By("validating the RoleBinding created on the remote cluster")
		var remoteRoleBindings = new(rbacv1.RoleBindingList)
		Eventually(func(g Gomega) {
			err := remoteClient.List(ctx, remoteRoleBindings, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", trbV1alpha2.GetRBACName()),
			})
			g.Expect(err).ToNot(HaveOccurred(), "There should be no error listing remote RoleBindings")
			g.Expect(remoteRoleBindings.Items).To(HaveLen(1), "There should be exactly one RoleBinding on the remote cluster")
			roleBinding := remoteRoleBindings.Items[0]
			g.Expect(roleBinding.RoleRef.Name).To(HavePrefix(greenhouseapis.RBACPrefix))
			g.Expect(roleBinding.RoleRef.Name).To(ContainSubstring(teamRoleUT.Name))
			g.Expect(roleBinding.Namespace).To(Equal(env.TestNamespace))
		}).Should(Succeed(), "there should be no error getting the RoleBindings")

		By("cleaning up the created TeamRoleBinding")
		test.EventuallyDeleted(ctx, adminClient, trbV1alpha1)
	})
})
