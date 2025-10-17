// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	artifactAlias   = "catalog"
	artifactCopyKey = "artifact"
	artifactToDir   = "catalogs"
)

func getArtifactSource(catalog *greenhousev1alpha1.Catalog) []sourcev2.SourceReference {
	return []sourcev2.SourceReference{
		{
			Alias:     artifactAlias,
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
			Kind:      sourcev1.GitRepositoryKind,
		},
	}
}

func getArtifacts(catalog *greenhousev1alpha1.Catalog) []sourcev2.OutputArtifact {
	resources := catalog.Spec.Source.Resources
	copyOps := make([]sourcev2.CopyOperation, 0, len(resources))
	for _, res := range resources {
		res = strings.TrimPrefix(res, "/")
		copyOps = append(copyOps, sourcev2.CopyOperation{
			From:     fmt.Sprintf("@%s/%s", artifactAlias, res),
			To:       fmt.Sprintf("@%s/%s/%s", artifactCopyKey, artifactToDir, res),
			Strategy: sourcev2.MergeStrategy,
		})
	}
	return []sourcev2.OutputArtifact{
		{
			Name: catalog.Name,
			Copy: copyOps,
		},
	}
}

func (r *CatalogReconciler) ensureArtifactGenerated(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	generator := &sourcev2.ArtifactGenerator{}
	generator.SetName(catalog.Name)
	generator.SetNamespace(catalog.Namespace)
	artifactSources := getArtifactSource(catalog)
	artifacts := getArtifacts(catalog)
	result, err := controllerutil.CreateOrPatch(ctx, r.Client, generator, func() error {
		generator.Spec.Sources = artifactSources
		generator.Spec.OutputArtifacts = artifacts
		return controllerutil.SetControllerReference(catalog, generator, r.Scheme)
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		r.Log.Info("created artifact generator for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultUpdated:
		r.Log.Info("updated artifact generator for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultNone:
		r.Log.Info("no changes to catalog artifact generator", "name", generator.Name, "namespace", generator.Namespace)
	default:
		r.Log.Info("result is unknown for catalog artifact generator", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
	}
	return nil
}
