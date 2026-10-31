// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"encoding/base64"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
		clusterSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: clusterName, Namespace: setup.Namespace()}}
		test.EventuallyDeleted(test.Ctx, test.K8sClient, clusterSecret)
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
		// and reconcile failed should be false on success
		failed := clusterKubeconfig.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
		Expect(failed).NotTo(BeNil())
		Expect(failed.Status).To(Equal(metav1.ConditionFalse))
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

	It("should set kubernetes-version label from cluster status", func() {
		// Wait for the cluster controller to set KubernetesVersion from the envtest API server.
		clusterWithVersion := v1alpha1.Cluster{}
		Eventually(func(g Gomega) string {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterWithVersion)).To(Succeed())
			return clusterWithVersion.Status.KubernetesVersion
		}).ShouldNot(BeEmpty())

		// The kubeconfig controller must mirror the version in label-safe form (+→-).
		expectedLabel := strings.ReplaceAll(clusterWithVersion.Status.KubernetesVersion, "+", "-")
		Eventually(func(g Gomega) string {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			return ck.Labels[greenhouseapis.LabelKeyKubernetesVersion]
		}).Should(Equal(expectedLabel))

		// Inject a stale value directly on the ClusterKubeconfig and confirm the
		// controller reconciles it back to the correct value.
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Labels[greenhouseapis.LabelKeyKubernetesVersion] = "stale-value"
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed())

		// Controller must reconcile it back to the correct normalized value.
		Eventually(func(g Gomega) string {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			return ck.Labels[greenhouseapis.LabelKeyKubernetesVersion]
		}).Should(Equal(expectedLabel))
	})

	It("should update ClusterKubeconfig when cluster secret data changes", func() {

		baseCfg, err := clientcmd.Load(test.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		baseCluster := baseCfg.Clusters[baseCfg.Contexts[baseCfg.CurrentContext].Cluster]
		baseCluster.Server = "https://updated:9090"
		nextKubeconfig, err := clientcmd.Write(*baseCfg)
		Expect(err).NotTo(HaveOccurred())

		cfg, err := clientcmd.Load(nextKubeconfig)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			secret := corev1.Secret{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &secret)).To(Succeed())

			secret.Data[greenhouseapis.GreenHouseKubeConfigKey] = nextKubeconfig
			g.Expect(test.K8sClient.Update(test.Ctx, &secret)).To(Succeed())
		}).Should(Succeed(), "there should be no error updating the cluster secret")

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
		org := v1alpha1.Organization{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: setup.Namespace(), Namespace: setup.Namespace()}, &org)).To(Succeed())
		org.Spec.Authentication.OIDCConfig = nil
		Expect(test.K8sClient.Update(test.Ctx, &org)).To(Succeed())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &clusterKubeconfig)).Should(Succeed())
			return clusterKubeconfig.Status.Conditions.IsReadyTrue()
		}).Should(BeFalse())

		// reconcile failed must be True
		cond := clusterKubeconfig.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	})

	It("should clear ReconcileFailed once OIDC data is restored (no sticky failure)", func() {
		org := v1alpha1.Organization{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{
			Name: setup.Namespace(), Namespace: setup.Namespace(),
		}, &org)).To(Succeed())

		org.Spec.Authentication.OIDCConfig = &v1alpha1.OIDCConfig{
			Issuer: oidcIssuer,
			ClientIDReference: v1alpha1.SecretKeyReference{
				Name: oidcSecretResource, Key: oidcClientIDKey,
			},
			ClientSecretReference: v1alpha1.SecretKeyReference{
				Name: oidcSecretResource, Key: oidcClientSecretKey,
			},
		}
		Expect(test.K8sClient.Update(test.Ctx, &org)).To(Succeed())

		// 1) Ready should flip to True
		Eventually(func(g Gomega) bool {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{
				Name: cluster.Name, Namespace: setup.Namespace(),
			}, &ck)).Should(Succeed())
			return ck.Status.Conditions.IsReadyTrue()
		}).Should(BeTrue())

		// 2) ReconcileFailed should be False
		Eventually(func(g Gomega) metav1.ConditionStatus {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{
				Name: cluster.Name, Namespace: setup.Namespace(),
			}, &ck)).Should(Succeed())
			failed := ck.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
			if failed == nil {
				return metav1.ConditionUnknown
			}
			return failed.Status
		}).Should(Equal(metav1.ConditionFalse))
	})

	It("should override server URL via spec configOverrides", func() {
		const overrideServer = "https://new-tenant-api.example.com:6443"

		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = &v1alpha1.ClusterKubeconfigOverrides{
				Clusters: []v1alpha1.ClusterItemOverride{
					{Name: cluster.Name, Server: overrideServer},
				},
			}
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig should be updated with server override")

		Eventually(func(g Gomega) string {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Spec.Kubeconfig.Clusters[0].Cluster.Server
		}).Should(Equal(overrideServer))

		// Clean up for subsequent tests
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = nil
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed())
	})

	It("should override OIDC client-id inline via spec configOverrides", func() {
		const overrideClientID = "spec-override-client-id"

		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = &v1alpha1.ClusterKubeconfigOverrides{
				Users: []v1alpha1.AuthInfoOverride{
					{
						Name: "oidc@" + cluster.Name,
						OIDC: &v1alpha1.OIDCConfigOverride{ClientID: overrideClientID},
					},
				},
			}
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig should be updated with OIDC client-id override")

		Eventually(func(g Gomega) string {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-id"]
		}).Should(Equal(overrideClientID))

		// client-secret remains from org
		Eventually(func(g Gomega) string {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-secret"]
		}).Should(Equal(oidcClientSecret))

		// Clean up
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = nil
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed())
	})

	It("should override OIDC client-id via secretRef in spec configOverrides", func() {
		const overrideClientID = "secretref-override-client-id"

		overrideSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ckc-override-secret",
				Namespace: setup.Namespace(),
			},
			Data: map[string][]byte{
				"client-id": []byte(overrideClientID),
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, overrideSecret)).To(Succeed())
		DeferCleanup(func() {
			Expect(test.K8sClient.Delete(test.Ctx, overrideSecret)).To(Succeed())
		})

		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = &v1alpha1.ClusterKubeconfigOverrides{
				Users: []v1alpha1.AuthInfoOverride{
					{
						Name: "oidc@" + cluster.Name,
						OIDC: &v1alpha1.OIDCConfigOverride{
							ClientIDRef: &v1alpha1.SecretKeyReference{
								Name: overrideSecret.Name,
								Key:  "client-id",
							},
						},
					},
				},
			}
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig should be updated with secretRef override")

		Eventually(func(g Gomega) string {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Spec.Kubeconfig.AuthInfo[0].AuthInfo.AuthProvider.Config["client-id"]
		}).Should(Equal(overrideClientID))

		// Clean up
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = nil
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed())
	})

	It("should override context namespace via spec configOverrides", func() {
		const overrideNamespace = "kube-system"

		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = &v1alpha1.ClusterKubeconfigOverrides{
				Contexts: []v1alpha1.ContextItemOverride{
					{Name: cluster.Name, Namespace: overrideNamespace},
				},
			}
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig should be updated with context namespace override")

		Eventually(func(g Gomega) string {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Spec.Kubeconfig.Contexts[0].Context.Namespace
		}).Should(Equal(overrideNamespace))

		// Clean up
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = nil
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed())
	})

	It("should set ReconcileFailedCondition when secretRef in configOverrides references missing secret", func() {
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = &v1alpha1.ClusterKubeconfigOverrides{
				Users: []v1alpha1.AuthInfoOverride{
					{
						Name: "oidc@" + cluster.Name,
						OIDC: &v1alpha1.OIDCConfigOverride{
							ClientIDRef: &v1alpha1.SecretKeyReference{
								Name: "does-not-exist",
								Key:  "client-id",
							},
						},
					},
				},
			}
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig should be updated with bad secretRef")

		Eventually(func(g Gomega) bool {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			cond := fresh.Status.Conditions.GetConditionByType(v1alpha1.KubeconfigReconcileFailedCondition)
			return cond != nil && cond.Status == metav1.ConditionTrue
		}).Should(BeTrue(), "ReconcileFailedCondition should be set when secretRef target is missing")

		// Restore healthy state
		Eventually(func(g Gomega) {
			ck := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &ck)).To(Succeed())
			ck.Spec.ConfigOverrides = nil
			g.Expect(test.K8sClient.Update(test.Ctx, &ck)).To(Succeed())
		}).Should(Succeed(), "ClusterKubeconfig configOverrides should be cleared")

		Eventually(func(g Gomega) bool {
			fresh := v1alpha1.ClusterKubeconfig{}
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &fresh)).ShouldNot(HaveOccurred())
			return fresh.Status.Conditions.IsReadyTrue()
		}).Should(BeTrue(), "ClusterKubeconfig should be Ready after clearing configOverrides")
	})
})

var _ = Describe("ClusterKubeconfig controller with workload identity", func() {
	It("should build the kubeconfig from the OIDC secret", func() {
		const (
			clusterName  = "wi-cluster"
			apiServerURL = "https://api.remote.example.com"
			caCert       = "dummy-ca"
		)
		key := types.NamespacedName{Name: clusterName, Namespace: test.TestNamespace}
		c := fake.NewClientBuilder().WithScheme(test.GreenhouseV1Alpha1Scheme()).
			WithStatusSubresource(&v1alpha1.ClusterKubeconfig{}).
			WithObjects(
				test.NewOrganization(test.Ctx, test.TestNamespace, test.WithOIDCConfig(test.OIDCIssuer, test.OIDCSecretResource, test.OIDCClientIDKey, test.OIDCClientSecretKey)),
				test.NewSecret(test.OIDCSecretResource, test.TestNamespace, test.WithSecretData(map[string][]byte{
					test.OIDCClientIDKey:     []byte(test.OIDCClientID),
					test.OIDCClientSecretKey: []byte(test.OIDCClientSecret),
				})),
				test.NewCluster(test.Ctx, clusterName, test.TestNamespace, test.WithClusterAnnotations(map[string]string{
					greenhouseapis.ClusterWorkloadIdentityAnnotation: greenhouseapis.ClusterWorkloadIdentityEnabled,
				})),
				test.NewSecret(clusterName, test.TestNamespace,
					test.WithSecretType(greenhouseapis.SecretTypeOIDCConfig),
					test.WithSecretAnnotations(map[string]string{greenhouseapis.SecretAPIServerURLAnnotation: apiServerURL}),
					test.WithSecretData(map[string][]byte{
						greenhouseapis.SecretAPIServerCAKey: []byte(base64.StdEncoding.EncodeToString([]byte(caCert))),
					})),
			).Build()

		_, err := (&clusterpkg.KubeconfigReconciler{Client: c}).Reconcile(test.Ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		clusterKubeconfig := v1alpha1.ClusterKubeconfig{}
		Expect(c.Get(test.Ctx, key, &clusterKubeconfig)).To(Succeed())
		Expect(clusterKubeconfig.Status.Conditions.IsReadyTrue()).To(BeTrue())
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters).To(HaveLen(1))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.Server).To(Equal(apiServerURL))
		Expect(clusterKubeconfig.Spec.Kubeconfig.Clusters[0].Cluster.CertificateAuthorityData).To(Equal([]byte(caCert)))
	})
})
