// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/util"
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

		util.UpdateOwnedByLabelMissingMetric(cluster, true)

		counterAfter := prometheusTest.ToFloat64(util.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(1))

		util.UpdateOwnedByLabelMissingMetric(cluster, false)

		counterAfter = prometheusTest.ToFloat64(util.OwnedByLabelMissingGauge.
			WithLabelValues(cluster.Kind, cluster.Name, cluster.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(0))
	})

	It("Should update the OwnedByLabelMissingMetric when Plugin reconciler is calling", func() {
		plugin := test.NewPlugin(test.Ctx, "test-plugin-a", setup.Namespace())

		util.UpdateOwnedByLabelMissingMetric(plugin, true)

		counterAfter := prometheusTest.ToFloat64(util.OwnedByLabelMissingGauge.
			WithLabelValues(plugin.Kind, plugin.Name, plugin.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(1))

		util.UpdateOwnedByLabelMissingMetric(plugin, false)

		counterAfter = prometheusTest.ToFloat64(util.OwnedByLabelMissingGauge.
			WithLabelValues(plugin.Kind, plugin.Name, plugin.Namespace))
		Expect(counterAfter).To(BeEquivalentTo(0))
	})
})

var _ = Describe("Shared Plugin metrics", Ordered, func() {
	DescribeTable("update metrics", func(plugin *greenhousev1alpha1.Plugin, expectedCounter string, result util.MetricResult, reason util.MetricReason) {
		registerPluginMetrics()
		util.UpdatePluginReconcileTotalMetric(plugin, result, reason)

		err := prometheusTest.CollectAndCompare(util.PluginReconcileTotal, strings.NewReader(expectedCounter))
		Expect(err).ShouldNot(HaveOccurred())
	},
		Entry("empty plugin",
			&greenhousev1alpha1.Plugin{},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="",namespace="",owned_by="",plugin="",pluginDefinition="",reason="",result="success"} 1
    		`,
			util.MetricResultSuccess,
			util.MetricReasonEmpty),
		Entry("success plugin with data",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_success_plugin",
					Namespace: "test_organization",
					Labels: map[string]string{
						greenhouseapis.LabelKeyOwnedBy: "test_owner",
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName:      "cluster-a",
					PluginDefinition: "test-plugin-definition",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="test_owner",plugin="test_success_plugin",pluginDefinition="test-plugin-definition",reason="",result="success"} 1
    		`,
			util.MetricResultSuccess,
			util.MetricReasonEmpty),
		Entry("error plugin with reconcile conditions",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_error_plugin",
					Namespace: "test_organization",
				},
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: "cluster-a",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="",plugin="test_error_plugin",pluginDefinition="",reason="template_failed",result="error"} 1
    		`,
			util.MetricResultError,
			util.MetricReasonTemplateFailed),
		Entry("error plugin with drift conditions",
			&greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_error_plugin",
					Namespace: "test_organization",
				},
				Spec: greenhousev1alpha1.PluginSpec{
					ClusterName: "cluster-a",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="",plugin="test_error_plugin",pluginDefinition="",reason="diff_failed",result="error"} 1
    		`,
			util.MetricResultError,
			util.MetricReasonDiffFailed,
		),
	)
})

func registerPluginMetrics() {
	crmetrics.Registry.Unregister(util.PluginReconcileTotal)
	util.PluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "namespace", "result", "reason", "owned_by"})
	crmetrics.Registry.MustRegister(util.PluginReconcileTotal)
}
