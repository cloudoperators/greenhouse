// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

var (
	OwnedByLabelMissingGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_owned_by_label_missing",
			Help: "Indicates if the greenhouse.sap/owned-by label is missing or invalid",
		},
		[]string{"resource_kind", "resource_name", "organization"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(OwnedByLabelMissingGauge)
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
