// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/internal/metrics"

	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Common Metrics", Ordered, func() {
	var (
		remoteEnvTest *envtest.Environment
		setup         *test.TestSetup
	)

	BeforeAll(func() {
		_, _, remoteEnvTest, _ = test.StartControlPlane("6888", false, false)
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "commonmetrics")
	})

	AfterAll(func() {
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})

	It("Should update the OwnedByLabelMissingMetric when Cluster reconciler is calling", func() {
		cluster := test.NewCluster(test.Ctx, "test-cluster-a", setup.Namespace())

		metrics.UpdateOwnedByLabelMissingMetric(cluster, true)

		counterAfter := prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(1))

		metrics.UpdateOwnedByLabelMissingMetric(cluster, false)

		counterAfter = prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(0))
	})

	It("Should update the OwnedByLabelMissingMetric when Plugin reconciler is calling", func() {
		plugin := test.NewPlugin(test.Ctx, "test-plugin-a", setup.Namespace())

		metrics.UpdateOwnedByLabelMissingMetric(plugin, true)

		counterAfter := prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(plugin.Kind, plugin.Name, plugin.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(1))

		metrics.UpdateOwnedByLabelMissingMetric(plugin, false)

		counterAfter = prometheusTest.ToFloat64(metrics.OwnedByLabelMissingGauge.
			WithLabelValues(plugin.Kind, plugin.Name, plugin.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(0))
	})
})
