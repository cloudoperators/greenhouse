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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	clusterpkg "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Reconciling a Headscale Cluster with mocked Headscale GRPC client and swapped client getter", Ordered, func() {
	const (
		headscaleClusterName      = "headscale-cluster"
		headscaleClusterNamespace = "headscale"
	)

	var (
		cluster = &greenhousev1alpha1.Cluster{}
		secret  = &corev1.Secret{}
	)

	BeforeAll(func() {
		// Mitigate https://book.kubebuilder.io/reference/envtest.html#:~:text=EnvTest%20does%20not%20support%20namespace,and%20never%20actually%20be%20reclaimed
		// CRBs and SAs are bound by owner-reference to the namespace which is never really deleted...
		Expect(test.K8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: headscaleClusterNamespace}})).NotTo(HaveOccurred(), "there should be no error creating the test namespace")
		Expect(remoteClient.DeleteAllOf(test.Ctx, &rbacv1.ClusterRoleBinding{})).To(Succeed(), "All CRBs should clean up before the tests")

		cluster = &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      headscaleClusterName,
				Namespace: headscaleClusterNamespace,
			},
			Spec: greenhousev1alpha1.ClusterSpec{
				AccessMode: greenhousev1alpha1.ClusterAccessModeHeadscale,
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, cluster)).Should(Succeed(), "creating a cluster should be successful")

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      headscaleClusterName,
				Namespace: headscaleClusterNamespace,
				Labels: map[string]string{
					"greenhouse/test": "headscale",
				},
			},
			Type: greenhouseapis.SecretTypeKubeConfig,
			Data: map[string][]byte{
				greenhouseapis.KubeConfigKey:           remoteKubeConfig,
				greenhouseapis.GreenHouseKubeConfigKey: remoteKubeConfig,
			},
		}
		Expect(test.K8sClient.Create(test.Ctx, secret)).Should(Succeed(), "creating a secret should be successful")

	})
	AfterAll(func() {
		test.MustDeleteCluster(test.Ctx, test.K8sClient, types.NamespacedName{Name: headscaleClusterName, Namespace: headscaleClusterNamespace})
		test.MustDeleteSecretWithLabel(test.Ctx, test.K8sClient, "headscale")
	})
	It("should reconcile headscale cluster", func() {
		By("Checking the Headscale Status is being set in the local cluster")
		getCluster := &greenhousev1alpha1.Cluster{}
		id := types.NamespacedName{Name: headscaleClusterName, Namespace: headscaleClusterNamespace}
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, id, getCluster)).To(Succeed(), "There should be no error getting the cluster")
			g.Expect(getCluster.Status.HeadScaleStatus).ToNot(BeNil(), "headscale status should be set")
			headscaleCondition := getCluster.Status.GetConditionByType(greenhousev1alpha1.HeadscaleReady)
			g.Expect(headscaleCondition).ToNot(BeNil(), "The HeadscaleReady condition should be present")
			g.Expect(headscaleCondition.Status).To(Equal(metav1.ConditionTrue), "The HeadscaleReady condition status should be true")
			return true
		}).
			Should(BeTrue(), "getting the cluster should succeed eventually and status should be set correctly")

		By("Checking the Namespace is created in the Remote Cluster")
		getNamespace := &corev1.Namespace{}
		id = types.NamespacedName{Name: headscaleClusterNamespace}
		Eventually(func(g Gomega) bool {
			g.Expect(remoteClient.Get(test.Ctx, id, getNamespace)).To(Succeed(), "There should be no error getting the remote namespace")
			g.Expect(getNamespace.GetName()).To(Equal(headscaleClusterNamespace), "The remote namespace name should be correct")
			g.Expect(getNamespace.Status.Phase).To(Equal(corev1.NamespaceActive), "The remote namespace should be active")
			return true
		}).Should(BeTrue(), "getting the namespace should succeed eventually")

		By("Checking the Service Account is created in the Remote Cluster")
		getServiceAccount := &corev1.ServiceAccount{}
		id = types.NamespacedName{Name: clusterpkg.ExportServiceAccountName, Namespace: headscaleClusterNamespace}
		Eventually(func(g Gomega) bool {
			g.Expect(remoteClient.Get(test.Ctx, id, getServiceAccount)).To(Succeed(), "There should be no error getting the remote service account")
			g.Expect(getServiceAccount.GetName()).To(Equal(clusterpkg.ExportServiceAccountName), "The SA name should be correct")
			g.Expect(getServiceAccount.Namespace).To(Equal(headscaleClusterNamespace), "The SA should be deployed to the correct namespace")
			return true
		}).Should(BeTrue(), "getting the service account should succeed eventually")

		By("Checking the Cluster Role Binding is created in the Remote Cluster")
		getClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		id = types.NamespacedName{Name: "greenhouse"}
		Eventually(func(g Gomega) bool {
			g.Expect(remoteClient.Get(test.Ctx, id, getClusterRoleBinding)).To(Succeed(), "There should be no error getting the remote crb")
			g.Expect(getClusterRoleBinding.RoleRef.Name).To(Equal("cluster-admin"), "crb should bind cluster-admin")
			g.Expect(getClusterRoleBinding.Subjects[0].Namespace).To(Equal(headscaleClusterNamespace), "crb should be deployed to correct namespace")
			g.Expect(getClusterRoleBinding.OwnerReferences[0].Name).To(Equal(headscaleClusterNamespace), "crb should have owner-reference to namespace")
			return true
		}).Should(BeTrue(), "getting the cluster role binding should succeed eventually")

		By("Checking the Service Account Token is updated in the Local Cluster")
		getSecret := &corev1.Secret{}
		id = types.NamespacedName{Name: headscaleClusterName, Namespace: headscaleClusterNamespace}
		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, id, getSecret)).To(Succeed(), "There should be no error getting the cluster secret")
			actConfig, ok := getSecret.Data[greenhouseapis.GreenHouseKubeConfigKey]
			if !ok {
				return false
			}
			g.Expect(strings.Contains(string(actConfig), tailscaleProxyURL)).To(BeTrue(), "The secret should contain the proxy url")
			return true
		}).Should(BeTrue(), "getting the secret should succeed eventually and the secret should contain the proxy url")

		By("Checking the Headscale PreAuthKey is set in the secret in the remote cluster")
		getSecret = &corev1.Secret{}
		id = types.NamespacedName{Name: "tailscale-auth", Namespace: headscaleClusterNamespace}
		Eventually(func(g Gomega) bool {
			g.Expect(remoteClient.Get(test.Ctx, id, getSecret)).To(Succeed(), "There should be no error getting the remote secret")
			actConfig, ok := getSecret.Data[clusterpkg.ExportTailscaleAuthorizationKey]
			if !ok {
				return false
			}
			g.Expect(actConfig).ToNot(BeNil(), "The secret should containt the preauthke")
			return true
		}).Should(BeTrue(), "getting the secret should succeed eventually and the secret should contain the preauthkey")

		By("Checking that an error is persisted in the headscaleReady condition message")
		// replace mock function with original client getter func as this client will fail
		grpcClient, err := clientutil.NewHeadscaleGRPCClient(headscaleReconciler.HeadscaleGRPCURL, headscaleReconciler.HeadscaleAPIKey)
		Expect(err).ToNot(HaveOccurred(), "There should be no error instantiating the original grpc client")
		clusterpkg.ExportSetHeadscaleGRPCClientOnHAR(headscaleReconciler, grpcClient)
		// trigger cluster reconcile
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: headscaleClusterName, Namespace: headscaleClusterNamespace}, getCluster)).Should(Succeed(), "There should be no error getting the cluster")
		getCluster.SetLabels(map[string]string{"reconcile-me": "true"})
		Expect(test.K8sClient.Update(test.Ctx, getCluster)).Should(Succeed(), "There should be no error updating the cluster")

		Eventually(func(g Gomega) bool {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: getCluster.Name, Namespace: headscaleClusterNamespace}, getCluster)).To(Succeed(), "There should be no error getting the cluster")
			g.Expect(getCluster.Status.HeadScaleStatus).ToNot(BeNil(), "headscale status should be set")
			headscaleCondition := getCluster.Status.GetConditionByType(greenhousev1alpha1.HeadscaleReady)
			g.Expect(headscaleCondition).ToNot(BeNil(), "The HeadscaleReady condition should be present")
			g.Expect(headscaleCondition.Status).To(Equal(metav1.ConditionFalse), "The HeadscaleReady condition status should be false")
			g.Expect(headscaleCondition.Message).To(ContainSubstring("no headscale machine found"), "The client error message should be reflected to the condition")

			// We are testing the part of the status controller depending on the headscale ready condition here!
			// All other test setups would expect separation of running controllers
			readyCondition := getCluster.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "The Ready condition should be present")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse), "The Ready condition status should be false")
			g.Expect(readyCondition.Message).To(ContainSubstring("Headscale connection not ready"), "The default headscale error message should be present")
			return true
		}).
			Should(BeTrue(), "getting the cluster should succeed eventually and status should be set correctly")

		/*
			This is commented as the access to the remote cluster requires a https proxy.
			Though the proxy is in-place, golang does not account for a proxy on localhost (1) and
			injecting custom transport in the client.Client is not supported when using TLS certificates (2).
			(1) https://maelvls.dev/go-ignores-proxy-localhost,
			(2) https://github.com/kubernetes/client-go/blob/master/transport/transport.go#L38-L40.

			By("Checking the Cluster Status contains the K8s Version in the local cluster")
			getCluster = &greenhousev1alpha1.Cluster{}
			id = types.NamespacedName{Name: headscaleClusterName, Namespace: orgName}
			Eventually(func() bool {
				err := test.K8sClient.Get(test.Ctx, id, getCluster)
				if err != nil {
					return false
				}
				return getCluster.Status.KubernetesVersion != ""
			}, updateTimeout, pollInterval).
				Should(BeTrue(), "getting the cluster should succeed eventually and the cluster kubernetes status should be set")
		*/
	})
})
