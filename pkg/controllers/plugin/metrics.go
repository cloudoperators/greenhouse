package plugin

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

const (
	metricResultSuccess = "success"
	metricResultError   = "error"

	metricReasonTemplateFailed = "template_failed"
	metricReasonDiffFailed     = "diff_failed"
	metricReasonUpgradeFailed  = "upgrade_failed"
)

var (
	pluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "organization", "result", "reason"})
)

func init() {
	metrics.Registry.MustRegister(pluginReconcileTotal)
}

func updateMetrics(plugin *greenhousev1alpha1.Plugin) {
	result := metricResultSuccess
	reason := ""

	for _, condition := range plugin.Status.Conditions {
		if condition.IsUnknown() {
			continue
		}

		switch condition.Type {
		case greenhousev1alpha1.HelmReconcileFailedCondition:
			if condition.IsTrue() {
				result = metricResultError
				reason = metricReasonTemplateFailed
			}
		case greenhousev1alpha1.HelmDriftDetectedCondition:
			if condition.IsTrue() {
				result = metricResultError
				reason = metricReasonDiffFailed
			}
		case greenhousev1alpha1.StatusUpToDateCondition:
			if condition.IsFalse() {
				result = metricResultError
				reason = metricReasonUpgradeFailed
			}
		}
	}

	pluginReconcileTotalLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinition,
		"clusterName":      plugin.Spec.ClusterName,
		"plugin":           plugin.Name,
		"organization":     plugin.Namespace,
		"result":           result,
		"reason":           reason,
	}
	pluginReconcileTotal.With(pluginReconcileTotalLabels).Inc()
}
