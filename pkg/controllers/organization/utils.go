// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// noStatus - empty noStatus func to avoid continuous reconciliations for org controllers that don't implement conditions
// TODO: remove this once organization controllers are merged
func noStatus() lifecycle.Conditioner {
	return func(_ context.Context, _ lifecycle.RuntimeObject) {}
}
