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
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

type CatalogReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Log         logr.Logger
	recorder    record.EventRecorder
	StoragePath string
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
		Watches(&sourcev1.ExternalArtifact{}, handler.EnqueueRequestsFromMapFunc(r.reconcileCatalogOnExternalArtifactChange)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5,
		}).
		Complete(r)
}

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
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Catalog{}, r, r.setStatus())
}

func (r *CatalogReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	// owner references on child resources will handle deletion
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *CatalogReconciler) EnsureSuspended(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	allErrors := make([]error, 0)
	for _, s := range catalog.Status.Inventory {
		for _, inv := range s {
			switch inv.Kind {
			case sourcev1.GitRepositoryKind:
				if err := r.suspendGitRepository(ctx, inv.Name, catalog.Namespace); err != nil {
					allErrors = append(allErrors, err)
				}
			case kustomizev1.KustomizationKind:
				if err := r.suspendKustomization(ctx, inv.Name, catalog.Namespace); err != nil {
					allErrors = append(allErrors, err)
				}
			case sourcev2.ArtifactGeneratorKind:
				if err := r.suspendArtifactGenerator(ctx, inv.Name, catalog.Namespace); err != nil {
					allErrors = append(allErrors, err)
				}
			case sourcev1.ExternalArtifactKind:
				// ignore as it is managed by ArtifactGenerator
				continue
			default:
				// ignore other kinds
				allErrors = append(allErrors, errors.New("unsupported kind for suspension: "+inv.Kind))
			}
		}
	}
	if len(allErrors) > 0 {
		var errMessages []string
		for _, oErr := range allErrors {
			errMessages = append(errMessages, oErr.Error())
		}
		err := errors.New(strings.Join(errMessages, "; "))
		r.Log.Error(err, "failed to suspend some catalog resources", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CatalogReconciler) suspendGitRepository(ctx context.Context, name, namespace string) error {
	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(name)
	gitRepository.SetNamespace(namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)

	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, gitRepository, func() error {
		gitRepository.Spec.Suspend = true
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes, Catalog's GitRepository already suspended", "name", gitRepository.Name)
	default:
		log.FromContext(ctx).Info("Suspend applied to Catalog's GitRepository", "name", gitRepository.Name)
	}
	return nil
}

func (r *CatalogReconciler) suspendKustomization(ctx context.Context, name, namespace string) error {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(name)
	kustomization.SetNamespace(namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)

	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, kustomization, func() error {
		kustomization.Spec.Suspend = true
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes, Catalog's GitRepository already suspended", "name", kustomization.Name)
	default:
		log.FromContext(ctx).Info("Suspend applied to Catalog's GitRepository", "name", kustomization.Name)
	}
	return nil
}

func (r *CatalogReconciler) suspendArtifactGenerator(ctx context.Context, name, namespace string) error {
	artifactGenerator := &sourcev2.ArtifactGenerator{}
	artifactGenerator.SetName(name)
	artifactGenerator.SetNamespace(namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(artifactGenerator), artifactGenerator)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, artifactGenerator, func() error {
		a := artifactGenerator.Annotations
		if a == nil {
			a = make(map[string]string)
		}
		a[sourcev2.ReconcileAnnotation] = sourcev2.DisabledValue
		artifactGenerator.Annotations = a
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes, Catalog's ArtifactGenerator already suspended", "name", artifactGenerator.Name)
	default:
		log.FromContext(ctx).Info("Suspend applied to Catalog's ArtifactGenerator", "name", artifactGenerator.Name)
	}
	return nil
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	catalog.SetUnknownCondition()

	if len(catalog.Spec.Sources) == 0 {
		msg := "no sources specified for catalog, skipping reconciliation"
		// in case of empty sources, check and clean inventory if existing to reflect the empty state
		err := r.checkAndCleanInventory(ctx, catalog)
		if err != nil {
			msg = msg + ": " + err.Error()
		}
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogEmptySources, msg)
		r.Log.Info(msg, "namespace", catalog.Namespace, "name", catalog.Name)
		return ctrl.Result{}, lifecycle.Pending, nil
	}

	if err := r.validateSources(catalog); err != nil {
		r.Log.Error(err, "failed to validate sources for catalog", "namespace", catalog.Namespace, "name", catalog.Name)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogSourceValidationFail, err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	if err := r.checkAndCleanInventory(ctx, catalog); err != nil {
		r.Log.Error(err, "failed to clean inventory for catalog", "namespace", catalog.Namespace, "name", catalog.Name)
		catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.OrphanedObjectCleanUpFail, err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	catalog.SetProgressingReason()

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

		ready, _ := sourcer.objectReadiness(ctx, externalArtifact)
		if ready != metav1.ConditionTrue {
			r.Log.Info("external artifact not ready yet, retry in next reconciliation loop", "namespace", catalog.Namespace, "name", sourcer.getArtifactName())
			continue
		}

		if err = sourcer.reconcileKustomization(ctx, externalArtifact); err != nil {
			r.Log.Error(err, "failed to reconcile kustomization for catalog source", "namespace", catalog.Namespace, "name", sourcer.getKustomizationName())
			allErrors = append(allErrors, err)
			continue
		}
	}
	if len(allErrors) > 0 {
		return ctrl.Result{}, lifecycle.Failed, utilerrors.NewAggregate(allErrors)
	}
	return ctrl.Result{}, lifecycle.Success, nil
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

func (r *CatalogReconciler) setStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		catalog := object.(*greenhousev1alpha1.Catalog) //nolint:errcheck
		existingNotReady := catalog.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		if existingNotReady != nil && existingNotReady.Status == metav1.ConditionFalse {
			if existingNotReady.Reason == greenhousev1alpha1.CatalogSourceValidationFail ||
				existingNotReady.Reason == greenhousev1alpha1.OrphanedObjectCleanUpFail ||
				existingNotReady.Reason == greenhousev1alpha1.CatalogEmptySources {
				return
			}
		}
		var allInventoryReady []metav1.ConditionStatus
		var srcErrs []error
		for _, source := range catalog.Spec.Sources {
			sourcer, srcErr := r.newCatalogSource(source, catalog)
			if srcErr != nil {
				srcErrs = append(srcErrs, srcErr)
				continue
			}
			gitRepo := &sourcev1.GitRepository{}
			gitRepo.SetName(sourcer.getGitRepoName())
			gitRepo.SetNamespace(catalog.Namespace)
			ready, msg := sourcer.objectReadiness(ctx, gitRepo)
			allInventoryReady = append(allInventoryReady, ready)
			sourcer.setInventory(sourcev1.GitRepositoryKind, gitRepo.Name, msg, ready)

			artifactGen := &sourcev2.ArtifactGenerator{}
			artifactGen.SetName(sourcer.getGeneratorName())
			artifactGen.SetNamespace(catalog.Namespace)
			ready, msg = sourcer.objectReadiness(ctx, artifactGen)
			allInventoryReady = append(allInventoryReady, ready)
			sourcer.setInventory(sourcev2.ArtifactGeneratorKind, artifactGen.Name, msg, ready)

			extArtifact := &sourcev1.ExternalArtifact{}
			extArtifact.SetName(sourcer.getArtifactName())
			extArtifact.SetNamespace(catalog.Namespace)
			ready, msg = sourcer.objectReadiness(ctx, extArtifact)
			allInventoryReady = append(allInventoryReady, ready)
			sourcer.setInventory(sourcev1.ExternalArtifactKind, extArtifact.Name, msg, ready)

			kustomization := &kustomizev1.Kustomization{}
			kustomization.SetName(sourcer.getKustomizationName())
			kustomization.SetNamespace(catalog.Namespace)
			ready, msg = sourcer.objectReadiness(ctx, kustomization)
			allInventoryReady = append(allInventoryReady, ready)
			sourcer.setInventory(kustomizev1.KustomizationKind, kustomization.Name, msg, ready)
		}

		if len(srcErrs) > 0 {
			err := utilerrors.NewAggregate(srcErrs)
			msg := "inventory readiness check incomplete: " + err.Error()
			r.Log.Error(err, "inventory readiness check incomplete", "namespace", catalog.Namespace, "name", catalog.Name)
			catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, msg)
			return
		}

		notReady := slices.ContainsFunc(allInventoryReady, func(status metav1.ConditionStatus) bool {
			return status != metav1.ConditionTrue // the status can contain Unknown as well
		})
		if notReady {
			r.Log.Info("not all catalog objects are ready", "namespace", catalog.Namespace, "name", catalog.Name)
			catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, "not all catalog objects are ready")
			return
		}
		r.Log.Info("all catalog objects are ready", "namespace", catalog.Namespace, "name", catalog.Name)
		catalog.SetTrueCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogReadyReason, "all catalog objects are ready")
	}
}

func (r *CatalogReconciler) reconcileCatalogOnExternalArtifactChange(_ context.Context, obj client.Object) []ctrl.Request {
	labels := obj.GetLabels()
	if labels == nil {
		return nil
	}
	catalogName, ok := labels[greenhouseapis.LabelKeyCatalog]
	if !ok {
		return nil
	}
	catalogNamespace := obj.GetNamespace()
	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKey{
				Name:      catalogName,
				Namespace: catalogNamespace,
			},
		},
	}
}
