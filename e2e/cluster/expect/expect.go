package expect

import (
	"context"
	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func OnboardRemoteCluster(ctx context.Context, k8sClient client.Client, kubeConfigBytes []byte, name, namespace string) {
	By("applying remote cluster kubeconfig as greenhouse secret")
	secret := &v1.Secret{
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
	Expect(err).NotTo(HaveOccurred())
}

func ClusterResourceIsReady(ctx context.Context, k8sClient client.Client, name, namespace string) {
	By("checking the cluster status is ready")
	cluster := &greenhousev1alpha1.Cluster{}
	Eventually(func(g Gomega) bool {
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cluster)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(cluster.Status.StatusConditions).ToNot(BeNil())
		g.Expect(cluster.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)).NotTo(BeNil())
		g.Expect(cluster.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ReadyCondition).Status).To(Equal(v1.ConditionTrue))
		g.Expect(cluster.Status.KubernetesVersion).ToNot(BeEmpty())
		return true
	}).Should(Succeed(), "cluster resource should be ready")
}
