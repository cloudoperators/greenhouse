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

	teamRBACClustersGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_rbac_clusters_total",
			Help: "Number of clusters currently targeted by a TeamRoleBinding",
		},
		[]string{"team_role_binding", "team", "organization", "owned_by"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(teamRBACReadyGauge, teamRBACClustersGauge)
}

func UpdateTeamRBACMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyGauge.DeletePartialMatch(prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"organization":      teamRoleBinding.Namespace,
	})

	var value float64
	if teamRoleBinding.Status.IsReadyTrue() {
		value = 1
	}

	teamRefs := resolveTeamRefs(teamRoleBinding)
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
	teamRBACClustersGauge.DeletePartialMatch(prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"organization":      teamRoleBinding.Namespace,
	})
}

func UpdateTeamRBACClustersMetric(trb *greenhousev1alpha2.TeamRoleBinding, count int) {
	teamRBACClustersGauge.DeletePartialMatch(prometheus.Labels{
		"team_role_binding": trb.Name,
		"organization":      trb.Namespace,
	})
	for _, team := range resolveTeamRefs(trb) {
		teamRBACClustersGauge.With(prometheus.Labels{
			"team_role_binding": trb.Name,
			"team":              team,
			"organization":      trb.Namespace,
			"owned_by":          trb.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
		}).Set(float64(count))
	}
}
