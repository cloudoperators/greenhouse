// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

type (
	MetricResult string
	MetricReason string
)

const (
	MetricResultSuccess MetricResult = "success"
	MetricResultError   MetricResult = "error"

	MetricReasonEmpty                    MetricReason = ""
	MetricReasonClusterAccessFailed      MetricReason = "cluster_access_failed"
	MetricReasonUninstallHelmFailed      MetricReason = "uninstall_helm_failed"
	MetricReasonInstallFailed            MetricReason = "upgrade_failed"
	MetricReasonUpgradeFailed            MetricReason = "upgrade_failed"
	MetricReasonPluginDefinitionNotFound MetricReason = "plugin_definition_not_found"
	MetricReasonTemplateFailed           MetricReason = "template_failed"
	MetricReasonDiffFailed               MetricReason = "diff_failed"
	MetricReasonHelmChartIsNotDefined    MetricReason = "helm_chart_is_not_defined"
)

var (
	PluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "namespace", "result", "reason", "owned_by"})

	pluginReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_plugin_ready",
			Help: "Indicates whether the plugin is ready",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "namespace", "owned_by"})
)

func init() {
	crmetrics.Registry.MustRegister(PluginReconcileTotal)
}

func UpdatePluginReconcileTotalMetric(plugin *greenhousev1alpha1.Plugin, result MetricResult, reason MetricReason) {
	pluginReconcileTotalLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinition,
		"clusterName":      plugin.Spec.ClusterName,
		"plugin":           plugin.Name,
		"namespace":        plugin.Namespace,
		"result":           string(result),
		"reason":           string(reason),
		"owned_by":         plugin.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	PluginReconcileTotal.With(pluginReconcileTotalLabels).Inc()
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
