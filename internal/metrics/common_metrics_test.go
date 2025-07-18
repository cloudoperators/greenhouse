// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/internal/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Common Metrics", Ordered, func() {
	var (
		remoteEnvTest *envtest.Environment
	)

	BeforeAll(func() {
		_, _, remoteEnvTest, _ = test.StartControlPlane("6888", false, false)
	})

	AfterAll(func() {
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})

	It("Should update the OwnedByLabelMissingMetric when reconciler is calling", func() {
		const clusterName = "test-cluster-a"
		setup := test.NewTestSetup(test.Ctx, test.K8sClient, "commonmetrics")
		cluster := &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: setup.Namespace(),
			},
		}

		metrics.UpdateOwnedByLabelMissingMetric(cluster, true)

		counterAfter := prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(1))

		metrics.UpdateOwnedByLabelMissingMetric(cluster, false)

		counterAfter = prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(0))
	})
})
