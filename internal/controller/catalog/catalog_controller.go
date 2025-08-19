// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/rbac"
)

type CatalogReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
	log      logr.Logger
}

func (r *CatalogReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Catalog{},
			builder.WithPredicates(
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					predicate.ResourceVersionChangedPredicate{},
					predicate.AnnotationChangedPredicate{},
					predicate.LabelChangedPredicate{},
				),
			),
		).
		Owns(&sourcev1.GitRepository{}).
		Owns(&kustomizev1.Kustomization{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *CatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Catalog{}, r, nil)
}

func (r *CatalogReconciler) EnsureDeleted(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	log := ctrl.LoggerFrom(ctx)
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	log.Info("attempting to delete catalog", "name", catalog.Name, "namespace", catalog.Namespace)

	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	shouldRequeue, err := r.ensureResourceIsDeleted(ctx, catalog, kustomization)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		log.Info("waiting for kustomization to be deleted", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}
	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted Kustomization for catalog: %s/%s", catalog.Namespace, catalog.Name)

	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	shouldRequeue, err = r.ensureResourceIsDeleted(ctx, catalog, gitRepository)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		log.Info("waiting for git repository to be deleted", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}
	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted GitRepository for catalog: %s/%s", catalog.Namespace, catalog.Name)

	log.Info("catalog deleted", "name", catalog.Name, "namespace", catalog.Namespace)
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *CatalogReconciler) ensureResourceIsDeleted(ctx context.Context, catalog, obj client.Object) (requeue bool, err error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err = r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if errors.IsNotFound(err) {
			err = nil
			return
		}
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: fetching %s for catalog: %s/%s - %s", kind, catalog.GetNamespace(), catalog.GetName(), err.Error())
		r.log.Error(err, "failed to get object", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return
	}
	if obj.GetDeletionTimestamp().IsZero() {
		if err = r.Delete(ctx, obj); err != nil {
			r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: deleting %s for catalog: %s/%s - %s", kind, catalog.GetNamespace(), catalog.GetName(), err.Error())
			r.log.Error(err, "failed to delete object", "name", obj.GetName(), "namespace", obj.GetNamespace())
			return
		}
	} else {
		obj.SetAnnotations(map[string]string{
			greenhouseapis.FluxReconcileRequestAnnotation: time.Now().Format(time.DateTime),
		})
		if err = r.Update(ctx, obj); err != nil {
			return
		}
	}
	requeue = true
	return
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	r.log = ctrl.LoggerFrom(ctx)
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck

	err := r.ensureGitRepository(ctx, catalog)
	if err != nil {
		r.log.Error(err, "failed to ensure git repository for catalog", "name", catalog, "namespace", catalog.Namespace)
	}

	gitRepository, shouldRequeue, err := r.ensureGitRepositoryIsReady(ctx, catalog)
	if err != nil {
		r.log.Error(err, "failed to ensure git repository is ready for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}

	err = r.ensureKustomization(ctx, gitRepository, catalog)
	if err != nil {
		r.log.Error(err, "failed to ensure kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{}, lifecycle.Failed, err
	}

	shouldRequeue, err = r.ensureKustomizationIsReady(ctx, catalog)
	if err != nil {
		r.log.Error(err, "failed to ensure kustomization is ready for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *CatalogReconciler) ensureGitRepository(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	var err error
	gitSource := catalog.GetCatalogSource()
	gitReference := &sourcev1.GitRepositoryRef{}
	if gitSource.Ref != nil {
		// flux precedence 1
		if gitSource.Ref.SHA != nil {
			gitReference.Commit = *gitSource.Ref.SHA
		}
		// flux precedence 2
		if gitSource.Ref.Tag != nil {
			gitReference.Tag = *gitSource.Ref.Tag
		}
		// flux precedence 3
		if gitSource.Ref.Branch != nil {
			gitReference.Branch = *gitSource.Ref.Branch
		}
	}

	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	// when flux resources is being updated by greenhouse controller and in parallel by flux controller, we need to retryOnConflict
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := ctrl.CreateOrUpdate(ctx, r.Client, gitRepository, func() error {
			gitRepository.Spec = sourcev1.GitRepositorySpec{
				URL:       gitSource.URL,
				Interval:  metav1.Duration{Duration: flux.DefaultInterval},
				Reference: gitReference,
			}
			return controllerutil.SetControllerReference(catalog, gitRepository, r.Scheme)
		})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			r.log.Info("created git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Created", "Created GitRepository %s", gitRepository.Name)
		case controllerutil.OperationResultUpdated:
			r.log.Info("updated git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Updated", "Updated GitRepository %s", gitRepository.Name)
		case controllerutil.OperationResultNone:
			r.log.Info("No changes to catalog git repository", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
		default:
			r.log.Info("result is unknown for catalog git repository", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
		}
		return nil
	})
	if err != nil {
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: GitRepository %s - %s", gitRepository.Name, err.Error())
		return err
	}
	return nil
}

func (r *CatalogReconciler) ensureGitRepositoryIsReady(ctx context.Context, catalog *greenhousev1alpha1.Catalog) (gitRepository *sourcev1.GitRepository, requeue bool, err error) {
	gitRepository = &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	err = r.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)
	if err != nil {
		return
	}

	readyCond := meta.FindStatusCondition(gitRepository.Status.Conditions, "Ready")
	if readyCond == nil {
		r.log.Info("catalog git repository resource is not yet ready", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
		requeue = true
		return
	}

	if readyCond.Status == metav1.ConditionUnknown {
		r.log.Info("catalog git repository resource is in an unknown state", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
		requeue = true
		return
	}

	if readyCond.Status == metav1.ConditionFalse {
		err = fmt.Errorf("catalog git repository %s/%s is not ready: %s", gitRepository.Namespace, gitRepository.Name, readyCond.Message)
		return
	}

	r.log.Info("catalog git repository resource is ready", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
	return
}

func (r *CatalogReconciler) ensureKustomization(ctx context.Context, gitRepository *sourcev1.GitRepository, catalog *greenhousev1alpha1.Catalog) error {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	ggvk := gitRepository.GroupVersionKind()
	kuz := flux.NewKustomizationSpecBuilder(r.log)
	kuz = kuz.WithSourceRef(ggvk.String(), ggvk.Kind, gitRepository.Name, gitRepository.Namespace)
	if catalog.ResourcePath() != "" {
		kuz = kuz.WithPath(catalog.ResourcePath())
	}
	if len(catalog.Spec.Overrides) > 0 {
		patches, err := flux.PrepareKustomizePatches(catalog.Spec.Overrides, greenhousev1alpha1.GroupVersion.Group)
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
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := ctrl.CreateOrUpdate(ctx, r.Client, kustomization, func() error {
			kustomization.Spec = kustomizationSpec
			return controllerutil.SetControllerReference(catalog, kustomization, r.Scheme)
		})
		if err != nil {
			return err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			r.log.Info("created kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Created", "Created Kustomization %s", kustomization.Name)
		case controllerutil.OperationResultUpdated:
			r.log.Info("updated kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Updated", "Updated Kustomization %s", kustomization.Name)
		case controllerutil.OperationResultNone:
			r.log.Info("No changes to catalog kustomization", "name", kustomization.Name, "namespace", kustomization.Namespace)
		default:
			r.log.Info("result is unknown for catalog kustomization", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
		}
		return nil
	})
	if err != nil {
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: Kustomization %s - %s", kustomization.Name, err.Error())
		return err
	}
	return nil
}

func (r *CatalogReconciler) ensureKustomizationIsReady(ctx context.Context, catalog *greenhousev1alpha1.Catalog) (requeue bool, err error) {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	err = r.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)
	if err != nil {
		r.log.Error(err, "failed to get kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		return
	}

	readyCond := meta.FindStatusCondition(kustomization.Status.Conditions, "Ready")
	if readyCond == nil {
		r.log.Info("catalog kustomization resource is not yet ready", "name", kustomization.Name, "namespace", kustomization.Namespace)
		requeue = true
		return
	}

	if readyCond.Status == metav1.ConditionUnknown {
		r.log.Info("catalog kustomization resource is in an unknown state", "name", kustomization.Name, "namespace", kustomization.Namespace)
		requeue = true
		return
	}

	if readyCond.Status == metav1.ConditionFalse {
		err = fmt.Errorf("catalog kustomization %s/%s is not ready: %s", kustomization.Namespace, kustomization.Name, readyCond.Message)
		return
	}
	r.log.Info("kustomization resource is ready", "name", kustomization.Name, "namespace", kustomization.Namespace)
	return
}
