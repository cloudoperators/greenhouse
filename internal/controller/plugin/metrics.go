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
)

func init() {
	crmetrics.Registry.MustRegister(pluginReady)
}

func UpdatePluginReadyMetric(plugin *greenhousev1alpha1.Plugin, ready bool) {
	pluginReadyLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinitionRef.Name,
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
