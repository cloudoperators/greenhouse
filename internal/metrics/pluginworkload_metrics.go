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
	workloadStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_plugin_workload_status_up",
			Help: "The workload status of the plugin",
		},
		[]string{"namespace", "plugin", "pluginDefinition", "cluster", "owned_by"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(workloadStatusGauge)
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
