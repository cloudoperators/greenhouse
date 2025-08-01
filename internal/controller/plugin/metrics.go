// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	pluginReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_plugin_ready",
			Help: "Indicates whether the plugin is ready",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "namespace", "owned_by"})
	chartTestRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_chart_test_runs_total",
			Help: "Total number of Helm Chart test runs with their results",
		},
		[]string{"cluster", "plugin", "namespace", "result", "owned_by"})
	workloadStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_plugin_workload_status_up",
			Help: "The workload status of the plugin",
		},
		[]string{"namespace", "plugin", "pluginDefinition", "cluster", "owned_by"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(pluginReady)
	crmetrics.Registry.MustRegister(chartTestRunsTotal)
	crmetrics.Registry.MustRegister(workloadStatusGauge)
}

func UpdatePluginReadyMetric(plugin *greenhousev1alpha1.Plugin, ready bool) {
	pluginReadyLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinition,
		"clusterName":      plugin.Spec.ClusterName,
		"plugin":           plugin.Name,
		"namespace":        plugin.Namespace,
		"owned_by":         plugin.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	if ready {
		pluginReady.With(pluginReadyLabels).Set(1)
	} else {
		pluginReady.With(pluginReadyLabels).Set(0)
	}
}

func IncrementHelmChartTestRunsTotal(plugin *greenhousev1alpha1.Plugin, testRunResult string) {
	prometheusLabels := prometheus.Labels{
		"cluster":   plugin.Spec.ClusterName,
		"plugin":    plugin.Name,
		"namespace": plugin.Namespace,
		"owned_by":  plugin.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
		"result":    testRunResult,
	}
	chartTestRunsTotal.With(prometheusLabels).Inc()
}

// setWorkloadMetrics sets the workload status metric to the given status
func UpdatePluginWorkloadMetrics(plugin *greenhousev1alpha1.Plugin, status float64) {
	workloadStatusGauge.WithLabelValues(
		plugin.GetNamespace(),
		plugin.Name,
		plugin.Spec.PluginDefinition,
		plugin.Spec.ClusterName,
		plugin.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
	).Set(status)
}
