package expect

import (
	"context"
	"fmt"
	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/e2e"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	RemoteClusterName   = "remote-int-cluster"
	ManagedResourceName = "greenhouse"
)

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

func ClusterDeletionIsScheduled(ctx context.Context, adminClient client.Client, name, namespace string) {
	now := time.Now().UTC()
	cluster := &greenhousev1alpha1.Cluster{}
	objKey := client.ObjectKey{Name: name, Namespace: namespace}

	By("marking the cluster for deletion")
	err := adminClient.Get(ctx, objKey, cluster)
	Expect(err).NotTo(HaveOccurred(), "there should be no error getting the cluster")
	err = markClusterToBeDeleted(ctx, adminClient, cluster)
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
	}, 1*time.Minute).Should(BeTrue(), "cluster should have a deletion schedule annotation")
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
		g.Expect(client.IgnoreNotFound(err)).To(BeNil())
		return true
	}, 1*time.Minute).Should(BeTrue(), "cluster resource should be deleted")

	By("verifying that the remote cluster managed service account and cluster role binding is deleted")
	Eventually(func(g Gomega) bool {
		crb := &rbacv1.ClusterRoleBinding{}
		err := remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName}, crb)
		GinkgoWriter.Printf("crb err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(BeNil(), "cluster role binding should be deleted")
		managedSA := &corev1.ServiceAccount{}
		err = remoteClient.Get(ctx, client.ObjectKey{Name: ManagedResourceName, Namespace: namespace}, managedSA)
		GinkgoWriter.Printf("sa err: %v\n", err)
		g.Expect(client.IgnoreNotFound(err)).To(BeNil(), "managed service account should be deleted")
		return true
	}, 1*time.Minute).Should(BeTrue(), "managed service account should be deleted")
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

func RevokingRemoteServiceAccount(ctx context.Context, adminClient client.Client, remoteClient client.Client, serviceAccountName, clusterName, namespace string) {
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
	OnboardRemoteCluster(ctx, adminClient, kubeConfigBytes, clusterName, namespace)
	err := e2e.WaitUntilResourceReadyOrNotReady(ctx, adminClient, &greenhousev1alpha1.Cluster{}, clusterName, namespace, nil, true)
	Expect(err).NotTo(HaveOccurred(), "cluster should be ready")
}

func IsOwner(owner, owned metav1.Object) bool {
	runtimeObj, ok := (owner).(runtime.Object)
	if !ok {
		return false
	}
	// ClusterRoleBinding does not have a type information (So We add it)
	if runtimeObj.GetObjectKind().GroupVersionKind() == (schema.GroupVersionKind{}) {
		if err := addTypeInformationToObject(runtimeObj); err != nil {
			GinkgoWriter.Printf("error adding type information to object: %s", err.Error())
			return false
		}
	}
	for _, ownerRef := range owned.GetOwnerReferences() {
		if ownerRef.Name == owner.GetName() && ownerRef.UID == owner.GetUID() && ownerRef.Kind == runtimeObj.GetObjectKind().GroupVersionKind().Kind {
			return true
		}
	}
	return false
}

func reconcileReadyNotReady(ctx context.Context, adminClient client.Client, clusterName, namespace string, readyStatus bool) {
	err := e2e.WaitUntilResourceReadyOrNotReady(ctx, adminClient, &greenhousev1alpha1.Cluster{}, clusterName, namespace, func(resource lifecycle.RuntimeObject) error {
		By("triggering a reconcile of the cluster resource")
		resource.SetLabels(map[string]string{
			"greenhouse.sap/last-apply": fmt.Sprintf("%d", time.Now().Unix()),
		})
		return adminClient.Update(ctx, resource)
	}, readyStatus)
	Expect(err).NotTo(HaveOccurred(), "cluster should be in desired status")
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

func markClusterToBeDeleted(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	cluster.SetAnnotations(map[string]string{
		greenhouseapis.MarkClusterDeletionAnnotation: "true",
	})
	return k8sClient.Update(ctx, cluster)
}

func addTypeInformationToObject(obj runtime.Object) error {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range gvks {
		if gvk.Kind == "" || gvk.Version == "" || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}
	return nil
}
