// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var pluginPresetReady = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "greenhouse_pluginpreset_ready",
		Help: "Indicates whether the pluginpreset is ready",
	},
	[]string{"pluginPreset", "organization", "owned_by"},
)

func init() {
	crmetrics.Registry.MustRegister(pluginPresetReady)
}

func updatePluginPresetReadyMetric(preset *greenhousev1alpha1.PluginPreset) {
	pluginPresetReady.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})
	labels := prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
		"owned_by":     preset.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	readyCondition := preset.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyCondition != nil && readyCondition.Status == metav1.ConditionTrue {
		pluginPresetReady.With(labels).Set(1)
	} else {
		pluginPresetReady.With(labels).Set(0)
	}
}

func deletePluginPresetReadyMetric(preset *greenhousev1alpha1.PluginPreset) {
	pluginPresetReady.DeletePartialMatch(prometheus.Labels{
		"pluginPreset": preset.Name,
		"organization": preset.Namespace,
	})
}
