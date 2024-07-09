// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	"github.com/cloudoperators/greenhouse/pkg/test"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var _ = Describe("OnboardSelf", Ordered, func() {
	Context("When onboarding itself as a cluster resource", func() {
		It("Should create a cluster resource for itself", func() {
			By("Creating a secret with a valid kubeconfig for a remote cluster")
			validKubeConfigSecret := corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: corev1.GroupName,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "greenhouse-self",
					Namespace: test.TestNamespace,
				},
				Data: map[string][]byte{
					greenhouseapis.KubeConfigKey: test.KubeConfig,
				},
				Type: greenhouseapis.SecretTypeKubeConfig,
			}
			Expect(test.K8sClient.Create(test.Ctx, &validKubeConfigSecret, &client.CreateOptions{})).Should(Succeed())

			By("Checking the resource exists and is ready")
			greenhouseCluster := &greenhousev1alpha1.Cluster{}
			id := types.NamespacedName{Name: "greenhouse-self", Namespace: test.TestNamespace}
			Eventually(func(g Gomega) bool {
				g.Expect(test.K8sClient.Get(test.Ctx, id, greenhouseCluster)).Should(Succeed(), "the cluster should have been created")
				g.Expect(greenhouseCluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeDirect), "the cluster accessmode should be set to direct")
				readyCondition := greenhouseCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
				return true
			}).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")
		})

		It("Should delete the cluster resource correctly", func() {
			By("Deleting the cluster resource")
			greenhouseCluster := &greenhousev1alpha1.Cluster{}
			Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "greenhouse-self", Namespace: test.TestNamespace}, greenhouseCluster)).Should(Succeed())
			id := types.NamespacedName{Name: "greenhouse-self", Namespace: test.TestNamespace}
			Expect(test.K8sClient.Delete(test.Ctx, greenhouseCluster, &client.DeleteOptions{})).Should(Succeed())

			By("Checking the resource is deleted")
			Eventually(func(g Gomega) bool {
				err := test.K8sClient.Get(test.Ctx, id, greenhouseCluster)
				g.Expect(err).To(HaveOccurred())
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				return true
			}).Should(BeTrue(), "getting the cluster should fail eventually")
		})
	})
})
