// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/ocimirror"
)

// createRegistryMirrorPostRenderer handles the full OCI mirroring flow.
func (r *PluginReconciler) createRegistryMirrorPostRenderer(
	ctx context.Context,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	optionValues []greenhousev1alpha1.PluginOptionValue,
) (*helmv2.PostRenderer, error) {

	if !r.OCIMirroringEnabled {
		return nil, nil
	}

	mirrorConfig, err := ocimirror.GetRegistryMirrorConfig(ctx, r.Client, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReleaseCreatedCondition, greenhousev1alpha1.ImageReplicationFailedReason, "Failed to read Organization registry mirror configuration: "+err.Error()))
		return nil, fmt.Errorf("failed to read registry mirror configuration for Plugin %s: %w", plugin.Name, err)
	}

	var mirror *ocimirror.ImageMirror
	if mirrorConfig != nil && len(mirrorConfig.RegistryMirrors) > 0 {
		mirror, err = ocimirror.NewImageMirror(ctx, r.Client, mirrorConfig, plugin.GetNamespace())
		if err != nil {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.HelmReleaseCreatedCondition, greenhousev1alpha1.ImageReplicationFailedReason, "Failed to create image mirror: "+err.Error()))
			return nil, fmt.Errorf("failed to create image mirror for Plugin %s: %w", plugin.Name, err)
		}
	}

	if mirror == nil {
		return nil, nil
	}

	restClientGetter, _, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to init client getter for Plugin %s: %w", plugin.Name, err)
	}

	helmRelease, err := helm.TemplateHelmChartFromPluginOptionValues(ctx, r.Client, restClientGetter, &pluginDefinitionSpec, plugin, optionValues)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReleaseCreatedCondition, greenhousev1alpha1.PluginHelmTemplateFailedReason, err.Error()))
		return nil, fmt.Errorf("failed to template helm chart for Plugin %s: %w", plugin.Name, err)
	}

	manifestSets := []string{helmRelease.Manifest}
	for _, h := range helmRelease.Hooks {
		manifestSets = append(manifestSets, h.Manifest)
	}

	if err := ensureImageReplication(ctx, mirror, plugin, manifestSets...); err != nil {
		return nil, err
	}

	return buildPostRenderer(mirror, manifestSets...), nil
}

func buildPostRenderer(mirror *ocimirror.ImageMirror, manifestSets ...string) *helmv2.PostRenderer {
	if mirror == nil {
		return nil
	}

	transforms := mirror.BuildImageTransformations(manifestSets...)
	if len(transforms) == 0 {
		return nil
	}

	images := make([]kustomize.Image, 0, len(transforms))
	for _, t := range transforms {
		images = append(images, kustomize.Image{
			Name:    t.Original,
			NewName: t.Mirrored,
		})
	}

	return &helmv2.PostRenderer{
		Kustomize: &helmv2.Kustomize{
			Images: images,
		},
	}
}

// ensureImageReplication pre-replicates container images referenced in renderedManifests.
// On failure it flags HelmReleaseCreatedCondition with ImageReplicationFailedReason and keeps
// the previous status.ImageReplication list intact, so transient errors dont wipe history.
func ensureImageReplication(ctx context.Context, mirror *ocimirror.ImageMirror, plugin *greenhousev1alpha1.Plugin, manifestSets ...string) error {
	replicated, err := mirror.ReplicateOCIArtifacts(ctx, plugin.Status.ImageReplication, manifestSets...)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReleaseCreatedCondition,
			greenhousev1alpha1.ImageReplicationFailedReason,
			err.Error()))
		return fmt.Errorf("image replication failed for Plugin %s: %w", plugin.Name, err)
	}

	plugin.Status.ImageReplication = replicated
	return nil
}
