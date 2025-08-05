// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

var catalogConditionTypes = []greenhousemetav1alpha1.ConditionType{
	greenhousemetav1alpha1.ReadyCondition,
	greenhousev1alpha1.GitRepositoryReady,
	greenhousev1alpha1.KustomizationReady,
	greenhousev1alpha1.CatalogSuspended,
}

type CatalogReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	recorder         record.EventRecorder
	log              logr.Logger
	gitRepository    *sourcev1.GitRepository
	suspendResources bool
}

func (r *CatalogReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginDefinitionCatalog{},
			builder.WithPredicates(
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					predicate.AnnotationChangedPredicate{},
					predicate.LabelChangedPredicate{},
				),
			),
		).
		Owns(&sourcev1.GitRepository{}).
		Owns(&kustomizev1.Kustomization{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}

func (r *CatalogReconciler) catalogStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		conditions := object.GetConditions()
		gitRepositoryReady := conditions.GetConditionByType(greenhousev1alpha1.GitRepositoryReady)
		kustomizationReady := conditions.GetConditionByType(greenhousev1alpha1.KustomizationReady)

		if gitRepositoryReady == nil {
			conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, gitRepositoryReady.Reason, gitRepositoryReady.Message))
			return
		}

		if kustomizationReady == nil {
			conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, kustomizationReady.Reason, kustomizationReady.Message))
			return
		}

		if gitRepositoryReady.IsTrue() && kustomizationReady.IsTrue() {
			conditions.SetConditions(greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogReadyReason, "Catalog is ready"))
			return
		}
		conditions.SetConditions(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, "Catalog is not ready"))
	}
}

func (r *CatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.PluginDefinitionCatalog{}, r, r.catalogStatus())
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	r.log = ctrl.LoggerFrom(ctx)
	catalog := obj.(*greenhousev1alpha1.PluginDefinitionCatalog) // nolint:staticcheck

	lifecycle.InitConditions(obj, catalogConditionTypes)

	if catalog.IsSuspended() {
		r.log.Info("catalog source is suspended", "name", catalog.Name, "namespace", catalog.Namespace)
		r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Skipped", "Catalog %s is suspended", catalog.Name)
		catalog.SetSuspendedCondition()
	} else {
		catalog.UnsetSuspendedCondition()
	}

	routines := []lifecycle.ReconcileRoutine{
		r.ensureGitRepository(catalog),
		r.ensureGitRepositoryIsReady(catalog),
		r.ensureKustomization(catalog),
		r.ensureKustomizationIsReady(catalog),
		// TODO: add ensure owner ref on pluginDefinition
		// looking at the status.KustomizeInventory, we can determine which plugin definitions are applied
		// if the plugin definition is namespaced, add the catalog controller owner reference to the plugin definition
		// so that reconcile is triggered if the plugin definition is manually changed
	}
	return lifecycle.ExecuteReconcileRoutine(ctx, routines)
}

func (r *CatalogReconciler) EnsureDeleted(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	log := ctrl.LoggerFrom(ctx)
	catalog := obj.(*greenhousev1alpha1.PluginDefinitionCatalog) // nolint:staticcheck
	log.Info("catalog deleted", "name", catalog.Name, "namespace", catalog.Namespace)
	return ctrl.Result{}, lifecycle.Success, nil
}
