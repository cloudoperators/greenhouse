// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Bootstrap controller", Ordered, func() {
	const bootstrapTestCase = "bootstrap"
	var (
		remoteEnvTest    *envtest.Environment
		remoteKubeConfig []byte
		setup            *test.TestSetup
		team             *greenhousev1alpha1.Team
	)

	BeforeAll(func() {
		_, _, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6888", false, false)
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, bootstrapTestCase)
		setup.CreateOrganization(test.Ctx, setup.Namespace())
		team = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	})

	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})

	Context("When reconciling a kubeConfig secret", func() {
		It("Should correctly set cluster.Spec.AccessMode and cluster.Status with valid remote kubeconfig and if remote api server is reachable",
			func() {
				By("Creating a secret with a valid kubeconfig for a remote cluster")
				validKubeConfigSecret := setup.CreateSecret(test.Ctx, bootstrapTestCase,
					test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
					test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
					test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
					test.WithSecretAnnotations(map[string]string{lifecycle.PropagateLabelsAnnotation: greenhouseapis.LabelKeyOwnedBy}),
				)

				By("Checking the accessmode is set correctly")
				cluster := &greenhousev1alpha1.Cluster{}
				id := types.NamespacedName{Name: validKubeConfigSecret.Name, Namespace: setup.Namespace()}
				Eventually(func(g Gomega) bool {
					g.Expect(test.K8sClient.Get(test.Ctx, id, cluster)).Should(Succeed(), "the cluster should have been created")
					g.Expect(cluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeDirect), "the cluster accessmode should be set to direct")
					g.Expect(cluster.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, team.Name), "the owned-by label value should be propagated to the cluster from secret")
					readyCondition := cluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
					g.Expect(readyCondition).ToNot(BeNil())
					g.Expect(readyCondition.Type).To(Equal(greenhousemetav1alpha1.ReadyCondition))
					g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
					return true
				}).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")

				By("Deleting the valid cluster")
				test.MustDeleteCluster(test.Ctx, test.K8sClient, cluster)
			})

		It("Should correctly set cluster.Spec.AccessMode and cluster.Status with invalid remote kubeconfig",
			func() {
				By("Creating a secret with an invalid kubeconfig for a remote cluster")
				kubeConfigString := string(remoteKubeConfig)
				// invalidate host
				invalidKubeConfigString := strings.ReplaceAll(kubeConfigString, "127.0.0.1", "invalid.host")
				invalidKubeConfigSecret := setup.CreateSecret(test.Ctx, bootstrapTestCase+"-invalid",
					test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
					test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: []byte(invalidKubeConfigString)}),
					test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
					test.WithSecretAnnotations(map[string]string{lifecycle.PropagateLabelsAnnotation: greenhouseapis.LabelKeyOwnedBy}),
				)

				By("Checking the accessmode is set correctly")
				cluster := &greenhousev1alpha1.Cluster{}
				id := types.NamespacedName{Name: invalidKubeConfigSecret.Name, Namespace: setup.Namespace()}
				Eventually(func(g Gomega) bool {
					g.Expect(test.K8sClient.Get(test.Ctx, id, cluster)).Should(Succeed(), "the cluster should have been created")
					g.Expect(cluster.Spec.AccessMode).To(Equal(greenhousev1alpha1.ClusterAccessModeDirect), "the cluster accessmode should still be direct")
					g.Expect(cluster.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, team.Name), "the owned-by label value should be propagated to the cluster from secret")
					g.Expect(cluster.Status.Conditions).ToNot(BeNil(), "status conditions should be present")
					readyCondition := cluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
					g.Expect(readyCondition).ToNot(BeNil())
					g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse), "the ready condition should be set to false")
					return true
				}).Should(BeTrue(), "getting the cluster should succeed eventually and the cluster accessmode and status should be set correctly")

				By("Deleting the invalid cluster")
				test.MustDeleteCluster(test.Ctx, test.K8sClient, cluster)
			})
		It("Should successfully propagate labels from the kubeconfig secret to the cluster resource", func() {
			By("Creating a kubeconfig secret with labels")
			kubeConfigSecret := setup.CreateSecret(test.Ctx, bootstrapTestCase+"-label-propagation",
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}),
			)

			By("Checking if the cluster is created")
			cluster := &greenhousev1alpha1.Cluster{}
			id := types.NamespacedName{Name: kubeConfigSecret.Name, Namespace: setup.Namespace()}
			Eventually(func(g Gomega) bool {
				g.Expect(test.K8sClient.Get(test.Ctx, id, cluster)).Should(Succeed(), "the cluster should have been created")
				return true
			}).Should(BeTrue(), "getting the cluster should succeed eventually")

			By("Adding labels to the kubeconfig secret")
			setup.UpdateSecret(test.Ctx, kubeConfigSecret.GetName(),
				test.WithSecretAnnotations(map[string]string{
					lifecycle.PropagateLabelsAnnotation: "support_group, region, greenhouse.sap/owned-by",
				}),
				test.WithSecretLabels(map[string]string{
					"support_group": "foo",
					"region":        "bar",
					"test-label":    "test-value",
				}),
				test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			)

			By("Checking the labels are propagated to the cluster resource")
			Eventually(func(g Gomega) bool {
				g.Expect(test.K8sClient.Get(test.Ctx, id, cluster)).Should(Succeed(), "the cluster should have been created")
				g.Expect(cluster.Labels).To(HaveKeyWithValue(greenhouseapis.LabelKeyOwnedBy, team.Name), "the owned-by label value should be propagated to the cluster from secret")
				g.Expect(cluster.Labels).To(HaveKey("support_group"), "the cluster should have the support_group propagated label")
				g.Expect(cluster.Labels).To(HaveKey("region"), "the cluster should have the region propagated label")
				return true
			}).Should(BeTrue(), "cluster should have labels propagated")

			By("Deleting the kubeconfig secret and checking the cluster is deleted")
			test.MustDeleteCluster(test.Ctx, test.K8sClient, cluster)
		})
	})
})
