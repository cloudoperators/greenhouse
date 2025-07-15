// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	clusterutils "github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("KubeConfig controller", func() {
	Context("When reconciling a cluster resource", func() {
		const (
			directAccessTestCase = "direct-access"
		)

		var (
			cluster = greenhousev1alpha1.Cluster{}

			remoteEnvTest    *envtest.Environment
			remoteKubeConfig []byte
			setup            *test.TestSetup
			team             *greenhousev1alpha1.Team
		)

		BeforeEach(func() {
			_, _, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, directAccessTestCase)
			team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			setup.CreateOrganizationWithOIDCConfig(test.Ctx, setup.Namespace(), team.Name)
		})

		AfterEach(func() {
			test.MustDeleteCluster(test.Ctx, test.K8sClient, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()})
			test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
			Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
		})

		It("Should correctly have created resources in remote cluster and provided a valid greenhouse kubeconfig secret",
			func() {
				By("Creating a secret with a valid kubeconfig for a remote cluster")
				secret := setup.CreateSecret(test.Ctx, directAccessTestCase,
					test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
					test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
					test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

				By("Checking the cluster resource with the same name as the secret has been created")
				Eventually(func() error {
					return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: secret.Name, Namespace: setup.Namespace()}, &cluster)
				}).Should(Succeed(), fmt.Sprintf("eventually the cluster %s should exist", secret.Name))

				By("checking the cluster's secret has an owner reference to the cluster")
				expectedOwnerReference := metav1.OwnerReference{
					Kind:       "Cluster",
					APIVersion: "greenhouse.sap/v1alpha1",
					UID:        cluster.UID,
					Name:       cluster.Name,
				}
				secretUT := corev1.Secret{}
				Eventually(func(g Gomega) bool {
					g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: cluster.Name, Namespace: setup.Namespace()}, &secretUT)).To(Succeed())
					g.Expect(secretUT.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference), "the kubeconfig secret should have an owner reference to the cluster")
					return true
				}).Should(BeTrue(), "eventually the secret should have an owner reference to the cluster")

				By("Checking namespace has been created in remote cluster")
				remoteClient, err := clientutil.NewK8sClientFromCluster(test.Ctx, test.K8sClient, &cluster)
				Expect(err).ToNot(HaveOccurred(), "there should be no error creating a new k8s client from the cluster")

				Eventually(func() error {
					var namespace = new(corev1.Namespace)
					return remoteClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: setup.Namespace()}, namespace)
				}).Should(Succeed(), fmt.Sprintf("eventually the namespace %s should exist", setup.Namespace()))

				By("Checking service account has been created in remote cluster")
				Eventually(func() error {
					var serviceAccount = new(corev1.ServiceAccount)
					return remoteClient.Get(test.Ctx, types.NamespacedName{Namespace: setup.Namespace(), Name: clusterutils.ServiceAccountName}, serviceAccount)
				}).Should(Succeed(), fmt.Sprintf("eventually the service account %s/%s should exist", setup.Namespace(), clusterutils.ServiceAccountName))

				By("Checking clusterRoleBinding has been created in remote cluster")
				clusterRoleBindingName := "greenhouse"
				Eventually(func() error {
					var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
					return remoteClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: clusterRoleBindingName}, clusterRoleBinding)
				}).Should(Succeed(), fmt.Sprintf("eventually the clusterRoleBinding %s should exist", clusterRoleBindingName))

				By("Checking greenhouse kubeConfig has been created and verifying validity")
				greenhouseKubeConfigSecret := corev1.Secret{}
				Eventually(func() map[string][]byte {
					err := test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: cluster.Name, Namespace: setup.Namespace()}, &greenhouseKubeConfigSecret)
					Expect(err).ToNot(HaveOccurred())
					return greenhouseKubeConfigSecret.Data
				}).Should(HaveKey(greenhouseapis.GreenHouseKubeConfigKey),
					"eventually the secret data should contain the key "+greenhouseapis.GreenHouseKubeConfigKey,
				)

				greenhouseRestClientGetter, err := clientutil.NewRestClientGetterFromSecret(&greenhouseKubeConfigSecret, setup.Namespace(), clientutil.WithPersistentConfig())
				Expect(err).NotTo(HaveOccurred(), "there should be no error getting the rest client getter from the secret")
				greenhouseRemoteConfig, err := greenhouseRestClientGetter.ToRESTConfig()
				Expect(err).NotTo(HaveOccurred(), "there should be no error creating a restConfig from a kubeConfig")
				Expect(greenhouseRemoteConfig).
					ToNot(BeNil(), "the greenhouse restConfig should not be nil")

				By("Getting the version of the remote cluster")
				kubeVersion, err := clientutil.GetKubernetesVersion(greenhouseRestClientGetter)
				Expect(err).NotTo(HaveOccurred(), "there should be no error getting the kubernetes version")
				Expect(kubeVersion).
					ToNot(BeNil(), "the kubernetes version should not be nil")
			})
	})
})
