// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func PrepareCertManagerPluginDefinition() *greenhousev1alpha1.ClusterPluginDefinition {
	return test.NewClusterPluginDefinition("cert-manager-v1.17.0",
		test.WithVersion("v1.17.0"),
		test.WithHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "cert-manager",
				Repository: "https://charts.jetstack.io",
				Version:    "v1.17.0",
			},
		),
	)
}
