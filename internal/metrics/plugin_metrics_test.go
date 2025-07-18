// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/metrics"
)

var _ = Describe("Metrics controller", Ordered, func() {

	DescribeTable("update metrics", func(plugin *greenhouseapisv1alpha1.Plugin, expectedCounter string, result metrics.MetricResult, reason metrics.MetricReason) {
		registerMetrics()
		metrics.UpdatePluginReconcileTotalMetric(plugin, result, reason)

		err := prometheusTest.CollectAndCompare(metrics.PluginReconcileTotal, strings.NewReader(expectedCounter))
		Expect(err).ShouldNot(HaveOccurred())
	},
		Entry("empty plugin",
			&greenhouseapisv1alpha1.Plugin{},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="",namespace="",owned_by="",plugin="",pluginDefinition="",reason="",result="success"} 1
    		`,
			metrics.MetricResultSuccess,
			metrics.MetricReasonEmpty),
		Entry("success plugin with data",
			&greenhouseapisv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_success_plugin",
					Namespace: "test_organization",
					Labels: map[string]string{
						greenhouseapis.LabelKeyOwnedBy: "test_owner",
					},
				},
				Spec: greenhouseapisv1alpha1.PluginSpec{
					ClusterName:      "cluster-a",
					PluginDefinition: "test-plugin-definition",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="test_owner",plugin="test_success_plugin",pluginDefinition="test-plugin-definition",reason="",result="success"} 1
    		`,
			metrics.MetricResultSuccess,
			metrics.MetricReasonEmpty),
		Entry("error plugin with reconcile conditions",
			&greenhouseapisv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_error_plugin",
					Namespace: "test_organization",
				},
				Spec: greenhouseapisv1alpha1.PluginSpec{
					ClusterName: "cluster-a",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="",plugin="test_error_plugin",pluginDefinition="",reason="template_failed",result="error"} 1
    		`,
			metrics.MetricResultError,
			metrics.MetricReasonTemplateFailed),
		Entry("error plugin with drift conditions",
			&greenhouseapisv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_error_plugin",
					Namespace: "test_organization",
				},
				Spec: greenhouseapisv1alpha1.PluginSpec{
					ClusterName: "cluster-a",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",namespace="test_organization",owned_by="",plugin="test_error_plugin",pluginDefinition="",reason="diff_failed",result="error"} 1
    		`,
			metrics.MetricResultError,
			metrics.MetricReasonDiffFailed,
		),
	)
})

func registerMetrics() {
	crmetrics.Registry.Unregister(metrics.PluginReconcileTotal)
	metrics.PluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "namespace", "result", "reason", "owned_by"})
	crmetrics.Registry.MustRegister(metrics.PluginReconcileTotal)
}
