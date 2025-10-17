// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/rbac"
)

func (r *CatalogReconciler) getExternalArtifact(ctx context.Context, catalog *greenhousev1alpha1.Catalog) (*sourcev1.ExternalArtifact, error) {
	artifact := &sourcev1.ExternalArtifact{}
	artifact.SetName(catalog.Name)
	artifact.SetNamespace(catalog.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(artifact), artifact)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

func (r *CatalogReconciler) ensureKustomization(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	externalArtifact, err := r.getExternalArtifact(ctx, catalog)
	if err != nil {
		return err
	}
	ggvk := externalArtifact.GroupVersionKind()
	kuz := flux.NewKustomizationSpecBuilder(r.Log)
	kuz = kuz.WithSourceRef(ggvk.String(), ggvk.Kind, externalArtifact.Name, externalArtifact.Namespace)
	if len(catalog.Spec.Source.Overrides) > 0 {
		patches, err := flux.PrepareKustomizePatches(catalog.Spec.Source.Overrides, greenhousev1alpha1.GroupVersion.Group)
		if err != nil {
			return err
		}
		kuz = kuz.WithPatches(patches)
	}
	// Set the ServiceAccount for the organization's PluginDefinitionCatalog operations
	serviceAccountName := rbac.OrgCatalogServiceAccountName(catalog.Namespace)
	kuz = kuz.WithServiceAccountName(serviceAccountName)
	kustomizationSpec, err := kuz.Build()
	if err != nil {
		return err
	}
	// when flux resources is being updated by greenhouse controller and in parallel by flux controller, we need to retryOnConflict
	result, err := controllerutil.CreateOrPatch(ctx, r.Client, kustomization, func() error {
		kustomization.Spec = kustomizationSpec
		return controllerutil.SetControllerReference(catalog, kustomization, r.Scheme)
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		r.Log.Info("created kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultUpdated:
		r.Log.Info("updated kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultNone:
		r.Log.Info("No changes to catalog kustomization", "name", kustomization.Name, "namespace", kustomization.Namespace)
	default:
		r.Log.Info("result is unknown for catalog kustomization", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
	}
	return nil
}
