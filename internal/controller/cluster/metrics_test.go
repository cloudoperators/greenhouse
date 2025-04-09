// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Metrics controller", Ordered, func() {
	var (
		remoteEnvTest *envtest.Environment
	)

	BeforeAll(func() {
		_, _, remoteEnvTest, _ = test.StartControlPlane("6888", false, false)
	})

	AfterAll(func() {
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})

	It("Should return metrics when reconciler is calling", func() {
		const clusterName = "test-cluster-a"
		setup := test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: setup.Namespace(),
			},
			Status: greenhousev1alpha1.ClusterStatus{
				KubernetesVersion:              "1.31.1",
				BearerTokenExpirationTimestamp: metav1.Time{Time: time.Now().Add(600 * time.Second)},
			},
		}

		updateMetrics(cluster)
		counterAfter := prometheusTest.ToFloat64(kubernetesVersionsGauge.WithLabelValues(cluster.Name, cluster.Namespace, cluster.Status.KubernetesVersion))
		Expect(counterAfter).To(BeEquivalentTo(1))
		tokenExpiry := prometheusTest.ToFloat64(secondsToTokenExpiryGauge.WithLabelValues(cluster.Name, cluster.Namespace))
		Expect(tokenExpiry).To(BeNumerically(">=", 595))
		Expect(tokenExpiry).To(BeNumerically("<=", 600))
	})
})
