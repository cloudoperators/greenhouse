// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

var (
	teamRBACReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_rbac_ready",
			Help: "Indicates whether the team RBAC is ready",
		},
		[]string{"team_role_binding", "team", "organization", "owned_by"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(teamRBACReadyGauge)
}

func UpdateTeamrbacMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyGauge.DeletePartialMatch(prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"organization":      teamRoleBinding.Namespace,
	})

	teamRefs := resolveTeamRefs(teamRoleBinding)

	var value float64
	if teamRoleBinding.Status.IsReadyTrue() {
		value = 1
	}

	for _, team := range teamRefs {
		teamRBACReadyGauge.With(prometheus.Labels{
			"team_role_binding": teamRoleBinding.Name,
			"team":              team,
			"organization":      teamRoleBinding.Namespace,
			"owned_by":          teamRoleBinding.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
		}).Set(value)
	}
}

func DeleteTeamRBACMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyGauge.DeletePartialMatch(prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"organization":      teamRoleBinding.Namespace,
	})
}
