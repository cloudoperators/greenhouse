// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Cluster Metrics", Ordered, func() {
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
		setup := test.NewTestSetup(test.Ctx, test.K8sClient, "clustermetrics")
		c := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: setup.Namespace(),
				Labels: map[string]string{
					greenhouseapis.LabelKeyOwnedBy: "test-owner",
				},
			},
			Status: greenhousev1alpha1.ClusterStatus{
				KubernetesVersion:              "1.31.1",
				BearerTokenExpirationTimestamp: metav1.Time{Time: time.Now().Add(600 * time.Second)},
			},
		}

		cluster.UpdateClusterMetrics(c)
		counterAfter := prometheusTest.ToFloat64(cluster.KubernetesVersionsGauge.WithLabelValues(c.Name, c.Namespace, c.Status.KubernetesVersion, "test-owner"))
		Expect(counterAfter).To(BeEquivalentTo(1))
		tokenExpiry := prometheusTest.ToFloat64(cluster.SecondsToTokenExpiryGauge.WithLabelValues(c.Name, c.Namespace, "test-owner"))
		Expect(tokenExpiry).To(BeNumerically(">=", 595))
		Expect(tokenExpiry).To(BeNumerically("<=", 600))
		readyGauge := prometheusTest.ToFloat64(cluster.ClusterReadyGauge.WithLabelValues(c.Name, c.Namespace, "test-owner"))
		Expect(readyGauge).To(BeEquivalentTo(float64(0)), "clusterReady metric should be present and the cluster should not be ready")
	})
})
