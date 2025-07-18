// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	chartTestRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_chart_test_runs_total",
			Help: "Total number of Helm Chart test runs with their results",
		},
		[]string{"cluster", "plugin", "namespace", "result", "owned_by"})
)

func init() {
	crmetrics.Registry.MustRegister(chartTestRunsTotal)
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
