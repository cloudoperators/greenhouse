// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const RequeueInterval = 10 * time.Minute

func (p *Phase) ensureRequeue(_ *greenhousev1alpha1.Team) lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		return lifecycle.RequeueAfter(wait.Jitter(RequeueInterval, 0.1)), nil
	}
}
