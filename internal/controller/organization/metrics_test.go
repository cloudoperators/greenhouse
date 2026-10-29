// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"testing"

	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/organization"
)

func TestUpdateOrganizationMetricsSCIMAccessReady(t *testing.T) {
	tests := []struct {
		name       string
		condition  greenhousemetav1alpha1.Condition
		wantSeries int
		wantValue  float64
	}{
		{
			name:       "SCIM API available",
			condition:  greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""),
			wantSeries: 1,
			wantValue:  1,
		},
		{
			name:       "SCIM API not available",
			condition:  greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""),
			wantSeries: 1,
			wantValue:  0,
		},
		{
			name:       "SCIM not configured",
			condition:  greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""),
			wantSeries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organization.ExportSCIMAccessReadyGauge.Reset()
			org := &greenhousev1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: "test-org"}}
			org.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""))
			organization.ExportUpdateOrganizationMetrics(org)

			org.SetCondition(tt.condition)
			organization.ExportUpdateOrganizationMetrics(org)

			require.Equal(t, tt.wantSeries, prometheusTest.CollectAndCount(organization.ExportSCIMAccessReadyGauge))
			if tt.wantSeries > 0 {
				require.EqualValues(t, tt.wantValue, prometheusTest.ToFloat64(organization.ExportSCIMAccessReadyGauge.WithLabelValues(org.Name)))
			}
		})
	}
}
