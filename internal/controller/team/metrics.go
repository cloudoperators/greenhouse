// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	membersCountMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_team_members_count",
			Help: "Members count in team",
		},
		[]string{"namespace", "team"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(membersCountMetric)
}

func UpdateTeamMembersCountMetric(team *greenhousev1alpha1.Team, membersCount int) {
	membersCountMetric.With(prometheus.Labels{
		"namespace": team.Namespace,
		"team":      team.Name,
	}).Set(float64(membersCount))
}
