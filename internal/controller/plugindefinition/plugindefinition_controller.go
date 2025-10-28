// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const (
	pluginDefinitionURLField = "spec.url"
)

// PluginDefinitionReconciler reconciles a PluginDefinition object.
type PluginDefinitionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *PluginDefinitionReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginDefinition{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/finalizers,verbs=get;create;update;patch;delete

func (r *PluginDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.PluginDefinition{}, r, nil)
}

func (r *PluginDefinitionReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginDef := obj.(*greenhousev1alpha1.PluginDefinition) //nolint:errcheck

	if pluginDef.Spec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRepository creation", "name", pluginDef.Name)
		r.recorder.Event(pluginDef, corev1.EventTypeNormal, "Skipped", "Skipped HelmRepository creation")
		return ctrl.Result{}, lifecycle.Success, nil
	}
	err := r.reconcileHelmRepository(ctx, pluginDef)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginDefinitionReconciler) reconcileHelmRepository(ctx context.Context, pluginDef *greenhousev1alpha1.PluginDefinition) error {
	repositoryURL := pluginDef.Spec.HelmChart.Repository
	helmRepository := &sourcecontroller.HelmRepository{}
	helmRepository.SetName(flux.ChartURLToName(repositoryURL))
	helmRepository.SetNamespace(pluginDef.GetNamespace()) // Set organization namespace

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, helmRepository, func() error {
		helmRepository.Spec.Type = flux.GetSourceRepositoryType(repositoryURL)
		helmRepository.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmRepository.Spec.URL = repositoryURL
		return controllerutil.SetOwnerReference(pluginDef, helmRepository, r.Scheme)
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to create or update HelmRepository", "name", helmRepository.Name)
		r.recorder.Eventf(pluginDef, corev1.EventTypeWarning, "Failed", "Failed to create or update HelmRepository %s: %s", helmRepository.Name, err.Error())
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRepository", "name", helmRepository.Name)
		r.recorder.Eventf(pluginDef, corev1.EventTypeNormal, "Created", "Created HelmRepository %s", helmRepository.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRepository", "name", helmRepository.Name)
		r.recorder.Eventf(pluginDef, corev1.EventTypeNormal, "Updated", "Updated HelmRepository %s", helmRepository.Name)
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmRepository", "name", helmRepository.Name)
	}
	return nil
}

func (r *PluginDefinitionReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginDefinitionReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
