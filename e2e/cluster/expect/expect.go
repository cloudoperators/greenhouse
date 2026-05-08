// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package expect

import (
	"context"
	"fmt"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/rest"

	"github.com/cloudoperators/greenhouse/e2e/shared"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func SetupOIDCClusterRoleBinding(ctx context.Context, remoteClient client.Client, clusterRoleBindingName, clusterName, namespace string) {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     fmt.Sprintf("greenhouse:system:serviceaccount:%s:%s", namespace, clusterName),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
	err := remoteClient.Create(ctx, crb)
	if apierrors.IsAlreadyExists(err) {
		err = remoteClient.Update(ctx, crb)
	}
	Expect(err).NotTo(HaveOccurred(), "there should be no error creating the oidc cluster role binding")
}

func VerifyClusterVersion(ctx context.Context, adminClient client.Client, remoteRestClient *clientutil.RestClientGetter, name, namespace string) {
	cluster := &greenhousev1alpha1.Cluster{}
	err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
	Expect(err).NotTo(HaveOccurred(), "there should be no error getting the cluster")
	statusKubeVersion := cluster.Status.KubernetesVersion
	dc, err := remoteRestClient.ToDiscoveryClient()
	Expect(err).NotTo(HaveOccurred(), "there should be no error creating the discovery client")
	expectedKubeVersion, err := dc.ServerVersion()
	Expect(err).NotTo(HaveOccurred(), "there should be no error getting the server version")
	Expect(statusKubeVersion).To(Equal(expectedKubeVersion.String()))
}

func ClusterDeletionIsScheduled(ctx context.Context, adminClient client.Client, name, namespace string) {
	now := time.Now().UTC()
	cluster := &greenhousev1alpha1.Cluster{}
	cluster.Name = name
	cluster.Namespace = namespace
	objKey := client.ObjectKeyFromObject(cluster)

	By("marking the cluster for deletion")
	test.MustSetAnnotation(ctx, adminClient, cluster, greenhouseapis.MarkClusterDeletionAnnotation, "true")

	Eventually(func(g Gomega) bool {
		cluster := &greenhousev1alpha1.Cluster{}
		err := adminClient.Get(ctx, objKey, cluster)
		g.Expect(err).ToNot(HaveOccurred())
		annotations := cluster.GetAnnotations()
		ok, schedule, err := clientutil.ExtractDeletionSchedule(annotations)
		g.Expect(err).ToNot(HaveOccurred(), "there should be no error extracting the deletion schedule")
		g.Expect(ok).To(BeTrue(), "cluster should be marked for deletion")
		diff := schedule.Sub(now).Hours()
		GinkgoWriter.Printf("diff: %f\n", diff)
		g.Expect(diff).To(BeNumerically("~", 48, 0.04), "deletion schedule should be within 1 hour")
		return true
	}).Should(BeTrue(), "cluster should have a deletion schedule annotation")
}

func RevokingRemoteClusterAccess(ctx context.Context, adminClient, remoteClient client.Client, serviceAccountName, clusterName, namespace string) {
	By("replacing the kubeconfig key data with the greenhouse kubeconfig key data")
	Eventually(func(g Gomega) bool {
		secret := &corev1.Secret{}
		err := adminClient.Get(ctx, client.ObjectKey{Name: clusterName, Namespace: namespace}, secret)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the cluster secret")
		g.Expect(clientutil.IsSecretContainsKey(secret, greenhouseapis.GreenHouseKubeConfigKey)).To(BeTrue(), "secret should contain the greenhouse kubeconfig key")
		secret.Data[greenhouseapis.KubeConfigKey] = secret.Data[greenhouseapis.GreenHouseKubeConfigKey]
		err = adminClient.Update(ctx, secret)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error updating the cluster secret")
		return true
	}).Should(BeTrue(), "kubeconfig key data should be replaced")

	By("deleting the cluster role binding in the remote cluster")
	Eventually(func(g Gomega) bool {
		crb := &rbacv1.ClusterRoleBinding{}
		err := remoteClient.Get(ctx, client.ObjectKey{Name: serviceAccountName}, crb)
		if err != nil && apierrors.IsNotFound(err) {
			return true
		}
		err = remoteClient.Delete(ctx, crb)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error deleting the cluster role binding")
		sa := &corev1.ServiceAccount{}
		err = remoteClient.Get(ctx, client.ObjectKey{Name: serviceAccountName, Namespace: namespace}, sa)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "service account should be deleted")
		return true
	}).Should(BeTrue(), "remote service account should be deleted")
	ReconcileReadyNotReady(ctx, adminClient, clusterName, namespace, false)
}

func ReconcileReadyNotReady(ctx context.Context, adminClient client.Client, clusterName, namespace string, readyStatus bool) {
	err := shared.WaitUntilResourceReadyOrNotReady(ctx, adminClient, &greenhousev1alpha1.Cluster{}, clusterName, namespace, func(resource lifecycle.RuntimeObject) error {
		By("triggering a reconcile of the cluster resource")
		resourceAnnotations := resource.GetAnnotations()
		if resourceAnnotations == nil {
			resourceAnnotations = make(map[string]string)
		}
		resourceAnnotations[lifecycle.ReconcileAnnotation] = metav1.Now().Format(time.DateTime)
		resource.SetAnnotations(resourceAnnotations)
		return adminClient.Update(ctx, resource)
	}, readyStatus)
	Expect(err).NotTo(HaveOccurred(), "cluster should be in desired status")
}

func ReconcileAndWaitForPayloadSchedulable(ctx context.Context, adminClient client.Client, clusterName, namespace string, schedulable bool) {
	Eventually(func(g Gomega) {
		cluster := &greenhousev1alpha1.Cluster{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: clusterName, Namespace: namespace}, cluster)).To(Succeed())
		By("triggering a reconcile of the cluster resource")
		resourceAnnotations := cluster.GetAnnotations()
		if resourceAnnotations == nil {
			resourceAnnotations = make(map[string]string)
		}
		resourceAnnotations[lifecycle.ReconcileAnnotation] = metav1.Now().Format(time.DateTime)
		cluster.SetAnnotations(resourceAnnotations)
		g.Expect(adminClient.Update(ctx, cluster)).To(Succeed())
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: clusterName, Namespace: namespace}, cluster)).To(Succeed())
		conditions := cluster.GetConditions()
		payloadCondition := conditions.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
		g.Expect(payloadCondition).ToNot(BeNil(), "cluster should have PayloadSchedulable condition")
		g.Expect(payloadCondition.IsTrue()).To(Equal(schedulable), "PayloadSchedulable should be %v", schedulable)
	}).Should(Succeed(), "PayloadSchedulable condition should reach desired state")
}

func GetRestConfig(restClientGetter *clientutil.RestClientGetter) *rest.Config {
	restConfig, err := restClientGetter.ToRESTConfig()
	Expect(err).NotTo(HaveOccurred(), "there should be no error creating the remote REST config")
	return restConfig
}

func CordonRemoteNodes(ctx context.Context, remoteClient client.Client) {
	nodes := &corev1.NodeList{}
	err := remoteClient.List(ctx, nodes)
	Expect(err).NotTo(HaveOccurred(), "there should be no error while listing nodes")
	for _, node := range nodes.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			By("skipping cordoning control plane node: " + node.Name)
			continue
		}
		if node.Spec.Unschedulable {
			continue
		}
		base := node.DeepCopy()
		node.Spec.Unschedulable = true
		err = remoteClient.Patch(ctx, &node, client.MergeFrom(base))
		Expect(err).NotTo(HaveOccurred(), "there should be no error while patching node")
	}
}

func UnCordonRemoteNodes(ctx context.Context, remoteClient client.Client) {
	nodes := &corev1.NodeList{}
	err := remoteClient.List(ctx, nodes)
	Expect(err).NotTo(HaveOccurred(), "there should be no error while listing nodes")
	for _, node := range nodes.Items {
		if !node.Spec.Unschedulable {
			continue
		}
		base := node.DeepCopy()
		node.Spec.Unschedulable = false
		err = remoteClient.Patch(ctx, &node, client.MergeFrom(base))
		Expect(err).NotTo(HaveOccurred(), "there should be no error while patching node")
	}
}
