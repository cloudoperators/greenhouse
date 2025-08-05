package catalog

import (
	"context"
	"fmt"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const kustomizeCatalogSuffix = "-kustomization"

func (r *CatalogReconciler) buildKustomizationSpec(catalog *greenhousev1alpha1.PluginDefinitionCatalog) (kustomizev1.KustomizationSpec, error) {
	ggvk := r.gitRepository.GroupVersionKind()
	kuz := flux.NewKustomizationSpecBuilder(r.log)
	kuz = kuz.WithSourceRef(ggvk.String(), ggvk.Kind, r.gitRepository.Name, r.gitRepository.Namespace).
		WithCommonMetadata(nil, map[string]string{
			// TODO: add owned by label to catalog resources
			greenhouseapis.LabelKeyCatalog:          catalog.Name,
			greenhouseapis.LabelKeyCatalogNamespace: catalog.Namespace,
		}).
		// TODO: Implement RBAC for SA for restricted catalogs
		// WithServiceAccountName("some-restricted-sa").
		WithTimeout(catalog.Timeout()).
		WithInterval(catalog.Interval())
	if catalog.ResourcePath() != "" {
		kuz.WithPath(catalog.ResourcePath())
	}
	if catalog.IsSuspended() {
		kuz.WithSuspend(catalog.IsSuspended())
	}
	return kuz.Build()
}

func (r *CatalogReconciler) ensureKustomization(catalog *greenhousev1alpha1.PluginDefinitionCatalog) lifecycle.ReconcileRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		kustomization := &kustomizev1.Kustomization{}
		kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
		kustomization.SetNamespace(catalog.Namespace)
		kustomizationSpec, err := r.buildKustomizationSpec(catalog)
		if err != nil {
			catalog.SetKustomizationReadyFalse(greenhousev1alpha1.CatalogKustomizationBuildFailReason, err.Error())
			return lifecycle.Break(), err
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
			catalog.SetKustomizationReadyFalse(greenhousev1alpha1.CatalogKustomizationFailReason, err.Error())
			return lifecycle.Break(), err
		}
		catalog.SetKustomizationReadyUnknown("", "kustomization for catalog is being created or updated")
		return lifecycle.Continue(), nil
	}
}

func (r *CatalogReconciler) ensureKustomizationIsReady(catalog *greenhousev1alpha1.PluginDefinitionCatalog) lifecycle.ReconcileRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if catalog.IsSuspended() {
			r.log.Info("catalog is suspended, skipping readiness check for git repository", "name", catalog.Name, "namespace", catalog.Namespace)
			catalog.SetKustomizationReadyUnknown(greenhousev1alpha1.CatalogSuspendedReason, "kustomization is suspended")
			return lifecycle.Exit(), nil
		}

		kustomization := &kustomizev1.Kustomization{}
		kustomization.SetName(catalog.Name + kustomizeCatalogSuffix)
		kustomization.SetNamespace(catalog.Namespace)
		err := r.Client.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization)
		if err != nil {
			r.log.Error(err, "failed to get kustomization for catalog", "name", catalog.Name, "namespace", catalog.Namespace)
			return lifecycle.Break(), err
		}

		readyCond := getReadyCondition(kustomization.Status.Conditions)
		if readyCond == nil {
			r.log.Info("catalog kustomization resource is not yet ready", "name", kustomization.Name, "namespace", kustomization.Namespace)
			return lifecycle.Requeue(), nil
		}

		if readyCond.Status == metav1.ConditionUnknown {
			catalog.SetKustomizationReadyUnknown(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
			r.log.Info("catalog kustomization resource is in an unknown state, retrying in 10 seconds", "name", kustomization.Name, "namespace", kustomization.Namespace)
			return lifecycle.RequeueAfter(10 * time.Second), nil
		}

		if readyCond.Status == metav1.ConditionFalse {
			catalog.SetKustomizationReadyFalse(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
			return lifecycle.Break(), fmt.Errorf("catalog kustomization %s/%s is not ready: %s", kustomization.Namespace, kustomization.Name, readyCond.Message)
		}

		catalog.Status.KustomizeInventory = kustomization.Status.Inventory
		catalog.SetKustomizationReadyTrue(greenhousemetav1alpha1.ConditionReason(readyCond.Reason), readyCond.Message)
		r.log.Info("kustomization resource is ready", "name", kustomization.Name, "namespace", kustomization.Namespace)

		return lifecycle.Continue(), nil
	}
}
