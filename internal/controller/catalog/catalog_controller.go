// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"fmt"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/rbac"
)

const (
	genericAuthProvider   = "generic"
	githubAppAuthProvider = "github"
	githubAppIDKey        = "githubAppID"
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
		Owns(&kustomizev1.Kustomization{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.reconcileCatalogOnSourceSecretChanges)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5,
		}).
		Complete(r)
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=catalogs/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *CatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Catalog{}, r, r.updateStatus())
}

func (r *CatalogReconciler) EnsureDeleted(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	r.Log.Info("attempting to delete catalog", "name", catalog.Name, "namespace", catalog.Namespace)

	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	shouldRequeue, err := r.ensureResourceIsDeleted(ctx, catalog, kustomization)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		r.Log.Info("waiting for kustomization to be deleted", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}

	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	shouldRequeue, err = r.ensureResourceIsDeleted(ctx, catalog, gitRepository)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		r.Log.Info("waiting for git repository to be deleted", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}

	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted Kustomization for catalog: %s/%s", catalog.Namespace, catalog.Name)
	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted GitRepository for catalog: %s/%s", catalog.Namespace, catalog.Name)
	r.Log.Info("catalog deleted", "name", catalog.Name, "namespace", catalog.Namespace)
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *CatalogReconciler) ensureResourceIsDeleted(ctx context.Context, catalog, obj client.Object) (requeue bool, err error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err = r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
			return
		}
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: fetching %s for catalog: %s/%s - %s", kind, catalog.GetNamespace(), catalog.GetName(), err.Error())
		r.Log.Error(err, "failed to get object", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return
	}
	if obj.GetDeletionTimestamp().IsZero() {
		if err = r.Delete(ctx, obj); err != nil {
			r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: deleting %s for catalog: %s/%s - %s", kind, catalog.GetNamespace(), catalog.GetName(), err.Error())
			r.Log.Error(err, "failed to delete object", "name", obj.GetName(), "namespace", obj.GetNamespace())
			return
		}
	} else {
		obj.SetAnnotations(map[string]string{
			greenhouseapis.FluxReconcileRequestAnnotation: time.Now().Format(time.DateTime),
		})
		if err = r.Update(ctx, obj); err != nil {
			if apierrors.IsConflict(err) {
				err = nil
				requeue = true
				return
			}
			return
		}
	}
	requeue = true
	return
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck

	sourceSecret, err := r.getSourceSecret(ctx, catalog)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.ensureGitRepository(ctx, catalog, sourceSecret)
	if err != nil {
		r.Log.Error(err, "failed to ensure git repository for catalog", "name", catalog, "namespace", catalog.Namespace)
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: ensuring GitRepository for catalog: %s/%s - %s", catalog.GetNamespace(), catalog.GetName(), err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.ensureKustomization(ctx, catalog)
	if err != nil {
		r.Log.Error(err, "failed to ensure kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: ensuring Kustomization for catalog: %s/%s - %s", catalog.GetNamespace(), catalog.GetName(), err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *CatalogReconciler) getSourceSecret(ctx context.Context, catalog *greenhousev1alpha1.Catalog) (*corev1.Secret, error) {
	gitSource := catalog.GetCatalogSource()
	if gitSource.SecretName == nil {
		return nil, nil
	}
	secret := &corev1.Secret{}
	secret.SetName(*gitSource.SecretName)
	secret.SetNamespace(catalog.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s for catalog %s/%s: %w", *gitSource.SecretName, catalog.Namespace, catalog.Name, err)
	}
	return secret, nil
}

func (r *CatalogReconciler) ensureGitRepository(ctx context.Context, catalog *greenhousev1alpha1.Catalog, secret *corev1.Secret) error {
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
	authProvider := genericAuthProvider

	if secret != nil {
		if _, ok := secret.Data[githubAppIDKey]; ok {
			authProvider = githubAppAuthProvider
		}
	}

	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)

	result, err := controllerutil.CreateOrPatch(ctx, r.Client, gitRepository, func() error {
		gitRepository.Spec = sourcev1.GitRepositorySpec{
			URL:       gitSource.URL,
			Interval:  metav1.Duration{Duration: flux.DefaultInterval},
			Reference: gitReference,
			Provider:  authProvider,
		}
		if secret != nil {
			gitRepository.Spec.SecretRef = &fluxmeta.LocalObjectReference{Name: *gitSource.SecretName}
		}
		return controllerutil.SetControllerReference(catalog, gitRepository, r.Scheme)
	})
	if err != nil {
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: GitRepository %s - %s", gitRepository.Name, err.Error())
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		r.Log.Info("created git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultUpdated:
		r.Log.Info("updated git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
	case controllerutil.OperationResultNone:
		r.Log.Info("No changes to catalog git repository", "name", gitRepository.Name, "namespace", gitRepository.Namespace)
	default:
		r.Log.Info("result is unknown for catalog git repository", "name", catalog.Name, "namespace", catalog.Namespace, "result", result)
	}
	return nil
}

func (r *CatalogReconciler) ensureKustomization(ctx context.Context, catalog *greenhousev1alpha1.Catalog) error {
	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	gitRepository := &sourcev1.GitRepository{}
	gitRepository.SetName(catalog.Name)
	gitRepository.SetNamespace(catalog.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(gitRepository), gitRepository)
	if err != nil {
		return err
	}
	ggvk := gitRepository.GroupVersionKind()
	kuz := flux.NewKustomizationSpecBuilder(r.Log)
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
	result, err := controllerutil.CreateOrPatch(ctx, r.Client, kustomization, func() error {
		kustomization.Spec = kustomizationSpec
		return controllerutil.SetControllerReference(catalog, kustomization, r.Scheme)
	})
	if err != nil {
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: Kustomization %s - %s", kustomization.Name, err.Error())
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

func (r *CatalogReconciler) updateStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		catalog := object.(*greenhousev1alpha1.Catalog) //nolint:errcheck
		catalogReady := greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, "Catalog reconciliation in progress")
		key := types.NamespacedName{
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
		}
		gitRepository := &sourcev1.GitRepository{}

		err := r.Get(ctx, key, gitRepository)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				catalog.SetCondition(catalogReady)
				return
			}
			r.Log.Error(err, "catalog status update failed to get git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return
		}
		kustomization := &kustomizev1.Kustomization{}
		err = r.Get(ctx, key, kustomization)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				catalog.SetCondition(catalogReady)
				return
			}
			r.Log.Error(err, "catalog status update failed to get kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return
		}

		gitReady := meta.FindStatusCondition(gitRepository.Status.Conditions, string(greenhousemetav1alpha1.ReadyCondition))
		kuzReady := meta.FindStatusCondition(kustomization.Status.Conditions, string(greenhousemetav1alpha1.ReadyCondition))

		if gitReady != nil && kuzReady != nil {
			overallReady := gitReady.Status == metav1.ConditionTrue && kuzReady.Status == metav1.ConditionTrue
			readyMsg := fmt.Sprintf("GitRepository Ready: %s, Kustomization Ready: %s", gitReady.Message, kuzReady.Message)
			if overallReady {
				catalogReady = greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogReadyReason, readyMsg)
				r.recorder.Eventf(catalog, corev1.EventTypeNormal, string(greenhousev1alpha1.CatalogReadyReason), "Catalog Ready - %s", readyMsg)
			} else {
				catalogReady = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, readyMsg)
				if gitReady.Reason != fluxmeta.ProgressingReason || kuzReady.Reason != fluxmeta.ProgressingReason {
					r.recorder.Eventf(catalog, corev1.EventTypeWarning, string(greenhousev1alpha1.CatalogNotReadyReason), "Catalog Not Ready - %s", readyMsg)
				}
			}
		}
		catalog.SetCondition(catalogReady)
	}
}

func (r *CatalogReconciler) reconcileCatalogOnSourceSecretChanges(ctx context.Context, obj client.Object) []ctrl.Request {
	labels := obj.GetLabels()
	if labels == nil {
		return nil
	}
	catalogName, ok := labels[greenhouseapis.SecretManagedByCatalogLabel]
	if !ok {
		return nil
	}
	// find the catalog in the same namespace as the secret
	catalog := &greenhousev1alpha1.Catalog{}
	err := r.Get(ctx, types.NamespacedName{Name: catalogName, Namespace: obj.GetNamespace()}, catalog)
	if client.IgnoreNotFound(err) != nil {
		r.Log.Error(err, "failed to get catalog for secret", "namespace", obj.GetNamespace(), "catalog", catalogName)
		return nil
	}
	return []ctrl.Request{
		{NamespacedName: types.NamespacedName{Name: catalog.Name, Namespace: catalog.Namespace}},
	}
}
