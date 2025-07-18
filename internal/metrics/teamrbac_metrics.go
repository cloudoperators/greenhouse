// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

var (
	teamRBACReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_rbac_ready",
			Help: "Indicates whether the team RBAC is ready",
		},
		[]string{"team_role_binding", "team", "namespace", "owned_by"},
	)
	ownedByLabelMissingGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_owned_by_label_missing",
			Help: "Indicates if the greenhouse.sap/owned-by label is missing or invalid",
		},
		[]string{"resource_kind", "resource_name", "organization"},
	)
)

func init() {
	prometheus.MustRegister(teamRBACReadyGauge)
	prometheus.MustRegister(ownedByLabelMissingGauge)
}

func UpdateTeamrbacMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyLabels := prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"team":              teamRoleBinding.Spec.TeamRef,
		"namespace":         teamRoleBinding.Namespace,
		"owned_by":          teamRoleBinding.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
	}
	if teamRoleBinding.Status.IsReadyTrue() {
		teamRBACReadyGauge.With(teamRBACReadyLabels).Set(1)
	} else {
		teamRBACReadyGauge.With(teamRBACReadyLabels).Set(0)
	}

	ownedByLabelMissingLabels := prometheus.Labels{
		"resource_kind": teamRoleBinding.Kind,
		"resource_name": teamRoleBinding.Name,
		"organization":  teamRoleBinding.Namespace,
	}
	if teamRoleBinding.Status.GetConditionByType(greenhousemetav1alpha1.OwnerLabelSetCondition).IsFalse() {
		ownedByLabelMissingGauge.With(ownedByLabelMissingLabels).Set(float64(1))
	} else {
		ownedByLabelMissingGauge.With(ownedByLabelMissingLabels).Set(float64(0))
	}
}
