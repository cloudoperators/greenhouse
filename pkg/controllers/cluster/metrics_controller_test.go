package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Metrics controller", Ordered, func() {
	var (
		remoteClient     client.Client
		remoteEnvTest    *envtest.Environment
		remoteKubeConfig []byte
	)

	BeforeAll(func() {
		_, remoteClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6888", false, false)
	})

	AfterAll(func() {
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})

	It("Should return error when reconcile with an incorrect input object", func() {
		reconciler := ClusterMetricsReconciler{
			Client: remoteClient,
		}

		result, err := reconciler.Reconcile(test.Ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{},
		})
		Expect(err).To(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{}))
	})

	It("Should return metrics when reconciler is calling", func() {
		const clusterName = "test-cluster-a"
		setup := test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
		secret := setup.CreateSecret(test.Ctx, clusterName,
			test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
			test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}))
		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: setup.Namespace(),
			},
		}
		test.EventuallyCreated(test.Ctx, test.K8sClient, cluster)

		reconciler := ClusterMetricsReconciler{
			Client: test.K8sClient,
		}

		counterBefore := prometheusTest.CollectAndCount(kubernetesVersionsCounter)
		result, err := reconciler.Reconcile(test.Ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      secret.Name,
				Namespace: cluster.Namespace,
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{RequeueAfter: metricsRequeueInterval}))
		counterAfter := prometheusTest.CollectAndCount(kubernetesVersionsCounter)
		Expect(counterAfter).To(BeEquivalentTo(counterBefore + 1))
		tokenExpiry := prometheusTest.ToFloat64(secondsToTokenExpiryGauge)
		Expect(tokenExpiry).To(BeEquivalentTo(float64(599)))
	})
})
