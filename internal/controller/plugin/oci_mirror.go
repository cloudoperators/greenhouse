// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"

	"github.com/cloudoperators/greenhouse/internal/ocimirror"
)

func createRegistryMirrorPostRenderer(mirrorConfig *ocimirror.RegistryMirrorConfig, renderedManifests string) *helmv2.PostRenderer {
	if mirrorConfig == nil || len(mirrorConfig.RegistryMirrors) == 0 {
		return nil
	}

	images := buildImageTransformations(renderedManifests, mirrorConfig)
	if len(images) == 0 {
		return nil
	}

	return &helmv2.PostRenderer{
		Kustomize: &helmv2.Kustomize{
			Images: images,
		},
	}
}

func buildImageTransformations(manifests string, config *ocimirror.RegistryMirrorConfig) []kustomize.Image {
	imageRefs := ocimirror.ExtractUniqueOCIRefs(manifests)

	var transformations []kustomize.Image
	for _, imageRef := range imageRefs {
		resolved := config.ResolveOCIRef(imageRef)
		if resolved == nil {
			continue
		}

		baseName := fmt.Sprintf("%s/%s", resolved.Registry, resolved.Repository)
		newName := fmt.Sprintf("%s/%s/%s", resolved.Mirror.BaseDomain, resolved.Mirror.SubPath, resolved.Repository)

		transformations = append(transformations, kustomize.Image{
			Name:    baseName,
			NewName: newName,
		})
	}

	return transformations
}
