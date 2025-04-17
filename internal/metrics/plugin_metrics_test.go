// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("Metrics controller", Ordered, func() {

	DescribeTable("update metrics", func(plugin *greenhouseapisv1alpha1.Plugin, expectedCounter string, result MetricResult, reason MetricReason) {
		registerMetrics()
		UpdateMetrics(plugin, result, reason)

		err := prometheusTest.CollectAndCompare(pluginReconcileTotal, strings.NewReader(expectedCounter))
		Expect(err).ShouldNot(HaveOccurred())
	},
		Entry("empty plugin",
			&greenhouseapisv1alpha1.Plugin{},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="",organization="",plugin="",pluginDefinition="",reason="",result="success"} 1
    		`,
			MetricResultSuccess,
			MetricReasonEmpty),
		Entry("success plugin with data",
			&greenhouseapisv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_success_plugin",
					Namespace: "test_organization",
				},
				Spec: greenhouseapisv1alpha1.PluginSpec{
					ClusterName:      "cluster-a",
					PluginDefinition: "test-plugin-definition",
				},
			},
			`
        	# HELP greenhouse_plugin_reconcile_total 
      		# TYPE greenhouse_plugin_reconcile_total counter
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",organization="test_organization",plugin="test_success_plugin",pluginDefinition="test-plugin-definition",reason="",result="success"} 1
    		`,
			MetricResultSuccess,
			MetricReasonEmpty),
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
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",organization="test_organization",plugin="test_error_plugin",pluginDefinition="",reason="template_failed",result="error"} 1
    		`,
			MetricResultError,
			MetricReasonTemplateFailed),
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
      		greenhouse_plugin_reconcile_total{clusterName="cluster-a",organization="test_organization",plugin="test_error_plugin",pluginDefinition="",reason="diff_failed",result="error"} 1
    		`,
			MetricResultError,
			MetricReasonDiffFailed,
		),
	)
})

func registerMetrics() {
	metrics.Registry.Unregister(pluginReconcileTotal)
	pluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "organization", "result", "reason"})
	metrics.Registry.MustRegister(pluginReconcileTotal)
}
