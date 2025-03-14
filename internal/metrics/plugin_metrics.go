// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	controllerMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/greenhouse/v1alpha1"
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
	pluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "organization", "result", "reason"})
)

func init() {
	controllerMetrics.Registry.MustRegister(pluginReconcileTotal)
}

func UpdateMetrics(plugin *greenhousev1alpha1.Plugin, result MetricResult, reason MetricReason) {
	pluginReconcileTotalLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinition,
		"clusterName":      plugin.Spec.ClusterName,
		"plugin":           plugin.Name,
		"organization":     plugin.Namespace,
		"result":           string(result),
		"reason":           string(reason),
	}
	pluginReconcileTotal.With(pluginReconcileTotalLabels).Inc()
}
