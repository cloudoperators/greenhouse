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

// ensureImageReplication pre-replicates container images referenced in renderedManifests.
// On failure it flags HelmReleaseCreatedCondition with ImageReplicationFailedReason and keeps
// the previous status.ImageReplication list intact, so transient errors dont wipe history.
func ensureImageReplication(ctx context.Context, mirror *ocimirror.ImageMirror, plugin *greenhousev1alpha1.Plugin, renderedManifests string) error {
	replicated, err := mirror.ReplicateOCIArtifacts(ctx, renderedManifests, plugin.Status.ImageReplication)
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
