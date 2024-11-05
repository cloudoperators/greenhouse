package shared

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

const ManagedResourceName = "greenhouse"

func OnboardRemoteCluster(ctx context.Context, k8sClient client.Client, kubeConfigBytes []byte, name, namespace string) {
	By("applying remote cluster kubeconfig as greenhouse secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
		Data: map[string][]byte{
			greenhouseapis.KubeConfigKey: kubeConfigBytes,
		},
	}
	err := k8sClient.Create(ctx, secret)
	if apierrors.IsAlreadyExists(err) {
		err = k8sClient.Update(ctx, secret)
	}
	Expect(err).NotTo(HaveOccurred())
}

func OffBoardRemoteCluster(ctx context.Context, adminClient, remoteClient client.Client, testStartTime time.Time, name, namespace string) {
	cluster := &greenhousev1alpha1.Cluster{}
	err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
	Expect(err).NotTo(HaveOccurred())

	By("marking the cluster for deletion")
	err = triggerClusterDeletion(ctx, adminClient, cluster, testStartTime)
	Expect(err).NotTo(HaveOccurred())

	By("checking the cluster resource is eventually deleted")
	Eventually(func(g Gomega) bool {
		err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
		g.Expect(err).To(HaveOccurred())
		g.Expect(client.IgnoreNotFound(err)).To(Succeed())
		return true
	}).Should(BeTrue(), "cluster resource should be deleted")

	By("verifying that the remote cluster managed service account and cluster role binding is deleted")
	Eventually(func(g Gomega) bool {
		crb := &rbacv1.ClusterRoleBinding{}
		err := remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName}, crb)
		GinkgoWriter.Printf("crb err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "cluster role binding should be deleted")
		managedSA := &corev1.ServiceAccount{}
		err = remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName, Namespace: namespace}, managedSA)
		GinkgoWriter.Printf("sa err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "managed service account should be deleted")
		secret := &corev1.Secret{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret)
		GinkgoWriter.Printf("secret err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "cluster secret should be deleted")
		return true
	}).Should(BeTrue(), "managed service account should be deleted")

	By("verifying that the owned cluster secret is deleted")
	Eventually(func(g Gomega) bool {
		secret := &corev1.Secret{}
		err := adminClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret)
		GinkgoWriter.Printf("secret err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "cluster secret should be deleted")
		return true
	}).Should(BeTrue(), "managed service account should be deleted")
}

func ClusterIsReady(ctx context.Context, adminClient client.Client, clusterName, namespace string) {
	By("verifying if the cluster is in ready state")
	Eventually(func(g Gomega) bool {
		cluster := &greenhousev1alpha1.Cluster{}
		err := adminClient.Get(ctx, client.ObjectKey{Name: clusterName, Namespace: namespace}, cluster)
		g.Expect(err).ToNot(HaveOccurred())
		conditions := cluster.GetConditions()
		readyCondition := conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)
		g.Expect(readyCondition).ToNot(BeNil(), "cluster should have ready condition")
		g.Expect(readyCondition.IsTrue()).To(BeTrue(), "cluster should be ready")
		g.Expect(cluster.Status.KubernetesVersion).ToNot(BeEmpty(), "cluster should have kubernetes version")
		return true
	}).Should(BeTrue(), "cluster should be ready")
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
