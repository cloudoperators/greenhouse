// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

const (
	metricLabelOrganization    = "organization"
	metricLabelTeamRoleBinding = "team_role_binding"
	metricLabelTeam            = "team"
	metricLabelOwnedBy         = "owned_by"
)

var (
	teamRBACReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_rbac_ready",
			Help: "Indicates whether the team RBAC is ready",
		},
		[]string{metricLabelTeamRoleBinding, metricLabelTeam, metricLabelOrganization, metricLabelOwnedBy},
	)

	teamRBACClustersGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_rbac_clusters_total",
			Help: "Number of clusters currently targeted by a TeamRoleBinding",
		},
		[]string{metricLabelTeamRoleBinding, metricLabelTeam, metricLabelOrganization, metricLabelOwnedBy},
	)
)

func init() {
	crmetrics.Registry.MustRegister(teamRBACReadyGauge, teamRBACClustersGauge)
}

func UpdateTeamRBACMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyGauge.DeletePartialMatch(prometheus.Labels{
		metricLabelTeamRoleBinding: teamRoleBinding.Name,
		metricLabelOrganization:    teamRoleBinding.Namespace,
	})

	var value float64
	if teamRoleBinding.Status.IsReadyTrue() {
		value = 1
	}

	teamRefs := resolveTeamRefs(teamRoleBinding)
	for _, team := range teamRefs {
		teamRBACReadyGauge.With(prometheus.Labels{
			metricLabelTeamRoleBinding: teamRoleBinding.Name,
			metricLabelTeam:            team,
			metricLabelOrganization:    teamRoleBinding.Namespace,
			metricLabelOwnedBy:         teamRoleBinding.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
		}).Set(value)
	}
}

func DeleteTeamRBACMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	teamRBACReadyGauge.DeletePartialMatch(prometheus.Labels{
		metricLabelTeamRoleBinding: teamRoleBinding.Name,
		metricLabelOrganization:    teamRoleBinding.Namespace,
	})
	teamRBACClustersGauge.DeletePartialMatch(prometheus.Labels{
		metricLabelTeamRoleBinding: teamRoleBinding.Name,
		metricLabelOrganization:    teamRoleBinding.Namespace,
	})
}

func UpdateTeamRBACClustersMetric(trb *greenhousev1alpha2.TeamRoleBinding, count int) {
	teamRBACClustersGauge.DeletePartialMatch(prometheus.Labels{
		metricLabelTeamRoleBinding: trb.Name,
		metricLabelOrganization:    trb.Namespace,
	})
	for _, team := range resolveTeamRefs(trb) {
		teamRBACClustersGauge.With(prometheus.Labels{
			metricLabelTeamRoleBinding: trb.Name,
			metricLabelTeam:            team,
			metricLabelOrganization:    trb.Namespace,
			metricLabelOwnedBy:         trb.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
		}).Set(float64(count))
	}
}
