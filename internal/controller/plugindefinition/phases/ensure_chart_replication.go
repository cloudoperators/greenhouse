// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/ocimirror"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ensureChartReplication triggers replication for the Helm chart OCI artifact to the configured mirror registry.
func (p *Phase) ensureChartReplication() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if !p.OCIMirroringEnabled {
			p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
				greenhousev1alpha1.OCIReplicationReadyCondition,
				greenhousev1alpha1.OCIReplicationNotConfiguredReason,
				"OCI mirroring is disabled"))
			return lifecycle.Continue(), nil
		}

		logger := log.FromContext(ctx)

		// Check if the chart is OCI before making any API calls — non-OCI charts cannot be replicated.
		helmChart := p.PluginDef.GetPluginDefinitionSpec().HelmChart
		if !strings.HasPrefix(helmChart.Repository, "oci://") {
			logger.V(1).Info("chart repository is not OCI, skipping replication", "repository", helmChart.Repository)
			p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
				greenhousev1alpha1.OCIReplicationReadyCondition,
				greenhousev1alpha1.OCIReplicationNotConfiguredReason,
				"Chart replication only applies to OCI repositories"))
			return lifecycle.Continue(), nil
		}

		failReplication := func(err error, msg string) (lifecycle.Result, error) {
			logger.Error(err, msg)
			p.PluginDef.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.OCIReplicationReadyCondition,
				greenhousev1alpha1.OCIReplicationFailedReason,
				msg+": "+err.Error()))
			return lifecycle.Break(), err
		}

		mirrorConfig, err := ocimirror.GetRegistryMirrorConfig(ctx, p.Client, p.NamespaceName)
		if err != nil {
			return failReplication(err, "failed to get registry mirror config")
		}

		if mirrorConfig == nil || len(mirrorConfig.RegistryMirrors) == 0 {
			p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
				greenhousev1alpha1.OCIReplicationReadyCondition,
				greenhousev1alpha1.OCIReplicationNotConfiguredReason,
				"OCI replication is not configured"))
			return lifecycle.Continue(), nil
		}

		chartRef := strings.TrimPrefix(helmChart.Repository, "oci://") + "/" + helmChart.Name + ":" + helmChart.Version
		registry, chartName, _ := ocimirror.SplitOCIRef(chartRef)
		version := helmChart.Version

		// Skip replication if the current chart version was already replicated (idempotency).
		if ShouldSkipChartReplication(p.PluginDef, registry, chartName, version) {
			logger.V(1).Info("chart already replicated, skipping", "registry", registry, "chart", chartName, "version", version)
			return lifecycle.Continue(), nil
		}

		mirror, err := ocimirror.NewImageMirror(ctx, p.Client, mirrorConfig, p.NamespaceName)
		if err != nil {
			return failReplication(err, "failed to create image mirror")
		}
		replicatedRef, manifest, err := mirror.EnsureChartReplicated(ctx, chartRef)
		if err != nil {
			return failReplication(err, "chart replication failed")
		}
		if replicatedRef == "" {
			logger.Info("no mirror configured for chart registry, skipping replication", "chart", chartRef)
			p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
				greenhousev1alpha1.OCIReplicationReadyCondition,
				greenhousev1alpha1.OCIReplicationNotConfiguredReason,
				"No mirror configured for chart registry"))
			return lifecycle.Continue(), nil
		}

		digest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifest))

		p.PluginDef.SetLastSyncedArtifact(&greenhousev1alpha1.LastSyncedArtifact{
			Registry:          registry,
			ChartName:         chartName,
			Version:           version,
			Digest:            digest,
			ReplicationStatus: greenhousev1alpha1.ReplicationStatusReplicated,
		})
		p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.OCIReplicationReadyCondition,
			greenhousev1alpha1.OCIReplicationSucceededReason,
			"Chart replicated successfully: "+chartRef))
		return lifecycle.Continue(), nil
	}
}

// ShouldSkipChartReplication reports whether this chart version is already replicated.
// The reconcile annotation forces a re-fetch in case the mirror cleaned up the artifact.
func ShouldSkipChartReplication(pluginDef common.GenericPluginDefinition, registry, chartName, version string) bool {
	if _, requested := lifecycle.ReconcileAnnotationValue(pluginDef); requested {
		return false
	}
	artifact := pluginDef.GetLastSyncedArtifact()
	return artifact != nil &&
		artifact.Registry == registry &&
		artifact.ChartName == chartName &&
		artifact.Version == version &&
		artifact.ReplicationStatus == greenhousev1alpha1.ReplicationStatusReplicated
}
