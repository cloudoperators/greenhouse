// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const ManagedResourceName = "greenhouse"

func OnboardRemoteCluster(ctx context.Context, k8sClient client.Client, kubeConfigBytes []byte, name, namespace, supportGroupTeamName string) {
	By("applying remote cluster kubeconfig as greenhouse secret")
	secret := test.NewSecret(name, namespace, test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
		test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: kubeConfigBytes}),
		test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, supportGroupTeamName),
		test.WithSecretAnnotations(map[string]string{lifecycle.PropagateLabelsAnnotation: greenhouseapis.LabelKeyOwnedBy}),
	)
	err := k8sClient.Create(ctx, secret)
	if apierrors.IsAlreadyExists(err) {
		err = k8sClient.Update(ctx, secret)
	}
	Expect(err).NotTo(HaveOccurred())
}

func OnboardRemoteOIDCCluster(ctx context.Context, k8sClient client.Client, caCert []byte, apiServerURL, name, namespace, supportGroupTeamName string) {
	By("applying remote cluster OIDC configuration as greenhouse secret")
	secret := test.NewSecret(name, namespace, test.WithSecretType(greenhouseapis.SecretTypeOIDCConfig),
		test.WithSecretData(map[string][]byte{greenhouseapis.SecretAPIServerCAKey: caCert}),
		test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, supportGroupTeamName),
		test.WithSecretAnnotations(map[string]string{
			greenhouseapis.SecretAPIServerURLAnnotation: apiServerURL,
			lifecycle.PropagateLabelsAnnotation:         greenhouseapis.LabelKeyOwnedBy,
		}),
	)
	err := k8sClient.Create(ctx, secret)
	if apierrors.IsAlreadyExists(err) {
		err = k8sClient.Update(ctx, secret)
	}
	Expect(err).NotTo(HaveOccurred())
}

func OffBoardRemoteCluster(ctx context.Context, adminClient, remoteClient client.Client, testStartTime time.Time, name, namespace string) {
	cluster := &greenhousev1alpha1.Cluster{}
	err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	By("marking the cluster for deletion")
	err = triggerClusterDeletion(ctx, adminClient, cluster, testStartTime)
	Expect(err).NotTo(HaveOccurred())

	By("checking the cluster resource is eventually deleted")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "cluster resource should be deleted")
		secret := &corev1.Secret{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret)
		GinkgoWriter.Printf("secret err: %v\n", err)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "cluster secret should be deleted")
	}).Should(Succeed(), "cluster resource & secret should be deleted")

	By("verifying that the remote cluster managed service account and cluster role binding is deleted")
	Eventually(func(g Gomega) {
		crb := &rbacv1.ClusterRoleBinding{}
		err := remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName}, crb)
		GinkgoWriter.Printf("crb err: %v\n", err)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "cluster role binding should be deleted")
		managedSA := &corev1.ServiceAccount{}
		err = remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName, Namespace: namespace}, managedSA)
		GinkgoWriter.Printf("sa err: %v\n", err)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "managed service account should be deleted")
	}).Should(Succeed(), "RBAC on remote cluster should be deleted")
}

func ClusterIsReady(ctx context.Context, adminClient client.Client, clusterName, namespace string) {
	By("verifying if the cluster is in ready state")
	Eventually(func(g Gomega) {
		cluster := &greenhousev1alpha1.Cluster{}
		err := adminClient.Get(ctx, client.ObjectKey{Name: clusterName, Namespace: namespace}, cluster)
		g.Expect(err).ToNot(HaveOccurred())
		conditions := cluster.GetConditions()
		readyCondition := conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(readyCondition).ToNot(BeNil(), "cluster should have ready condition")
		g.Expect(readyCondition.IsTrue()).To(BeTrue(), "cluster should be ready")
		g.Expect(cluster.Status.KubernetesVersion).ToNot(BeEmpty(), "cluster should have kubernetes version")
	}).Should(Succeed(), "cluster should be ready")
}

func triggerClusterDeletion(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster, testStartTime time.Time) error {
	schedule, err := clientutil.ParseDateTime(testStartTime)
	Expect(err).ToNot(HaveOccurred(), "there should be no error parsing the time")
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)
		if err != nil {
			return err
		}
		cluster.SetAnnotations(map[string]string{
			greenhouseapis.MarkClusterDeletionAnnotation:     "true",
			greenhouseapis.ScheduleClusterDeletionAnnotation: schedule.Format(time.DateTime),
		})
		return k8sClient.Update(ctx, cluster)
	})
}
