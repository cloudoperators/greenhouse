// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/flux"
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

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginDefinition{}).
		Owns(&sourcev1.HelmChart{}, builder.WithPredicates(clientutil.PredicateIgnoreDeletingResources())).Complete(r)
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts/finalizers,verbs=get;create;update;patch;delete
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

func (r *PluginDefinitionReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginDef := obj.(*greenhousev1alpha1.PluginDefinition) //nolint:errcheck

	initializeConditions(pluginDef, greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.HelmChartReadyCondition)

	if pluginDef.Spec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRepository creation", "name", pluginDef.Name)
		r.recorder.Eventf(pluginDef, nil, corev1.EventTypeNormal, "Skipped", "reconciling PluginDefinition", "Skipped HelmRepository creation")
		return ctrl.Result{}, lifecycle.Success, nil
	}

	// Check if this is a local file path (used in tests) - skip Flux HelmRepository creation
	if flux.IsLocalFilePath(pluginDef.Spec.HelmChart.Repository) {
		log.FromContext(ctx).Info("Local file path detected for HelmChart, skipping Flux resource creation", "path", pluginDef.Spec.HelmChart.Repository)
		r.recorder.Eventf(pluginDef, nil, corev1.EventTypeNormal, "Skipped", "reconciling PluginDefinition", "Skipped Flux resources for local chart path: %s", pluginDef.Spec.HelmChart.Repository)
		// Set HelmChartReady condition to True since local paths are always available
		pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmChartReadyCondition, "", "Local chart path is available"))
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
