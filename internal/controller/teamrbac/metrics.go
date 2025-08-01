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
		[]string{"team_role_binding", "team", "namespace", "owned_by"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(teamRBACReadyGauge)
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
}
