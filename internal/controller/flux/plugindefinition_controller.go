// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"
	"time"

	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
)

const (
	defautlNameSpace         = "greenhouse"
	pluginDefinitionUrlField = "spec.url"
)

// PluginDefinitionReconciler reconciles plugindefinitions and translates them into Flux resources
type PluginDefinitionReconciler struct {
	client.Client
}

// Greenhouse related RBAC rules for the FluxReconciler
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch

// Flux related RBAC rules for the FluxReconciler
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmrepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *PluginDefinitionReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()

	// index PluginDefinitions by the ClusterName field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &sourcecontroller.HelmRepository{}, pluginDefinitionUrlField, func(rawObj client.Object) []string {
		helmRepository, ok := rawObj.(*sourcecontroller.HelmRepository)
		if helmRepository.Spec.URL == "" || !ok {
			return nil
		}
		return []string{helmRepository.Spec.URL}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
				workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](30*time.Second, 1*time.Hour),
				&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)}),
			MaxConcurrentReconciles: 3,
		}).
		For(&greenhousev1alpha1.PluginDefinition{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(r)
}

// Reconcile reads the PluginDefinition object and makes creates a HelmRepository and a HelmChart flux object
func (r *PluginDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	var nS string

	if err := r.Get(ctx, req.NamespacedName, pluginDefinition); err != nil {
		log.FromContext(ctx).Error(err, "Failed to get pluginDefinition")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	nS = req.Namespace
	if req.Namespace == "" {
		// if the namespace is empty, we use the default namespace
		// preparation for the future when we will use the namespaced plugin definitions
		nS = defautlNameSpace
	}

	// reconcileHelmRepository is creating or updating the helmRepository out form PluginDefinition spec
	if err := r.reconcileHelmRepository(ctx, pluginDefinition, nS); err != nil {
		return ctrl.Result{}, err
	}

	// reconcileHelmChart is creating or update the helmChart out form PluginDefinition spec
	// also creates or updates the helmRepository if it does not exist

	/*
		if err := r.reconcileHelmChart(ctx, pluginDefinition, nS); err != nil {
			return ctrl.Result{}, err
		}
	*/

	return ctrl.Result{RequeueAfter: utils.DefaultRequeueInterval}, nil
}

func (r *PluginDefinitionReconciler) reconcileHelmRepository(ctx context.Context, pluginDef *greenhousev1alpha1.PluginDefinition, nS string) error {
	var helmRepository = new(sourcecontroller.HelmRepository)

	result, err := clientutil.CreateOrPatch(ctx, r.Client, helmRepository, func() error {
		helmRepository.Name, helmRepository.Spec.Type = convertName(pluginDef.Spec.HelmChart.Repository)
		helmRepository.Namespace = nS
		helmRepository.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmRepository.Spec.URL = pluginDef.Spec.HelmChart.Repository
		return controllerutil.SetOwnerReference(pluginDef, helmRepository, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRepository", "name", helmRepository.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRepository", "name", helmRepository.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmRepository", "name", helmRepository.Name)
	}

	return nil
}

func (r *PluginDefinitionReconciler) reconcileHelmChart(ctx context.Context, pluginDef *greenhousev1alpha1.PluginDefinition, nS string) error {
	helmChart := new(sourcecontroller.HelmChart)
	helmRepository := FindHelmRepositoryByUrl(ctx, r.Client, nS, pluginDef.Spec.HelmChart.Repository)

	if helmRepository == nil {
		if err := r.reconcileHelmRepository(ctx, pluginDef, nS); err != nil {
			return err
		}
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, helmChart, func() error {
		helmChart.Name = GenerateChartName(pluginDef)
		helmChart.Namespace = nS
		helmChart.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmChart.Spec.Chart = pluginDef.Spec.HelmChart.Name
		helmChart.Spec.SourceRef = sourcecontroller.LocalHelmChartSourceReference{
			Kind: sourcecontroller.HelmRepositoryKind,
			Name: helmRepository.Name,
		}
		helmChart.Spec.Version = pluginDef.Spec.HelmChart.Version
		helmChart.Spec.ReconcileStrategy = sourcecontroller.ReconcileStrategyChartVersion
		return controllerutil.SetOwnerReference(helmRepository, helmChart, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmChart", "name", helmChart.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmChart", "name", helmChart.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmChart", "name", helmChart.Name)
	}

	return nil
}
