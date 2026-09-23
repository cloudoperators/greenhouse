// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// Phase holds per-reconcile state for team subroutines.
type Phase struct {
	client.Client
	Recorder events.EventRecorder
}

// EnsureCreatePhases returns the ordered subroutines for a create/update reconcile.
func (p *Phase) EnsureCreatePhases(team *greenhousev1alpha1.Team) []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureCleanupIfNotSupportGroup(team),
		p.ensureSCIMMembers(team),
		p.ensureSupportGroupResources(team),
		p.ensureRequeue(team),
	}
}

// EnsureDeletePhases returns the ordered subroutines for a delete reconcile.
func (p *Phase) EnsureDeletePhases(team *greenhousev1alpha1.Team) []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureSupportGroupResourcesDeleted(team),
	}
}
