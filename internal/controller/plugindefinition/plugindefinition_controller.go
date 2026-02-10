// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// PluginDefinitionReconciler reconciles a PluginDefinition object.
type PluginDefinitionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder events.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *PluginDefinitionReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorder(name)

	return setupManagerBuilder(
		mgr,
		name,
		&greenhousev1alpha1.PluginDefinition{},
		r.helmRepositoryEventHandler,
	).Complete(r)
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions/finalizers,verbs=get;create;update;patch;delete

func (r *PluginDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.PluginDefinition{}, r, r.setConditions())
}

func (r *PluginDefinitionReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		pluginDef := resource.(*greenhousev1alpha1.PluginDefinition) //nolint:errcheck
		setReadyCondition(pluginDef)
	}
}

func (r *PluginDefinitionReconciler) helmRepositoryEventHandler(_ context.Context, obj client.Object) []ctrl.Request {
	return enqueueOwnersForHelmRepository(obj, greenhousev1alpha1.PluginDefinitionKind)
}

func (r *PluginDefinitionReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginDef := obj.(*greenhousev1alpha1.PluginDefinition) //nolint:errcheck

	initializeConditions(pluginDef, greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.HelmChartReadyCondition)

	if pluginDef.Spec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRepository creation", "name", pluginDef.Name)
		r.recorder.Eventf(pluginDef, nil, corev1.EventTypeNormal, "Skipped", "reconciling PluginDefinition", "Skipped HelmRepository creation")
		return ctrl.Result{}, lifecycle.Success, nil
	}

	h := &helmer{
		k8sClient:     r.Client,
		recorder:      r.recorder,
		pluginDef:     pluginDef,
		namespaceName: pluginDef.Namespace,
	}

	helmRepo, err := h.createUpdateHelmRepository(ctx)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	helmChart, err := h.createUpdateHelmChart(ctx, helmRepo)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	h.setHelmChartReadyCondition(ctx, helmChart)

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginDefinitionReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginDefinitionReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
