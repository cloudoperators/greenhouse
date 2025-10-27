// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

type CatalogReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	recorder record.EventRecorder
}

func (r *CatalogReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Catalog{}).
		Owns(&sourcev1.GitRepository{}).
		Owns(&sourcev2.ArtifactGenerator{}).
		Owns(&kustomizev1.Kustomization{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5,
		}).
		Complete(r)
}

var noOp = func(_ context.Context, _ lifecycle.RuntimeObject) {}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=externalartifacts, verbs=get;list;watch
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.extensions.fluxcd.io,resources=artifactgenerators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *CatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Catalog{}, r, noOp)
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	catalog.SetUnknownCondition()
	if err := r.checkAndCleanInventory(ctx, catalog); err != nil {
		r.Log.Error(err, "failed to clean inventory for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	if len(catalog.Spec.Sources) == 0 {
		msg := "no sources specified for catalog, skipping reconciliation"
		r.Log.Info(msg, "name", catalog.Name, "namespace", catalog.Namespace)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, msg)
		return ctrl.Result{}, lifecycle.Pending, nil
	}

	if err := r.validateSources(catalog); err != nil {
		r.Log.Error(err, "failed to validate sources for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	var allErrors []error
	for _, s := range catalog.Spec.Sources {
		sourcer, err := r.newCatalogSource(s, catalog)
		if err != nil {
			r.Log.Error(err, "failed to create source reconciler for catalog", "namespace", catalog.Namespace, "name", catalog.Name)
			allErrors = append(allErrors, err)
			continue
		}

		if err = sourcer.reconcileGitRepository(ctx); err != nil {
			r.Log.Error(err, "failed to reconcile git repository for catalog source", "namespace", catalog.Namespace, "name", sourcer.getGitRepoName())
			allErrors = append(allErrors, err)
			continue
		}

		var externalArtifact *sourcev1.ExternalArtifact
		if externalArtifact, err = sourcer.reconcileArtifactGeneration(ctx); err != nil {
			r.Log.Error(err, "failed to reconcile artifact generation for catalog source", "namespace", catalog.Namespace, "name", sourcer.getArtifactName())
			allErrors = append(allErrors, err)
			continue
		}

		if err = sourcer.reconcileKustomization(ctx, externalArtifact); err != nil {
			r.Log.Error(err, "failed to reconcile kustomization for catalog source", "namespace", catalog.Namespace, "name", sourcer.getKustomizationName())
			allErrors = append(allErrors, err)
			continue
		}
	}
	return r.setStatus(ctx, catalog, allErrors)
}

// checkAndCleanInventory compares the current inventory in status with the desired inventory based on spec sources
// if there is a diff in status vs desired inventory, it deletes the orphaned resources
func (r *CatalogReconciler) checkAndCleanInventory(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	if len(catalog.Status.Inventory) == 0 {
		return nil
	}
	statusInventory := catalog.Status.Inventory
	desiredInventory := make(map[string][]greenhousev1alpha1.SourceStatus)

	for _, s := range catalog.Spec.Sources {
		sourcer, err := r.newCatalogSource(s, catalog)
		if err != nil {
			return err
		}
		hash := sourcer.getSourceGroupHash()
		desiredInventory[hash] = sourcer.getInventory()
	}

	opts := []cmp.Option{
		cmpopts.SortSlices(func(a, b greenhousev1alpha1.SourceStatus) bool {
			return a.Name < b.Name
		}),
	}

	for hash, invList := range statusInventory {
		desiredList := desiredInventory[hash]
		if diff := cmp.Diff(invList, desiredList, opts...); diff != "" {
			// Find orphaned resources in invList not present in desiredList
			for _, inv := range invList {
				found := false
				for _, desInv := range desiredList {
					if inv.Kind == desInv.Kind && inv.Name == desInv.Name {
						found = true
						break
					}
				}
				if !found {
					switch inv.Kind {
					case sourcev1.GitRepositoryKind:
						err := r.deleteOrphanedResource(ctx, &sourcev1.GitRepository{}, inv.Kind, catalog.Namespace, inv.Name)
						if err != nil {
							return fmt.Errorf("failed to delete %s - %s/%s: %w", inv.Kind, catalog.Namespace, inv.Name, err)
						}
					case sourcev2.ArtifactGeneratorKind:
						err := r.deleteOrphanedResource(ctx, &sourcev2.ArtifactGenerator{}, inv.Kind, catalog.Namespace, inv.Name)
						if err != nil {
							return fmt.Errorf("failed to delete %s - %s/%s: %w", inv.Kind, catalog.Namespace, inv.Name, err)
						}
					case sourcev1.ExternalArtifactKind:
						// ignore as it is managed by ArtifactGenerator
						break
					case kustomizev1.KustomizationKind:
						err := r.deleteOrphanedResource(ctx, &kustomizev1.Kustomization{}, inv.Kind, catalog.Namespace, inv.Name)
						if err != nil {
							return fmt.Errorf("failed to delete %s - %s/%s: %w", inv.Kind, catalog.Namespace, inv.Name, err)
						}
					default:
						r.Log.Info("unknown source kind in inventory, skipping deletion", "kind", inv.Kind, "name", inv.Name)
						continue
					}
					// Remove from inventory after successful deletion (except for ignored kinds)
					statusInventory[hash] = slices.DeleteFunc(statusInventory[hash], func(s greenhousev1alpha1.SourceStatus) bool {
						return s.Kind == inv.Kind && s.Name == inv.Name
					})
				}
			}
		}
		if len(statusInventory[hash]) == 0 {
			delete(statusInventory, hash)
		}
	}
	catalog.Status.Inventory = statusInventory
	return nil
}

func (r *CatalogReconciler) deleteOrphanedResource(ctx context.Context, obj client.Object, kind, namespace, name string) error {
	obj.SetNamespace(namespace)
	obj.SetName(name)
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get orphaned resource %s - %s/%s: %w", kind, namespace, name, err)
	}
	err = r.Delete(ctx, obj)
	if err != nil {
		return fmt.Errorf("failed to delete orphaned resource %s - %s/%s: %w", kind, namespace, name, err)
	}
	r.Log.Info("deleted orphaned resource", "kind", kind, "namespace", namespace, "name", name)
	return nil
}

func (r *CatalogReconciler) setStatus(ctx context.Context, catalog *greenhousev1alpha1.Catalog, allErrors []error) (ctrl.Result, lifecycle.ReconcileResult, error) {
	var err error
	if len(allErrors) > 0 {
		var errMessages []string
		for _, oErr := range allErrors {
			var srcErr *sourceError
			if errors.As(oErr, &srcErr) {
				catalog.SetInventory(srcErr.groupKey, srcErr.kind, srcErr.name, srcErr.Error(), metav1.ConditionFalse)
			}
			errMessages = append(errMessages, oErr.Error())
		}
		err = errors.New(strings.Join(errMessages, "; "))
		r.Log.Error(err, "failed to reconcile some catalog sources", "name", catalog.Name, "namespace", catalog.Namespace)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}
	for _, source := range catalog.Spec.Sources {
		sourcer, err := r.newCatalogSource(source, catalog)
		if err != nil {
			continue
		}
		gitRepo := &sourcev1.GitRepository{}
		gitRepo.SetName(sourcer.getGitRepoName())
		gitRepo.SetNamespace(catalog.Namespace)
		ready, msg := sourcer.objectReadiness(ctx, gitRepo)
		sourcer.setInventory(sourcev1.GitRepositoryKind, gitRepo.Name, msg, ready)

		artifactGen := &sourcev2.ArtifactGenerator{}
		artifactGen.SetName(sourcer.getGeneratorName())
		artifactGen.SetNamespace(catalog.Namespace)
		ready, msg = sourcer.objectReadiness(ctx, artifactGen)
		sourcer.setInventory(sourcev2.ArtifactGeneratorKind, artifactGen.Name, msg, ready)

		extArtifact := &sourcev1.ExternalArtifact{}
		extArtifact.SetName(sourcer.getArtifactName())
		extArtifact.SetNamespace(catalog.Namespace)
		ready, msg = sourcer.objectReadiness(ctx, extArtifact)
		sourcer.setInventory(sourcev1.ExternalArtifactKind, extArtifact.Name, msg, ready)

		kustomization := &kustomizev1.Kustomization{}
		kustomization.SetName(sourcer.getKustomizationName())
		kustomization.SetNamespace(catalog.Namespace)
		ready, msg = sourcer.objectReadiness(ctx, kustomization)
		sourcer.setInventory(kustomizev1.KustomizationKind, kustomization.Name, msg, ready)
	}

	var allInventoryReady []bool
	for _, invList := range catalog.Status.Inventory {
		ready := checkInventoryReadiness(invList)
		allInventoryReady = append(allInventoryReady, ready)
	}
	if slices.Contains(allInventoryReady, false) {
		r.Log.Info("not all catalog objects are ready", "name", catalog.Name, "namespace", catalog.Namespace)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, "not all objects are ready")
		return ctrl.Result{}, lifecycle.Pending, nil
	}
	r.Log.Info("all catalog objects are ready", "name", catalog.Name, "namespace", catalog.Namespace)
	catalog.SetTrueCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogReadyReason, "all objects are ready")
	return ctrl.Result{}, lifecycle.Success, nil
}

func checkInventoryReadiness(invList []greenhousev1alpha1.SourceStatus) (allReady bool) {
	allReady = true
	for _, inv := range invList {
		if inv.Ready != metav1.ConditionTrue {
			allReady = false
			break
		}
	}
	return
}

func (r *CatalogReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	// owner references on child resources will handle deletion
	return ctrl.Result{}, lifecycle.Success, nil
}
