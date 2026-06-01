// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var pluginPresetReady = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "greenhouse_pluginpreset_ready",
		Help: "Indicates whether the PluginPreset is ready",
	},
	[]string{"pluginPreset", "organization", "owned_by"},
)

var pluginPresetPluginsReconciled = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "greenhouse_pluginpreset_plugins_reconciled",
		Help: "Indicates whether the PluginPreset's Plugins are reconciled successfully",
	},
	[]string{"pluginPreset", "organization", "owned_by"},
)

func init() {
	crmetrics.Registry.MustRegister(pluginPresetReady)
	crmetrics.Registry.MustRegister(pluginPresetPluginsReconciled)
}

func updatePluginPresetMetrics(preset *greenhousev1alpha1.PluginPreset) {
	pluginPresetReady.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})
	pluginPresetPluginsReconciled.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})

	labels := prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
		"owned_by":     preset.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	if preset.Status.IsReadyTrue() {
		pluginPresetReady.With(labels).Set(1)
	} else {
		pluginPresetReady.With(labels).Set(0)
	}

	pluginFailed := preset.Status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
	if pluginFailed.Status == metav1.ConditionFalse {
		pluginPresetPluginsReconciled.With(labels).Set(1)
	} else {
		pluginPresetPluginsReconciled.With(labels).Set(0)
	}
}

func deletePluginPresetMetrics(preset *greenhousev1alpha1.PluginPreset) {
	pluginPresetReady.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})
	pluginPresetPluginsReconciled.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})
}
