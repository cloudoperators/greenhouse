// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
			remoteK8sClient  client.Client
			setup            *test.TestSetup
			team             *greenhousev1alpha1.Team
		)

		BeforeEach(func() {
			_, remoteK8sClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
			remoteEnvTest.ControlPlaneStopTimeout = 60 * time.Second
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, directAccessTestCase)
			team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			setup.CreateOrganizationWithOIDCConfig(test.Ctx, setup.Namespace(), team.Name)
		})

		AfterEach(func() {
			test.MustDeleteCluster(test.Ctx, test.K8sClient, &cluster)
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

		It("Should regenerate greenhousekubeconfig when the key is deleted from the secret", func() {
			By("Creating a secret with a valid kubeconfig for a remote cluster")
			secret := setup.CreateSecret(test.Ctx, directAccessTestCase+"-recreate",
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
				test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

			By("Checking the cluster resource with the same name as the secret has been created")
			Eventually(func() error {
				return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: secret.Name, Namespace: setup.Namespace()}, &cluster)
			}).Should(Succeed(), fmt.Sprintf("eventually the cluster %s should exist", secret.Name))

			By("Waiting for the greenhousekubeconfig key to be created")
			greenhouseKubeConfigSecret := corev1.Secret{}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: secret.Name, Namespace: setup.Namespace()}, &greenhouseKubeConfigSecret)
				if err != nil {
					return false
				}
				return clientutil.IsSecretContainsKey(&greenhouseKubeConfigSecret, greenhouseapis.GreenHouseKubeConfigKey)
			}).Should(BeTrue(), "eventually the greenhousekubeconfig key should be created")

			By("Deleting the greenhousekubeconfig key from the secret")
			greenhouseKubeConfigSecret = corev1.Secret{}
			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: secret.Name, Namespace: setup.Namespace()}, &greenhouseKubeConfigSecret)).To(Succeed())
			delete(greenhouseKubeConfigSecret.Data, greenhouseapis.GreenHouseKubeConfigKey)
			Expect(test.K8sClient.Update(test.Ctx, &greenhouseKubeConfigSecret)).To(Succeed())

			By("Verifying the greenhousekubeconfig key is missing")
			greenhouseKubeConfigSecret = corev1.Secret{}
			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: secret.Name, Namespace: setup.Namespace()}, &greenhouseKubeConfigSecret)).To(Succeed())
			Expect(clientutil.IsSecretContainsKey(&greenhouseKubeConfigSecret, greenhouseapis.GreenHouseKubeConfigKey)).To(BeFalse(), "the greenhousekubeconfig key should be missing")

			By("Waiting for the controller to regenerate the greenhousekubeconfig key")
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: secret.Name, Namespace: setup.Namespace()}, &greenhouseKubeConfigSecret)
				if err != nil {
					return false
				}
				return clientutil.IsSecretContainsKey(&greenhouseKubeConfigSecret, greenhouseapis.GreenHouseKubeConfigKey)
			}).Should(BeTrue(), "eventually the greenhousekubeconfig key should be recreated by the controller")

			By("Verifying the regenerated kubeconfig is valid")
			greenhouseRestClientGetter, err := clientutil.NewRestClientGetterFromSecret(&greenhouseKubeConfigSecret, setup.Namespace(), clientutil.WithPersistentConfig())
			Expect(err).NotTo(HaveOccurred(), "there should be no error getting the rest client getter from the secret")
			greenhouseRemoteConfig, err := greenhouseRestClientGetter.ToRESTConfig()
			Expect(err).NotTo(HaveOccurred(), "there should be no error creating a restConfig from the regenerated kubeConfig")
			Expect(greenhouseRemoteConfig).ToNot(BeNil(), "the greenhouse restConfig should not be nil")
		})

		It("Should remove the TeamRoleBinding RBAC from the remote cluster when the cluster is deleted", func() {
			By("Creating a secret with a valid kubeconfig for a remote cluster")
			secret := setup.CreateSecret(test.Ctx, directAccessTestCase+"-team-rbac",
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
				test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(secret), &cluster)).To(Succeed())
				g.Expect(cluster.Status.IsReadyTrue()).To(BeTrue())
			}).Should(Succeed(), "eventually the cluster should be ready")

			By("Creating TeamRoleBinding RBAC in the remote cluster")
			clusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:   greenhouseapis.RBACPrefix + "test-role",
					Labels: map[string]string{greenhouseapis.LabelKeyRole: "test-role"},
				},
			}
			roleRef := rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: clusterRole.Name}
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   greenhouseapis.RBACPrefix + "test-cluster-rolebinding",
					Labels: map[string]string{greenhouseapis.LabelKeyRoleBinding: "test-cluster-rolebinding"},
				},
				RoleRef: roleRef,
			}
			roleBinding := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      greenhouseapis.RBACPrefix + "test-rolebinding",
					Namespace: metav1.NamespaceDefault,
					Labels:    map[string]string{greenhouseapis.LabelKeyRoleBinding: "test-rolebinding"},
				},
				RoleRef: roleRef,
			}
			teamRBAC := []client.Object{clusterRole, clusterRoleBinding, roleBinding}
			for _, obj := range teamRBAC {
				Expect(remoteK8sClient.Create(test.Ctx, obj)).To(Succeed(), "there should be no error creating %s in the remote cluster", obj.GetName())
			}

			By("Deleting the cluster")
			test.MustDeleteCluster(test.Ctx, test.K8sClient, &cluster)

			By("Checking the TeamRoleBinding RBAC is gone from the remote cluster")
			for _, obj := range teamRBAC {
				err := remoteK8sClient.Get(test.Ctx, client.ObjectKeyFromObject(obj), obj)
				Expect(apierrors.IsNotFound(err)).To(BeTrue(), "%s should be deleted from the remote cluster", obj.GetName())
			}
		})

		It("Should delete the cluster when greenhouse has no access to the remote cluster anymore", func() {
			By("Creating a secret with a valid kubeconfig for a remote cluster")
			secret := setup.CreateSecret(test.Ctx, directAccessTestCase+"-no-access",
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
				test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(secret), &cluster)).To(Succeed())
				g.Expect(cluster.Status.IsReadyTrue()).To(BeTrue())
			}).Should(Succeed(), "eventually the cluster should be ready")

			By("Creating TeamRoleBinding RBAC in the remote cluster")
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   greenhouseapis.RBACPrefix + "test-cluster-rolebinding",
					Labels: map[string]string{greenhouseapis.LabelKeyRoleBinding: "test-cluster-rolebinding"},
				},
				RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: greenhouseapis.RBACPrefix + "test-role"},
			}
			Expect(remoteK8sClient.Create(test.Ctx, clusterRoleBinding)).To(Succeed(), "there should be no error creating the clusterRoleBinding in the remote cluster")

			By("Revoking the permissions of greenhouse in the remote cluster")
			greenhouseClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			Expect(remoteK8sClient.Get(test.Ctx, client.ObjectKey{Name: clusterutils.ServiceAccountName}, greenhouseClusterRoleBinding)).To(Succeed())
			greenhouseClusterRoleBinding.Subjects = nil
			Expect(remoteK8sClient.Update(test.Ctx, greenhouseClusterRoleBinding)).To(Succeed(), "there should be no error removing the subjects of the greenhouse clusterRoleBinding")

			By("Deleting the cluster")
			test.MustDeleteCluster(test.Ctx, test.K8sClient, &cluster)

			By("Checking the TeamRoleBinding RBAC could not be removed from the remote cluster")
			Expect(remoteK8sClient.Get(test.Ctx, client.ObjectKeyFromObject(clusterRoleBinding), clusterRoleBinding)).To(Succeed(), "the clusterRoleBinding should still exist in the remote cluster")
		})
	})
})
