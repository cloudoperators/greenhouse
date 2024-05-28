// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

var _ = Describe("OnboardSelf", Ordered, func() {
	Context("When onboarding itself as a cluster resource", func() {

		namespacedName := types.NamespacedName{Name: "greenhouse-self", Namespace: centralClusterNamespace}

		It("Should create a cluster resource for itself", func(ctx context.Context) {
			By("Creating a secret with a valid kubeconfig for a remote cluster")
			validKubeConfigSecret := corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: corev1.GroupName,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "greenhouse-self",
					Namespace: centralClusterNamespace,
				},
				Data: map[string][]byte{
					greenhouseapis.KubeConfigKey: centralClusterKubeconfigData,
				},
				Type: greenhouseapis.SecretTypeKubeConfig,
			}
			Expect(centralClusterClient.Create(ctx, &validKubeConfigSecret, &client.CreateOptions{})).Should(Succeed())

			By("Checking the resource is created")
			Eventually(func() error {
				greenhouseCluster := &greenhousev1alpha1.Cluster{}
				return centralClusterClient.Get(ctx, namespacedName, greenhouseCluster)
			}, timeout, interval).ShouldNot(HaveOccurred(), "the cluster should have been created")

			By("Checking the status ready")
			Eventually(func() bool {
				greenhouseCluster := &greenhousev1alpha1.Cluster{}
				Expect(centralClusterClient.Get(ctx, namespacedName, greenhouseCluster)).Should(Succeed(), "the cluster should have been created")
				Expect(greenhouseCluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeDirect), "the cluster accessmode should be set to direct")
				readyCondition := greenhouseCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				Expect(readyCondition).ToNot(BeNil())
				Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
				return true
			}, timeout, interval).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")
		})

		It("Should delete the cluster resource correctly", func(ctx context.Context) {
			By("Deleting the cluster resource")
			greenhouseCluster := &greenhousev1alpha1.Cluster{}
			Expect(centralClusterClient.Get(ctx, namespacedName, greenhouseCluster)).Should(Succeed())
			Expect(centralClusterClient.Delete(ctx, greenhouseCluster)).Should(Succeed())

			By("Checking the resource is deleted")
			Eventually(func() bool {
				err := centralClusterClient.Get(ctx, namespacedName, greenhouseCluster)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue(), "getting the cluster should fail eventually")

		})
	})
})
