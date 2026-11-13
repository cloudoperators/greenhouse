// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	"github.com/dexidp/dex/storage"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	ExportDefaultTeamRoles          = defaultTeamRoles
	ExportIncrementResourceVersion  = incrementResourceVersion
	ExportUpdateOrganizationMetrics = updateOrganizationMetrics
	ExportSCIMAccessReadyGauge      = scimAccessReadyGauge
)

func ExportRemoveAuthRedirectFromDefaultConnector(ctx context.Context, dex storage.Storage, org *greenhousev1alpha1.Organization) error {
	return (&OrganizationReconciler{dex: dex}).removeAuthRedirectFromDefaultConnector(ctx, org)
}
