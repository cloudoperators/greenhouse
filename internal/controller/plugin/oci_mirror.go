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
	"github.com/cloudoperators/greenhouse/internal/ocimirror"
)

func createRegistryMirrorPostRenderer(mirror *ocimirror.ImageMirror, renderedManifests string) *helmv2.PostRenderer {
	if mirror == nil {
		return nil
	}

	transforms := mirror.BuildImageTransformations(renderedManifests)
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

// ensureImageReplication triggers pre-replication of container images found in renderedManifests.
// It updates the Plugin's ImageReplication status and sets the ImageReplicationReady condition.
func ensureImageReplication(ctx context.Context, mirror *ocimirror.ImageMirror, plugin *greenhousev1alpha1.Plugin, renderedManifests string) error {
	replicated, err := mirror.ReplicateOCIArtifacts(ctx, renderedManifests, plugin.Status.ImageReplication)
	plugin.Status.ImageReplication = replicated

	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ImageReplicationReadyCondition,
			greenhousev1alpha1.ImageReplicationFailedReason,
			err.Error()))
		return fmt.Errorf("image replication failed for Plugin %s: %w", plugin.Name, err)
	}

	if len(replicated) == 0 {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.ImageReplicationReadyCondition,
			greenhousev1alpha1.ImageReplicationNotConfiguredReason,
			"no container images matched configured mirrors"))
		return nil
	}

	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
		greenhousev1alpha1.ImageReplicationReadyCondition,
		greenhousev1alpha1.ImageReplicationSucceededReason,
		fmt.Sprintf("replicated %d container images", len(replicated))))
	return nil
}
