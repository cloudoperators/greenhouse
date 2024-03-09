// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	clusterpkg "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	directAccessClusterNamespace = "direct-access"
	directAccessClusterName      = "test-direct-access"
)

var _ = Describe("KubeConfig controller", func() {
	Context("When reconciling a cluster resource", func() {

		//delete all secrets and clusters
		BeforeEach(func() {
			// Restart pristine remote environment to mitigate https://book.kubebuilder.io/reference/envtest.html#:~:text=EnvTest%20does%20not%20support%20namespace,and%20never%20actually%20be%20reclaimed
			err := remoteEnvTest.Stop()
			Expect(err).
				NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
			bootstrapRemoteCluster()

			Expect(test.K8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: directAccessClusterNamespace}})).NotTo(HaveOccurred(), "there should be no error creating the test namespace")

		})

		AfterEach(func() {
			test.MustDeleteCluster(test.Ctx, test.K8sClient, types.NamespacedName{Name: directAccessClusterName, Namespace: directAccessClusterNamespace})
			test.MustDeleteSecretWithLabel(test.Ctx, test.K8sClient, "kubeconfig")
		})

		It("Should correctly have created resources in remote cluster and provided a valid greenhouse kubeconfig secret",
			func() {
				By("Creating a secret with a valid kubeconfig for a remote cluster")
				validKubeConfigSecret := corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: corev1.GroupName,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      directAccessClusterName,
						Namespace: directAccessClusterNamespace,
						Labels: map[string]string{
							"greenhouse/test": "kubeconfig",
						},
					},
					Data: map[string][]byte{
						greenhouseapis.KubeConfigKey: remoteKubeConfig,
					},
					Type: greenhouseapis.SecretTypeKubeConfig,
				}
				Expect(test.K8sClient.Create(test.Ctx, &validKubeConfigSecret, &client.CreateOptions{})).
					Should(Succeed(), "there should be no error creating the kubeconfig secret")

				By("Checking namespace has been created in remote cluster")
				config, err := clientcmd.RESTConfigFromKubeConfig(validKubeConfigSecret.Data[greenhouseapis.KubeConfigKey])
				Expect(err).
					ShouldNot(HaveOccurred(), "there should be no error creating a restConfig from the kubeConfig")
				remoteK8sClient, err := clientutil.NewK8sClient(config)
				Expect(err).
					ShouldNot(HaveOccurred(), "there should be no error creating a new k8s client from the restConfig")

				Eventually(func() error {
					var namespace = new(corev1.Namespace)
					return remoteK8sClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: directAccessClusterNamespace}, namespace)
				}).Should(Succeed(), fmt.Sprintf("eventually the namespace %s should exist", directAccessClusterNamespace))

				By("Checking service account has been created in remote cluster")
				Eventually(func() error {
					var serviceAccount = new(corev1.ServiceAccount)
					return remoteK8sClient.Get(test.Ctx, types.NamespacedName{Namespace: directAccessClusterNamespace, Name: clusterpkg.ExportServiceAccountName}, serviceAccount)
				}).Should(Succeed(), fmt.Sprintf("eventually the service account %s/%s should exist", directAccessClusterNamespace, clusterpkg.ExportServiceAccountName))

				By("Checking clusterRoleBinding has been created in remote cluster")
				clusterRoleBindingName := "greenhouse"
				Eventually(func() error {
					var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
					return remoteK8sClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: clusterRoleBindingName}, clusterRoleBinding)
				}).Should(Succeed(), fmt.Sprintf("eventually the clusterRoleBinding %s should exist", clusterRoleBindingName))

				By("Checking greenhouse kubeConfig has been created and verifying validity")
				greenhouseKubeConfigSecret := corev1.Secret{}
				Eventually(func() map[string][]byte {
					err := test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: directAccessClusterName, Namespace: directAccessClusterNamespace}, &greenhouseKubeConfigSecret)
					Expect(err).ToNot(HaveOccurred())
					return greenhouseKubeConfigSecret.Data
				}).Should(HaveKey(greenhouseapis.GreenHouseKubeConfigKey),
					fmt.Sprintf("eventually the secret data should contain the key %s", greenhouseapis.GreenHouseKubeConfigKey),
				)

				greenhouseRestClientGetter, err := clientutil.NewRestClientGetterFromSecret(&greenhouseKubeConfigSecret, directAccessClusterNamespace, clientutil.WithPersistentConfig())
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
