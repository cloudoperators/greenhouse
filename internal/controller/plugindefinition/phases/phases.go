// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// Phase holds per-reconcile state for plugindefinition subroutines.
type Phase struct {
	Client              client.Client
	Recorder            events.EventRecorder
	PluginDef           common.GenericPluginDefinition
	NamespaceName       string
	OCIMirroringEnabled bool

	// helmRepo is set by ensureHelmRepository and consumed by ensureHelmChart.
	helmRepo *sourcev1.HelmRepository
}

// EnsureCreatePhases returns the ordered subroutines for a create/update reconcile.
func (p *Phase) EnsureCreatePhases() []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureHelmRepository(),
		p.ensureChartReplication(),
		p.ensureHelmChart(),
	}
}

// EnsureDeletePhases returns the ordered subroutines for a delete reconcile.
func (p *Phase) EnsureDeletePhases() []lifecycle.SubRoutine {
	return []lifecycle.SubRoutine{
		p.ensureOrphanedHelmChartsDeleted(),
	}
}
