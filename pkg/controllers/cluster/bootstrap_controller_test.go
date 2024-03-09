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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const bootstrapClusterNamespace = "bootstrap"

var _ = Describe("Bootstrap controller", Ordered, func() {

	BeforeAll(func() {
		Expect(test.K8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: bootstrapClusterNamespace}})).NotTo(HaveOccurred(), "there should be no error creating the test namespace")
	})
	Context("When reconciling a kubeConfig secret", func() {

		It("Should correctly set cluster.Spec.AccessMode and cluster.Status with valid remote kubeconfig and if remote api server is reachable",
			func() {
				By("Creating a secret with a valid kubeconfig for a remote cluster")
				validKubeConfigSecret := corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: corev1.GroupName,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-boostrap",
						Namespace: bootstrapClusterNamespace,
						Labels: map[string]string{
							"greenhouse/test": "bootstrap",
						},
					},
					Data: map[string][]byte{
						greenhouseapis.KubeConfigKey: remoteKubeConfig,
					},
					Type: greenhouseapis.SecretTypeKubeConfig,
				}
				Expect(test.K8sClient.Create(test.Ctx, &validKubeConfigSecret, &client.CreateOptions{})).Should(Succeed())

				By("Checking the accessmode is set correctly")
				getCluster := &greenhousev1alpha1.Cluster{}
				id := types.NamespacedName{Name: "test-boostrap", Namespace: bootstrapClusterNamespace}
				Eventually(func(g Gomega) bool {
					g.Expect(test.K8sClient.Get(test.Ctx, id, getCluster)).Should(Succeed(), "the cluster should have been created")
					g.Expect(getCluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeDirect), "the cluster accessmode should be set to direct")
					readyCondition := getCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
					g.Expect(readyCondition).ToNot(BeNil())
					g.Expect(readyCondition.Type).To(Equal(greenhousev1alpha1.ReadyCondition))
					g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
					return true
				}).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")
			})

		It("Should correctly set cluster.Spec.AccessMode and cluster.Status with invalid remote kubeconfig",
			func() {
				By("Creating a secret with an invalid kubeconfig for a remote cluster")
				kubeConfigString := string(remoteKubeConfig)
				//invalidate host
				invalidKubeConfigString := strings.Replace(kubeConfigString, "127", "128", -1)
				invalidKubeConfigSecret := corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: corev1.GroupName,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-invalid-bootstrap",
						Namespace: bootstrapClusterNamespace,
						Labels: map[string]string{
							"greenhouse/test": "bootstrap",
						},
					},
					Data: map[string][]byte{
						greenhouseapis.KubeConfigKey: []byte(invalidKubeConfigString),
					},
					Type: greenhouseapis.SecretTypeKubeConfig,
				}
				Expect(test.K8sClient.Create(test.Ctx, &invalidKubeConfigSecret, &client.CreateOptions{})).Should(Succeed())

				By("Checking the accessmode is set correctly")
				getCluster := &greenhousev1alpha1.Cluster{}
				id := types.NamespacedName{Name: "test-invalid-bootstrap", Namespace: bootstrapClusterNamespace}
				Eventually(func(g Gomega) bool {
					g.Expect(test.K8sClient.Get(test.Ctx, id, getCluster)).Should(Succeed(), "the cluster should have been created")
					g.Expect(getCluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeHeadscale), "the cluster accessmode should be set to headscale")
					g.Expect(getCluster.Status.Conditions).ToNot(BeNil(), "status conditions should be present")
					readyCondition := getCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
					g.Expect(readyCondition).ToNot(BeNil())
					g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse), "the ready condition should be set to false")
					return true
				}).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")

				By("Deleting the invalid cluster")
				Expect(test.K8sClient.Delete(test.Ctx, getCluster)).To(Succeed(), "There should be no error deleting the invalid cluster resource")
			})
	})
})
