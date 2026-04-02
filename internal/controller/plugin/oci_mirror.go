// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"

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
