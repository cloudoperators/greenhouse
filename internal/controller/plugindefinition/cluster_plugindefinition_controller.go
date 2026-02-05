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
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type ClusterPluginDefinitionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder events.EventRecorder
}

func (r *ClusterPluginDefinitionReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.recorder = mgr.GetEventRecorder(name)
	return setupManagerBuilder(
		mgr,
		name,
		&greenhousev1alpha1.ClusterPluginDefinition{},
		r.helmRepositoryEventHandler,
	).Complete(r)
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories, verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=clusterplugindefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=greenhouse.sap,resources=clusterplugindefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=greenhouse.sap,resources=clusterplugindefinitions/finalizers,verbs=get;create;update;patch;delete

func (r *ClusterPluginDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.ClusterPluginDefinition{}, r, r.setConditions())
}

func (r *ClusterPluginDefinitionReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		pluginDef := resource.(*greenhousev1alpha1.ClusterPluginDefinition) //nolint:errcheck
		setReadyCondition(pluginDef)
	}
}

func (r *ClusterPluginDefinitionReconciler) helmRepositoryEventHandler(_ context.Context, obj client.Object) []ctrl.Request {
	return enqueueOwnersForHelmRepository(obj, greenhousev1alpha1.ClusterPluginDefinitionKind)
}

func (r *ClusterPluginDefinitionReconciler) EnsureCreated(ctx context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	clusterDef := obj.(*greenhousev1alpha1.ClusterPluginDefinition) //nolint:errcheck

	initializeConditions(clusterDef, greenhousemetav1alpha1.ReadyCondition, greenhousev1alpha1.HelmChartReadyCondition)

	if clusterDef.Spec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in ClusterPluginDefinition, skipping HelmRepository creation")
		r.recorder.Eventf(clusterDef, nil, corev1.EventTypeNormal, "Skipped", "reconciling ClusterPluginDefinition", "Skipped HelmRepository creation")
		return ctrl.Result{}, lifecycle.Success, nil
	}

	h := &helmer{
		k8sClient:     r.Client,
		recorder:      r.recorder,
		pluginDef:     clusterDef,
		namespaceName: flux.HelmRepositoryDefaultNamespace,
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

func (r *ClusterPluginDefinitionReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *ClusterPluginDefinitionReconciler) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
