// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type (
	MetricResult string
)

const (
	MetricResultSuccess MetricResult = "success"
	MetricResultError   MetricResult = "error"
)

var (
	PluginReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_reconcile_total",
		},
		[]string{"pluginDefinition", "clusterName", "plugin", "organization", "result", "owned_by"})
	OwnedByLabelMissingGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_owned_by_label_missing",
			Help: "Indicates if the greenhouse.sap/owned-by label is missing or invalid",
		},
		[]string{"resource_kind", "resource_name", "organization"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(PluginReconcileTotal)
	crmetrics.Registry.MustRegister(OwnedByLabelMissingGauge)
}

func UpdatePluginReconcileTotalMetric(plugin *greenhousev1alpha1.Plugin, result MetricResult) {
	pluginReconcileTotalLabels := prometheus.Labels{
		"pluginDefinition": plugin.Spec.PluginDefinitionRef.Name,
		"clusterName":      plugin.Spec.ClusterName,
		"plugin":           plugin.Name,
		"organization":     plugin.Namespace,
		"result":           string(result),
		"owned_by":         plugin.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	PluginReconcileTotal.With(pluginReconcileTotalLabels).Inc()
}

func UpdateOwnedByLabelMissingMetric(resource lifecycle.RuntimeObject, isOwnerLabelMissing bool) {
	ownedByLabelMissingLabels := prometheus.Labels{
		"resource_kind": resource.GetObjectKind().GroupVersionKind().Kind,
		"resource_name": resource.GetName(),
		"organization":  resource.GetNamespace(),
	}
	if isOwnerLabelMissing {
		OwnedByLabelMissingGauge.With(ownedByLabelMissingLabels).Set(float64(1))
	} else {
		OwnedByLabelMissingGauge.With(ownedByLabelMissingLabels).Set(float64(0))
	}
}
