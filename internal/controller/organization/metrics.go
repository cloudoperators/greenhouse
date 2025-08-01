// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	organizationReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_organization_ready",
			Help: "Indicates whether the overall ready state of the organization is true",
		},
		[]string{"organization"},
	)

	scimAccessReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_scim_access_ready",
			Help: "Indicates whether the SCIM access is ready",
		},
		[]string{"organization"},
	)
)

func init() {
	crmetrics.Registry.MustRegister(organizationReadyGauge)
	crmetrics.Registry.MustRegister(scimAccessReadyGauge)
}

func UpdateOrganizationMetrics(organization *greenhousev1alpha1.Organization) {
	organizationLabels := prometheus.Labels{
		"organization": organization.Name,
	}

	if organization.Status.IsReadyTrue() {
		organizationReadyGauge.With(organizationLabels).Set(float64(1))
	} else {
		organizationReadyGauge.With(organizationLabels).Set(float64(0))
	}

	if organization.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition).IsTrue() {
		scimAccessReadyGauge.With(organizationLabels).Set(float64(1))
	} else {
		scimAccessReadyGauge.With(organizationLabels).Set(float64(0))
	}
}
