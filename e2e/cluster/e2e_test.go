// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build clusterE2E

package cluster

import (
	"context"
	"log"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/e2e/cluster/expect"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const (
	remoteClusterName = "remote-int-cluster"
)

var (
	env              *shared.TestEnv
	ctx              context.Context
	adminClient      client.Client
	remoteClient     client.Client
	remoteRestClient *clientutil.RestClientGetter
	testStartTime    time.Time
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv(greenhousev1alpha1.AddToScheme).WithOrganization(ctx, "./testdata/organization.yaml")
	adminClient = env.GetClient(shared.AdminClient)
	Expect(adminClient).ToNot(BeNil(), "admin client should not be nil")
	remoteClient = env.GetClient(shared.RemoteClient)
	Expect(remoteClient).ToNot(BeNil(), "remote client should not be nil")
	remoteRestClient = env.GetRESTClient(shared.RemoteRESTClient)
	Expect(remoteRestClient).ToNot(BeNil(), "remote rest client should not be nil")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Cluster E2E", Ordered, func() {
	Context("Cluster Happy Path 🤖", Ordered, func() {
		It("should onboard remote cluster", func() {
			By("onboarding remote cluster")
			shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace)
		})
		It("should have a cluster resource created", func() {
			By("verifying if the cluster resource is created")
			Eventually(func(g Gomega) bool {
				err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
				g.Expect(err).ToNot(HaveOccurred())
				return true
			}).Should(BeTrue(), "cluster resource should be created")

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
		})

		It("should verify remote cluster objects", func() {
			By("verifying the remote cluster version")
			cluster := &greenhousev1alpha1.Cluster{}
			err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, cluster)
			statusKubeVersion := cluster.Status.KubernetesVersion
			dc, err := remoteRestClient.ToDiscoveryClient()
			Expect(err).NotTo(HaveOccurred(), "there should be no error creating the discovery client")
			expectedKubeVersion, err := dc.ServerVersion()
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the server version")
			Expect(statusKubeVersion).To(Equal(expectedKubeVersion.String()))

			By("verifying if the managed namespace exists in the remote cluster")
			ns := &corev1.Namespace{}
			err = remoteClient.Get(ctx, client.ObjectKey{Name: env.TestNamespace}, ns)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the managed namespace")
			Expect(ns.Status.Phase).To(Equal(corev1.NamespaceActive), "managed namespace must be active")

			By("verifying if the cluster role binding exists in the remote cluster")
			crb := &rbacv1.ClusterRoleBinding{}
			err = remoteClient.Get(ctx, client.ObjectKey{Name: shared.ManagedResourceName}, crb)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the cluster role binding")

			By("verifying if the greenhouse service account exists in the remote cluster")
			sa := &corev1.ServiceAccount{}
			err = remoteClient.Get(ctx, client.ObjectKey{Name: shared.ManagedResourceName, Namespace: env.TestNamespace}, sa)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the service account")

			By("verifying if the greenhouse service account is bound to the cluster role binding")
			found := false
			for _, subject := range crb.Subjects {
				if subject.Kind == rbacv1.ServiceAccountKind && subject.Name == sa.Name && subject.Namespace == env.TestNamespace {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "managed service account should be bound to the cluster role binding")

			By("verifying if the greenhouse service account has cluster role binding as owner reference")
			isOwner := shared.IsResourceOwnedByOwner(crb, sa)
			log.Printf("isOwner: %v\n", isOwner)
			Expect(isOwner).To(BeTrue(), "service account should have an owner reference")
		})

		It("should successfully schedule the cluster for deletion", func() {
			By("verifying for the cluster deletion schedule annotation")
			expect.ClusterDeletionIsScheduled(ctx, adminClient, remoteClusterName, env.TestNamespace)
		})

		It("should successfully off-board remote cluster", func() {
			shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
		})
	})

	Context("Cluster Fail Path 😵", Ordered, func() {
		It("should onboard remote cluster", func() {
			By("onboarding remote cluster")
			shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace)
		})
		It("should have a cluster resource created", func() {
			By("verifying if the cluster resource is created")
			Eventually(func(g Gomega) bool {
				err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
				g.Expect(err).ToNot(HaveOccurred())
				return true
			}).Should(BeTrue(), "cluster resource should be created")

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
		})

		It("should reach not ready state when kubeconfig has expired", func() {
			By("simulating a revoking of greenhouse service account token")
			expect.RevokingRemoteServiceAccount(ctx, adminClient, remoteClient, shared.ManagedResourceName, remoteClusterName, env.TestNamespace)
		})

		It("should restore the cluster to ready state", func() {
			expect.RestoreCluster(ctx, adminClient, remoteClusterName, env.TestNamespace, env.RemoteKubeConfigBytes)
		})
	})
})
