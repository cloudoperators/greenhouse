// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"context"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func PreparePodInfoPluginDefinition(ctx context.Context, namespace, version string) *greenhousev1alpha1.PluginDefinition {
	return test.NewPluginDefinition(ctx, "podinfo", namespace,
		test.WithPluginDefinitionVersion(version),
		test.WithPluginDefinitionHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "podinfo",
				Repository: "oci://ghcr.io/stefanprodan/charts",
				Version:    version,
			},
		),
	)
}
