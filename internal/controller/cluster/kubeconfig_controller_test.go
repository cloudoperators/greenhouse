// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"

	clusterpkg "github.com/cloudoperators/greenhouse/internal/controller/cluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ClusterKubeconfig controller", Ordered, func() {
	const (
		kubeconfigTestCase = "kubeconfig"
		clusterName        = "test-cluster"

		oidcIssuer          = "https://the-issuer"
		oidcClientID        = "the-client-id"
		oidcClientIDKey     = "clientID"
		oidcClientSecret    = "the-client-secret"
		oidcClientSecretKey = "clientSecret"
		oidcSecretResource  = "the-oidc-secret"

		mappedAdminID = "the-admin-id"
	)

	var (
		team         *v1alpha1.Team
		cluster      = v1alpha1.Cluster{}
		organization = &v1alpha1.Organization{}
		oidcSecret   = &corev1.Secret{}

		setup test.TestSetup
	)

	BeforeAll(func() {
		setup = *test.NewTestSetup(test.Ctx, test.K8sClient, kubeconfigTestCase)

		By("Creating a support-group Team")
		team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

		By("Creating a secret with OIDC data")
		oidcSecret = setup.CreateSecret(test.Ctx, oidcSecretResource,
			test.WithSecretNamespace(setup.Namespace()),
			test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithSecretData(map[string][]byte{
				oidcClientIDKey:     []byte(oidcClientID),
				oidcClientSecretKey: []byte(oidcClientSecret),
			}))

		By("Creating an organization with OIDC config")
		organization = setup.CreateOrganization(test.Ctx, setup.Namespace(),
			test.WithMappedAdminIDPGroup(mappedAdminID),
			test.WithOIDCConfig(oidcIssuer, oidcSecretResource, oidcClientIDKey, oidcClientSecretKey),
		)

		By("Creating a Secret with a valid KubeConfig")
		secret := setup.CreateSecret(test.Ctx, clusterName,
			test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
			test.WithSecretNamespace(organization.Name),
			test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithSecretData(map[string][]byte{
				greenhouseapis.KubeConfigKey:           test.KubeConfig,
				greenhouseapis.GreenHouseKubeConfigKey: test.KubeConfig,
			}))

		By("Checking the cluster resource has been created")
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: secret.Name, Namespace: setup.Namespace()}, &cluster)
		}).Should(Succeed(), fmt.Sprintf("eventually the cluster %s should exist", secret.Name))

	})

	AfterAll(func() {
		test.MustDeleteCluster(test.Ctx, test.K8sClient, client.ObjectKeyFromObject(&cluster))
		Expect(test.K8sClient.Delete(test.Ctx, oidcSecret)).To(Succeed())
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
	})

	clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
	It("should create ClusterKubeconfig resource and reconcile", func() {
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Status.Conditions.IsReadyTrue()
		}).Should(BeTrue())

		// ensure conditions are initialized
		Expect(clusterKubeconfig.Status.Conditions.Conditions).Should(HaveLen(len(clusterpkg.ExposedKubeconfigConditions)))
	})

	It("should ClusterKubeconfig has correct kubeconfig data", func() {
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")

		// compare fields
		Expect(clusterKubeconfig.Spec.Kubeconfig.APIVersion).Should(Equal("v1"))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Kind).Should(Equal("Config"))

		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Name).Should(Equal("oidc"))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config).Should(HaveLen(3))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-id"]).Should(Equal(oidcClientID))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-secret"]).Should(Equal(oidcClientSecret))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["idp-issuer-url"]).Should(Equal(oidcIssuer))

		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Name).Should(Equal(cluster.Name))
		kubeCfg, err := clientcmd.Load(test.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.Server).Should(Equal(kubeCfg.Clusters[kubeCfg.Contexts[kubeCfg.CurrentContext].Cluster].Server))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.CertificateAuthorityData).Should(Equal(kubeCfg.Clusters[kubeCfg.Contexts[kubeCfg.CurrentContext].Cluster].CertificateAuthorityData))

		Expect(clusterKubeconfig.Spec.Kubeconfig.Contexts).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Contexts[0].Name).Should(Equal(clusterKubeconfig.Spec.Kubeconfig.CurrentContext))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Contexts[0].Context.AuthInfo).Should(Equal(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].Name))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Contexts[0].Context.Cluster).Should(Equal(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Name))
	})

	It("should have cluster labels on ClusterKubeconfig", func() {

		clusterToBeUpdated := v1alpha1.Cluster{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterToBeUpdated)).To(Succeed())
		if clusterToBeUpdated.Labels == nil {
			clusterToBeUpdated.Labels = map[string]string{}
		}
		clusterToBeUpdated.Labels["key"] = "value"
		Expect(test.K8sClient.Update(test.Ctx, &clusterToBeUpdated)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Eventually(func(g Gomega) string {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Labels["key"]
		}).Should(Equal("value"))
	})

	It("should update ClusterKubeconfig when cluster secret data changes", func() {

		nextKubeconfig := []byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCkEKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://updated:9090
  name: updated-cluster
contexts:
- context:
    cluster: updated-cluster
    user: updated-user
  name: updated-context
current-context: updated-context
kind: Config
preferences: {}
users:
- name: updated-user
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCkEKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCkEKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
`)

		cfg, err := clientcmd.Load(nextKubeconfig)
		Expect(err).NotTo(HaveOccurred())

		secret := corev1.Secret{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &secret)).To(Succeed())

		secret.Data[greenhouseapis.GreenHouseKubeConfigKey] = nextKubeconfig
		Expect(test.K8sClient.Update(test.Ctx, &secret)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Eventually(func(g Gomega) string {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.Server
		}).Should(Equal(cfg.Clusters[cfg.Contexts[cfg.CurrentContext].Cluster].Server))

		// check for other fields
		Expect(clusterKubeconfig.Status.Conditions.IsReadyTrue()).To(BeTrue())
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.CertificateAuthorityData).Should(Equal(cfg.Clusters[cfg.Contexts[cfg.CurrentContext].Cluster].CertificateAuthorityData))

	})

	It("should update ClusterKubeconfig when organization OIDC data changes", func() {
		organization := v1alpha1.Organization{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: setup.Namespace(), Namespace: setup.Namespace()}, &organization)).To(Succeed())
		organization.Spec.Authentication.OIDCConfig.Issuer = "new-issuer-url"
		Expect(test.K8sClient.Update(test.Ctx, &organization)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Eventually(func(g Gomega) string {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["idp-issuer-url"]
		}).Should(Equal("new-issuer-url"))

		// check other fields
		Expect(clusterKubeconfig.Status.Conditions.IsReadyTrue()).To(BeTrue())
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config).Should(HaveLen(3))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-id"]).Should(Equal(oidcClientID))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-secret"]).Should(Equal(oidcClientSecret))

	})

	It("should update ClusterKubeconfig when organization OIDC secret changes", func() {
		secretToBeUpdated := corev1.Secret{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: oidcSecretResource, Namespace: setup.Namespace()}, &secretToBeUpdated)).To(Succeed())
		secretToBeUpdated.Data[oidcClientIDKey] = []byte("new-client-id")
		Expect(test.K8sClient.Update(test.Ctx, &secretToBeUpdated)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Eventually(func(g Gomega) string {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-id"]
		}).Should(Equal("new-client-id"))

		// check other fields
		Expect(clusterKubeconfig.Status.Conditions.IsReadyTrue()).To(BeTrue())
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo).Should(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config).Should(HaveLen(3))
		Expect(clusterKubeconfig.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-secret"]).Should(Equal(oidcClientSecret))

	})

	It("should fail with ClusterKubeconfig when organization OIDC data is not found", func() {
		organization := v1alpha1.Organization{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: setup.Namespace(), Namespace: setup.Namespace()}, &organization)).To(Succeed())
		organization.Spec.Authentication.OIDCConfig = nil
		Expect(test.K8sClient.Update(test.Ctx, &organization)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}

		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).ShouldNot(HaveOccurred(), "There should be no error getting the ClusterKubeconfig resource")
			return clusterKubeconfig.Status.Conditions.IsReadyTrue()
		}).Should(BeFalse())

		// check for reconcile failed condition
		Expect(clusterKubeconfig.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)).NotTo(BeNil())
		Expect(clusterKubeconfig.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition).Status).To(Equal(metav1.ConditionTrue))
	})
})
