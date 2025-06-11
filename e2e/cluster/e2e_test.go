// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build clusterE2E

package cluster

import (
	"context"
	"encoding/base64"
	"slices"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/cluster/expect"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

const (
	remoteClusterHName               = "remote-int-h-cluster"
	remoteClusterFName               = "remote-int-f-cluster"
	remoteOIDCClusterHName           = "remote-int-oidc-h-cluster"
	remoteOIDCClusterFName           = "remote-int-oidc-f-cluster"
	remoteOIDCClusterRoleBindingName = "greenhouse-odic-cluster-role-binding"
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
	var err error
	ctx = context.Background()
	env = shared.NewExecutionEnv()
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")
	remoteRestClient = env.RemoteRestClientGetter
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterHName, env.TestNamespace)
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterFName, env.TestNamespace)
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteOIDCClusterHName, env.TestNamespace)
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteOIDCClusterFName, env.TestNamespace)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Cluster E2E", Ordered, func() {
	// the context executes the tests for Cluster where a secret of type kubeconfig is provided
	// scenario: Happy Path
	Context("Cluster Happy Path 🤖", Ordered, func() {
		It("should onboard remote cluster", func() {
			By("onboarding remote cluster")
			shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterHName, env.TestNamespace)
		})
		It("should have a cluster resource created", func() {
			By("verifying if the cluster resource is created")
			Eventually(func(g Gomega) bool {
				err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterHName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
				g.Expect(err).ToNot(HaveOccurred())
				return true
			}).Should(BeTrue(), "cluster resource should be created")

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteClusterHName, env.TestNamespace)
		})

		It("should verify remote cluster objects", func() {
			By("verifying the remote cluster version")
			expect.VerifyClusterVersion(ctx, adminClient, remoteRestClient, remoteClusterHName, env.TestNamespace)

			By("verifying if the managed namespace exists in the remote cluster")
			ns := &corev1.Namespace{}
			err := remoteClient.Get(ctx, client.ObjectKey{Name: env.TestNamespace}, ns)
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
			found := slices.ContainsFunc(crb.Subjects, func(s rbacv1.Subject) bool {
				return s.Kind == rbacv1.ServiceAccountKind && s.Name == sa.Name && s.Namespace == env.TestNamespace
			})
			Expect(found).To(BeTrue(), "managed service account should be bound to the cluster role binding")

			By("verifying if the greenhouse service account has cluster role binding as owner reference")
			isOwner := shared.IsResourceOwnedByOwner(crb, sa)
			Expect(isOwner).To(BeTrue(), "service account should have an owner reference")
		})

		It("should successfully schedule the cluster for deletion", func() {
			By("verifying for the cluster deletion schedule annotation")
			expect.ClusterDeletionIsScheduled(ctx, adminClient, remoteClusterHName, env.TestNamespace)
		})
	})

	// the context executes the tests for Cluster where a secret of type kubeconfig is provided
	// scenario: Fail Path
	Context("Cluster Fail Path 😵", Ordered, func() {
		It("should onboard remote cluster", func() {
			By("onboarding remote cluster")
			shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterFName, env.TestNamespace)
		})
		It("should have a cluster resource created", func() {
			By("verifying if the cluster resource is created")
			Eventually(func(g Gomega) bool {
				err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterFName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
				g.Expect(err).ToNot(HaveOccurred())
				return true
			}).Should(BeTrue(), "cluster resource should be created")

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteClusterFName, env.TestNamespace)
		})

		It("should reach not ready state when kubeconfig has expired", func() {
			By("simulating a revoking of greenhouse service account token")
			expect.RevokingRemoteClusterAccess(ctx, adminClient, remoteClient, shared.ManagedResourceName, remoteClusterFName, env.TestNamespace)
		})
	})

	// the context executes the tests for Cluster where a secret of type oidc is provided
	// scenario: Happy Path
	Context("Cluster OIDC Happy Path 🤖", Ordered, func() {
		It("should setup role binding for OIDC on remote cluster", func() {
			By("setting up cluster role binding for OIDC on remote cluster")
			expect.SetupOIDCClusterRoleBinding(ctx, remoteClient, remoteOIDCClusterRoleBindingName, remoteOIDCClusterHName, env.TestNamespace)
		})
		It("should onboard remote cluster with OIDC", func() {
			By("onboarding remote cluster with OIDC")
			restClient := clientutil.NewRestClientGetterFromBytes(env.RemoteKubeConfigBytes, env.TestNamespace)
			restConfig, err := restClient.ToRESTConfig()
			Expect(err).NotTo(HaveOccurred(), "there should be no error creating the remote REST config")
			remoteAPIServerURL := restConfig.Host
			remoteCA := make([]byte, base64.StdEncoding.EncodedLen(len(restConfig.CAData)))
			base64.StdEncoding.Encode(remoteCA, restConfig.CAData)
			shared.OnboardRemoteOIDCCluster(ctx, adminClient, remoteCA, remoteAPIServerURL, remoteOIDCClusterHName, env.TestNamespace)

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteOIDCClusterHName, env.TestNamespace)

			By("verifying the remote cluster version")
			expect.VerifyClusterVersion(ctx, adminClient, remoteRestClient, remoteOIDCClusterHName, env.TestNamespace)

			By("verifying the oidc cluster service account is created")
			sa := &corev1.ServiceAccount{}
			err = adminClient.Get(ctx, client.ObjectKey{Name: remoteOIDCClusterHName, Namespace: env.TestNamespace}, sa)
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the service account")

			By("verifying the oidc cluster service account has the correct owner reference")
			ownerRef := clientutil.GetOwnerReference(sa, "Secret")
			Expect(ownerRef).NotTo(BeNil(), "service account should have an owner reference")
			Expect(ownerRef.Name).To(Equal(remoteOIDCClusterHName), "service account should have the correct owner reference")
		})

		It("should successfully off-board remote oidc cluster", func() {
			shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteOIDCClusterHName, env.TestNamespace)
			sa := &corev1.ServiceAccount{}
			err := adminClient.Get(ctx, client.ObjectKey{Name: remoteOIDCClusterHName, Namespace: env.TestNamespace}, sa)
			Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the service account should not exist")
		})
	})

	// the context executes the tests for Cluster where a secret of type oidc is provided
	// scenario: Fail Path
	// TODO: remove this test case
	Context("Cluster OIDC Fail Path 😵", Ordered, func() {
		It("should setup role binding for OIDC on remote cluster", func() {
			By("setting up cluster role binding for OIDC on remote cluster")
			expect.SetupOIDCClusterRoleBinding(ctx, remoteClient, remoteOIDCClusterRoleBindingName, remoteOIDCClusterFName, env.TestNamespace)
		})

		It("should onboard remote cluster with OIDC", func() {
			By("onboarding remote cluster with OIDC")
			restClient := clientutil.NewRestClientGetterFromBytes(env.RemoteKubeConfigBytes, env.TestNamespace)
			restConfig, err := restClient.ToRESTConfig()
			Expect(err).NotTo(HaveOccurred(), "there should be no error creating the remote REST config")
			remoteAPIServerURL := restConfig.Host
			remoteCA := make([]byte, base64.StdEncoding.EncodedLen(len(restConfig.CAData)))
			base64.StdEncoding.Encode(remoteCA, restConfig.CAData)
			shared.OnboardRemoteOIDCCluster(ctx, adminClient, remoteCA, remoteAPIServerURL, remoteOIDCClusterFName, env.TestNamespace)

			By("verifying the cluster status is ready")
			shared.ClusterIsReady(ctx, adminClient, remoteOIDCClusterFName, env.TestNamespace)
		})

		It("should reach not ready state when remote OIDC cluster role binding does not exist", func() {
			By("removing the OIDC cluster role binding")
			expect.RevokingOIDCClusterAccess(ctx, adminClient, remoteClient, remoteOIDCClusterRoleBindingName, remoteOIDCClusterFName, env.TestNamespace)

			By("verifying the allNodesReady condition is false")
			expect.VerifyFalseAllNodesReady(ctx, adminClient, remoteOIDCClusterFName, env.TestNamespace)
		})

		It("should restore the cluster to ready state", func() {
			By("restoring the OIDC cluster role binding")
			expect.SetupOIDCClusterRoleBinding(ctx, remoteClient, remoteOIDCClusterRoleBindingName, remoteOIDCClusterFName, env.TestNamespace)
			By("re-triggering the OIDC cluster reconciliation")
			expect.ReconcileReadyNotReady(ctx, adminClient, remoteOIDCClusterFName, env.TestNamespace, true)
		})
	})
})
