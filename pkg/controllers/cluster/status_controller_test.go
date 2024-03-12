// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	statusClusterName          = "test-cluster-status"
	statusClusterName2         = "test-cluster-status-2"
	nodeStatusClusterNamespace = "cluster-status"
)

var statusCluster = greenhousev1alpha1.Cluster{}

var _ = Describe("Cluster status controller", Ordered, func() {
	BeforeAll(func() {
		Expect(test.K8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nodeStatusClusterNamespace}})).NotTo(HaveOccurred(), "there should be no error creating the test namespace")
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

		By("Creating a secret with a valid kubeconfig for a remote cluster")
		validKubeConfigSecret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: corev1.GroupName,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      statusClusterName,
				Namespace: nodeStatusClusterNamespace,
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

		By("Checking the cluster resource has been created")
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName, Namespace: nodeStatusClusterNamespace}, &statusCluster)
		}).Should(Succeed(), fmt.Sprintf("eventually the cluster %s should exist", statusClusterName))

		By("Creating a cluster without a secret")
		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      statusClusterName2,
				Namespace: nodeStatusClusterNamespace,
			},
			Spec: greenhousev1alpha1.ClusterSpec{
				AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, cluster, &client.CreateOptions{})).Should(Succeed(), "there should be no error creating the cluster")

	})

	It("should reconcile the status of a cluster", func() {
		By("checking cluster ready condition")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName, Namespace: nodeStatusClusterNamespace}, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(statusCluster.Status.StatusConditions).ToNot(BeNil())
			readyCondition := statusCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "The ClusterReady condition should be present")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCluster.Status.KubernetesVersion).ToNot(BeNil())
			return true
		}).Should(BeTrue())

		By("checking cluster node status")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName, Namespace: nodeStatusClusterNamespace}, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)).ToNot(BeNil(), "The AllNodesReady condition should be present")
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Status).To(Equal(metav1.ConditionFalse))
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Message).To(ContainSubstring("test-node not ready, test-node-3 not ready"))
			g.Expect(statusCluster.Status.Nodes).ToNot(BeEmpty())
			g.Expect(statusCluster.Status.Nodes["test-node"].Conditions).ToNot(BeEmpty())
			g.Expect(statusCluster.Status.Nodes["test-node"].Ready).To(BeFalse())
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
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName, Namespace: nodeStatusClusterNamespace}, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
		statusCluster.SetLabels(map[string]string{"reconcile-me": "true"})
		Expect(test.K8sClient.Update(test.Ctx, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error updating the cluster resource")

		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName, Namespace: nodeStatusClusterNamespace}, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)).ToNot(BeNil(), "The AllNodesReady condition should be present")
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Status).To(Equal(metav1.ConditionTrue), "The AllNodesReady condition should be true")
			g.Expect(statusCluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady).Message).To(BeEmpty())
			g.Expect(statusCluster.Status.Nodes).ToNot(BeEmpty())
			g.Expect(statusCluster.Status.Nodes["test-node"].Conditions).ToNot(BeEmpty())
			g.Expect(statusCluster.Status.Nodes["test-node"].Ready).To(BeTrue())
			return true
		}).Should(BeTrue())

	})

	It("should reconcile the status of a cluster without a secret", func() {
		By("checking cluster conditions")
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: statusClusterName2, Namespace: nodeStatusClusterNamespace}, &statusCluster)).ShouldNot(HaveOccurred(), "There should be no error getting the cluster resource")
			g.Expect(statusCluster.Status.StatusConditions).ToNot(BeNil())
			kubeConfigValidCondition := statusCluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
			g.Expect(kubeConfigValidCondition).ToNot(BeNil(), "The KubeConfigValid condition should be present")
			g.Expect(kubeConfigValidCondition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(kubeConfigValidCondition.Message).To(ContainSubstring("Secret \"test-cluster-status-2\" not found"))
			readyCondition := statusCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "The ClusterReady condition should be present")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(readyCondition.Message).To(ContainSubstring("kubeconfig not valid - cannot access cluster"))
			return true
		}).Should(BeTrue())

	})
})
