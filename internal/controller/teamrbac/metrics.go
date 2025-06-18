// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teamrbac

import (
	"github.com/prometheus/client_golang/prometheus"

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
	prometheus.MustRegister(teamRBACReadyGauge)
}

func updateMetrics(teamRoleBinding *greenhousev1alpha2.TeamRoleBinding) {
	labels := prometheus.Labels{
		"team_role_binding": teamRoleBinding.Name,
		"team":              teamRoleBinding.Spec.TeamRef,
		"organization":      teamRoleBinding.Namespace,
		"owned_by":          teamRoleBinding.GetLabels()[greenhouseapis.LabelKeyOwnedBy],
	}

	if teamRoleBinding.Status.IsReadyTrue() {
		teamRBACReadyGauge.With(labels).Set(1)
	} else {
		teamRBACReadyGauge.With(labels).Set(0)
	}
}
