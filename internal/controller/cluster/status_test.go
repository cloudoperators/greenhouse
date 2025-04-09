// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Cluster status", Ordered, func() {
	const (
		clusterStatusTestCase = "cluster-status"
	)

	var (
		validCluster       = greenhousev1alpha1.Cluster{}
		validClusterName   = "cluster-status-with-valid-kubeconfig"
		invalidCluster     *greenhousev1alpha1.Cluster
		invalidClusterName = "cluster-status-without-kubeconfig"

		remoteEnv        *envtest.Environment
		remoteClient     client.Client
		remoteKubeConfig []byte

		setup test.TestSetup
	)

	BeforeAll(func() {
		_, remoteClient, remoteEnv, remoteKubeConfig = test.StartControlPlane("6886", false, false)

		setup = *test.NewTestSetup(test.Ctx, test.K8sClient, clusterStatusTestCase)

		By("Creating a node resource in the remote cluster")
		node := corev1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: corev1.GroupName,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-node",
				Namespace: "",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionUnknown,
					},
				},
			},
		}
		Expect(remoteClient.Create(test.Ctx, &node, &client.CreateOptions{})).
			Should(Succeed(), "there should be no error creating the node")

		node2 := corev1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: corev1.GroupName,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-node-2",
				Namespace: "",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		Expect(remoteClient.Create(test.Ctx, &node2, &client.CreateOptions{})).
			Should(Succeed(), "there should be no error creating the node")

		node3 := corev1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: corev1.GroupName,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-node-3",
				Namespace: "",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		}
		Expect(remoteClient.Create(test.Ctx, &node3, &client.CreateOptions{})).
			Should(Succeed(), "there should be no error creating the node")

		By("Creating a Secret with a valid KubeConfig for the remote cluster")
		secret := setup.CreateSecret(test.Ctx, validClusterName,
			test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
			test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}))

		By("Checking the cluster resource has been created")
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: secret.Name, Namespace: setup.Namespace()}, &validCluster)
		}).Should(Succeed(), fmt.Sprintf("eventually the cluster %s should exist", secret.Name))

		By("Creating a cluster without a secret")
		invalidCluster = setup.CreateCluster(test.Ctx, invalidClusterName, test.WithAccessMode(greenhousev1alpha1.ClusterAccessModeDirect))
	})

	AfterAll(func() {
		test.MustDeleteCluster(test.Ctx, test.K8sClient, client.ObjectKeyFromObject(&validCluster))
		test.MustDeleteCluster(test.Ctx, test.K8sClient, client.ObjectKeyFromObject(invalidCluster))
		Expect(remoteEnv.Stop()).Should(Succeed(), "there should be no error stopping the remote environment")
	})

	It("should reconcile the status of a cluster", func() {
		By("checking cluster node status")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)).ToNot(BeNil(), "The AllNodesReady condition should be present")
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Status).To(Equal(metav1.ConditionFalse))
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Message).To(ContainSubstring("test-node not ready, test-node-3 not ready"))
			g.Expect(validCluster.Status.Nodes).ToNot(BeEmpty())
			g.Expect(validCluster.Status.Nodes["test-node"].Conditions).ToNot(BeEmpty())
			g.Expect(validCluster.Status.Nodes["test-node"].Ready).To(BeFalse())
			return true
		}).Should(BeTrue())

		By("updating the node ready condition")
		node := &corev1.Node{}
		Expect(remoteClient.Get(test.Ctx, types.NamespacedName{Name: "test-node"}, node)).
			Should(Succeed(), "there should be no error getting the first remote node")

		node.Status.Conditions = []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		}
		Expect(remoteClient.Status().Update(test.Ctx, node)).
			Should(Succeed(), "there should be no error updating the fist remote node")

		Expect(remoteClient.Get(test.Ctx, types.NamespacedName{Name: "test-node-3"}, node)).
			Should(Succeed(), "there should be no error getting the third remote node")

		node.Status.Conditions = []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		}
		Expect(remoteClient.Status().Update(test.Ctx, node)).
			Should(Succeed(), "there should be no error updating the third remote node")
		Eventually(func(g Gomega) bool {
			g.Expect(remoteClient.Get(test.Ctx, types.NamespacedName{Name: "test-node"}, node)).Should(Succeed(), "There should be no error getting the remote node")
			g.Expect(node.Status.Conditions[0].Type).To(Equal(corev1.NodeReady))
			g.Expect(node.Status.Conditions[0].Status).To(Equal(corev1.ConditionTrue))
			return true
		}).Should(BeTrue(), "we should see the condition change on the remote node")

		By("Triggering a cluster reconcile by adding a label to speed up things. Requeue interval is set to 2min")
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
		validCluster.SetLabels(map[string]string{"reconcile-me": "true"})
		Expect(test.K8sClient.Update(test.Ctx, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error updating the cluster resource")

		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)).ToNot(BeNil(), "The AllNodesReady condition should be present")
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Status).To(Equal(metav1.ConditionTrue), "The AllNodesReady condition should be true")
			g.Expect(validCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Message).To(BeEmpty())
			g.Expect(validCluster.Status.Nodes).ToNot(BeEmpty())
			g.Expect(validCluster.Status.Nodes["test-node"].Conditions).ToNot(BeEmpty())
			g.Expect(validCluster.Status.Nodes["test-node"].Ready).To(BeTrue())
			return true
		}).Should(BeTrue())

		By("checking cluster ready condition")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(validCluster.Status.StatusConditions).ToNot(BeNil())
			readyCondition := validCluster.Status.GetConditionByType(greenhouseapis.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "The ClusterReady condition should be present")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(validCluster.Status.KubernetesVersion).ToNot(BeNil())
			return true
		}).Should(BeTrue())

	})

	It("should reconcile the status of a cluster without a secret", func() {
		By("checking cluster conditions")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: invalidCluster.Name, Namespace: setup.Namespace()}, invalidCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(invalidCluster.Status.StatusConditions).ToNot(BeNil())
			kubeConfigValidCondition := invalidCluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
			g.Expect(kubeConfigValidCondition).ToNot(BeNil(), "The KubeConfigValid condition should be present")
			g.Expect(kubeConfigValidCondition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(kubeConfigValidCondition.Message).To(ContainSubstring("Secret \"" + invalidCluster.Name + "\" not found"))
			readyCondition := invalidCluster.Status.GetConditionByType(greenhouseapis.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "The ClusterReady condition should be present")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			return true
		}).Should(BeTrue())
	})

	It("should set the deletion condition when the cluster is marked for deletion", func() {
		By("marking the cluster for deletion")
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
		validCluster.SetAnnotations(map[string]string{
			greenhouseapis.MarkClusterDeletionAnnotation: "true",
		})
		Expect(test.K8sClient.Update(test.Ctx, &validCluster)).To(Succeed(), "there must be no error updating the object", "key", client.ObjectKeyFromObject(&validCluster))

		By("checking the deletion condition")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: validCluster.Name, Namespace: setup.Namespace()}, &validCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(validCluster.Status.GetConditionByType(greenhouseapis.DeleteCondition)).ToNot(BeNil(), "The Delete condition should be present")
			g.Expect(validCluster.Status.GetConditionByType(greenhouseapis.DeleteCondition).Reason).To(Equal(lifecycle.ScheduledDeletionReason))
			return true
		}).Should(BeTrue())
	})
})
