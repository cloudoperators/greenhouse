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
	sourcev2 "github.com/fluxcd/source-watcher/api/v2/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
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
		For(&greenhousev1alpha1.Catalog{}, builder.WithPredicates(lifecycle.IgnoreStatusUpdatePredicate())).
		Owns(&sourcev1.GitRepository{}).
		Owns(&sourcev2.ArtifactGenerator{}).
		Owns(&kustomizev1.Kustomization{}).
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
// +kubebuilder:rbac:groups=source.extensions.fluxcd.io,resources=artifactgenerators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *CatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Catalog{}, r, r.updateStatus())
}

func (r *CatalogReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck

	if len(catalog.Spec.Source.Resources) == 0 {
		r.Log.Info("no resources specified for catalog, skipping reconciliation", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{}, lifecycle.Pending, nil
	}

	err := r.ensureGitRepository(ctx, catalog)
	if err != nil {
		r.Log.Error(err, "failed to ensure git repository for catalog", "name", catalog, "namespace", catalog.Namespace)
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: ensuring GitRepository for catalog: %s/%s - %s", catalog.GetNamespace(), catalog.GetName(), err.Error())
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.ensureArtifactGenerated(ctx, catalog)
	if err != nil {
		r.Log.Error(err, "failed to ensure artifact generator for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
		r.recorder.Eventf(catalog, corev1.EventTypeWarning, "Error", "Error: ensuring ArtifactGenerator for catalog: %s/%s - %s", catalog.GetNamespace(), catalog.GetName(), err.Error())
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

func (r *CatalogReconciler) EnsureDeleted(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	catalog := obj.(*greenhousev1alpha1.Catalog) //nolint:errcheck
	r.Log.Info("attempting to delete catalog", "name", catalog.Name, "namespace", catalog.Namespace)

	artifact := &sourcev2.ArtifactGenerator{}
	artifact.SetName(catalog.Name)
	artifact.SetNamespace(catalog.Namespace)
	shouldRequeue, err := r.ensureResourceIsDeleted(ctx, catalog, artifact)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
		r.Log.Info("waiting for artifact to be deleted", "name", catalog.Name, "namespace", catalog.Namespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, lifecycle.Pending, nil
	}

	kustomization := &kustomizev1.Kustomization{}
	kustomization.SetName(catalog.Name)
	kustomization.SetNamespace(catalog.Namespace)
	shouldRequeue, err = r.ensureResourceIsDeleted(ctx, catalog, kustomization)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if shouldRequeue {
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

	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted Artifact for catalog: %s/%s", catalog.Namespace, catalog.Name)
	r.recorder.Eventf(catalog, corev1.EventTypeNormal, "Deleted", "Deleted GitRepository for catalog: %s/%s", catalog.Namespace, catalog.Name)
	r.Log.Info("catalog deleted", "name", catalog.Name, "namespace", catalog.Namespace)
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
			if errors.IsConflict(err) {
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

func (r *CatalogReconciler) updateStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		catalog := object.(*greenhousev1alpha1.Catalog) //nolint:errcheck
		catalog.SetConditionsUnknown()

		key := types.NamespacedName{
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
		}
		gitRepository := &sourcev1.GitRepository{}
		err := r.Get(ctx, key, gitRepository)
		if err != nil {
			catalog.SetConditionsFalse(err.Error())
			r.Log.Error(err, "catalog status update failed to get git repository for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return
		}

		artifact := &sourcev2.ArtifactGenerator{}
		err = r.Get(ctx, key, artifact)
		if err != nil {
			catalog.SetConditionsFalse(err.Error())
			r.Log.Error(err, "catalog status update failed to get artifact generator for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return
		}

		kustomization := &kustomizev1.Kustomization{}
		err = r.Get(ctx, key, kustomization)
		if err != nil {
			catalog.SetConditionsFalse(err.Error())
			r.Log.Error(err, "catalog status update failed to get kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return
		}

		secretErred := catalog.FindCondition(greenhousev1alpha1.CatalogSourceSecretErrorCondition)
		if secretErred != nil {
			catalog.SetConditionsFalse(secretErred.Message)
			return
		}

		gitReady := meta.FindStatusCondition(gitRepository.Status.Conditions, string(greenhousemetav1alpha1.ReadyCondition))
		artifactReady := meta.FindStatusCondition(artifact.Status.Conditions, string(greenhousemetav1alpha1.ReadyCondition))
		kuzReady := meta.FindStatusCondition(kustomization.Status.Conditions, string(greenhousemetav1alpha1.ReadyCondition))

		if gitReady != nil && kuzReady != nil && artifactReady != nil && secretErred == nil {
			overallReady := gitReady.Status == metav1.ConditionTrue &&
				artifactReady.Status == metav1.ConditionTrue &&
				kuzReady.Status == metav1.ConditionTrue

			if gitReady.Status == metav1.ConditionTrue {
				catalog.SetTrueCondition(greenhousev1alpha1.CatalogSourceReadyCondition, greenhousev1alpha1.CatalogSourceReadyReason, fmt.Sprintf("GitRepository Ready: %s", gitReady.Message))
			} else {
				catalog.SetFalseCondition(greenhousev1alpha1.CatalogSourceReadyCondition, greenhousev1alpha1.CatalogSourceNotReadyReason, fmt.Sprintf("GitRepository Not Ready: %s", gitReady.Message))
			}

			if artifactReady.Status == metav1.ConditionTrue {
				catalog.SetTrueCondition(greenhousev1alpha1.CatalogArtifactReadyCondition, greenhousev1alpha1.CatalogArtifactReadyReason, fmt.Sprintf("Artifact Ready: %s", artifactReady.Message))
			} else {
				catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogArtifactNotReadyReason, fmt.Sprintf("Artifact Not Ready: %s", artifactReady.Message))
			}

			if kuzReady.Status == metav1.ConditionTrue {
				catalog.SetTrueCondition(greenhousev1alpha1.CatalogResourcesReadyCondition, greenhousev1alpha1.CatalogResourcesReadyReason, fmt.Sprintf("Resources Applied: %s", kuzReady.Message))
			} else {
				catalog.SetFalseCondition(greenhousev1alpha1.CatalogResourcesReadyCondition, greenhousev1alpha1.CatalogResourcesNotReadyReason, fmt.Sprintf("Resources Not Applied: %s", kuzReady.Message))
			}

			readyMsg := fmt.Sprintf("GitRepository Ready: %s, Artifact Ready: %s, Resources Applied: %s", gitReady.Message, artifactReady.Message, kuzReady.Message)
			if overallReady {
				catalog.SetTrueCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogReadyReason, readyMsg)
				r.recorder.Eventf(catalog, corev1.EventTypeNormal, string(greenhousev1alpha1.CatalogReadyReason), "Catalog Ready - %s", readyMsg)
			} else {
				catalog.SetFalseCondition(greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.CatalogNotReadyReason, readyMsg)
				if gitReady.Reason != fluxmeta.ProgressingReason || kuzReady.Reason != fluxmeta.ProgressingReason || artifactReady.Reason != fluxmeta.ProgressingReason {
					r.recorder.Eventf(catalog, corev1.EventTypeWarning, string(greenhousev1alpha1.CatalogNotReadyReason), "Catalog Not Ready - %s", readyMsg)
				}
			}
		}
	}
}

func getKey(catalog *greenhousev1alpha1.Catalog) types.NamespacedName {
	return types.NamespacedName{Name: catalog.Name, Namespace: catalog.Namespace}
}
