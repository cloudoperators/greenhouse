// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package expect

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudoperators/greenhouse/e2e/shared"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func ClusterDeletionIsScheduled(ctx context.Context, adminClient client.Client, name, namespace string) {
	now := time.Now().UTC()
	cluster := &greenhousev1alpha1.Cluster{}
	objKey := client.ObjectKey{Name: name, Namespace: namespace}

	By("marking the cluster for deletion")
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := adminClient.Get(ctx, objKey, cluster)
		if err != nil {
			return err
		}
		return markClusterToBeDeleted(ctx, adminClient, cluster)
	})
	Expect(err).NotTo(HaveOccurred(), "there should be no error marking the cluster for deletion")

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

func RevokingRemoteServiceAccount(ctx context.Context, adminClient, remoteClient client.Client, serviceAccountName, clusterName, namespace string) {
	By("deleting the managed service account in the remote cluster")
	Eventually(func(g Gomega) bool {
		sa := &corev1.ServiceAccount{}
		err := remoteClient.Get(ctx, client.ObjectKey{Name: serviceAccountName, Namespace: namespace}, sa)
		if err != nil && apierrors.IsNotFound(err) {
			return true
		}
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the service account")
		err = remoteClient.Delete(ctx, sa)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error deleting the service account")
		return true
	}).Should(BeTrue(), "remote service account should be deleted")
	reconcileReadyNotReady(ctx, adminClient, clusterName, namespace, false)
}

func RestoreCluster(ctx context.Context, adminClient client.Client, clusterName, namespace string, kubeConfigBytes []byte) {
	By("deleting the current cluster secret and re-onboarding the remote cluster")
	Eventually(func(g Gomega) bool {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}
		err := adminClient.Delete(ctx, secret)
		g.Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred(), "there should be no error deleting the cluster secret")
		return true
	}).Should(BeTrue(), "cluster secret should be deleted")
	shared.OnboardRemoteCluster(ctx, adminClient, kubeConfigBytes, clusterName, namespace)
	err := shared.WaitUntilResourceReadyOrNotReady(ctx, adminClient, &greenhousev1alpha1.Cluster{}, clusterName, namespace, nil, true)
	Expect(err).NotTo(HaveOccurred(), "cluster should be ready")
}

func reconcileReadyNotReady(ctx context.Context, adminClient client.Client, clusterName, namespace string, readyStatus bool) {
	err := shared.WaitUntilResourceReadyOrNotReady(ctx, adminClient, &greenhousev1alpha1.Cluster{}, clusterName, namespace, func(resource lifecycle.RuntimeObject) error {
		By("triggering a reconcile of the cluster resource")
		resource.SetLabels(map[string]string{
			"greenhouse.sap/last-apply": strconv.FormatInt(time.Now().Unix(), 10),
		})
		return adminClient.Update(ctx, resource)
	}, readyStatus)
	Expect(err).NotTo(HaveOccurred(), "cluster should be in desired status")
}

func markClusterToBeDeleted(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	cluster.SetAnnotations(map[string]string{
		greenhouseapis.MarkClusterDeletionAnnotation: "true",
	})
	return k8sClient.Update(ctx, cluster)
}
