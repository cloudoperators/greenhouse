package plugin

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// FluxReconciler reconciles pluginpresets and plugins and translates them into Flux resources
type FluxReconciler struct {
	client.Client
	KubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

// Greenhouse related RBAC rules for the FluxReconciler
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status;,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list;watch

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
func (r *FluxReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.kubeClientOpts = []clientutil.KubeClientOption{
		clientutil.WithRuntimeOptions(r.KubeRuntimeOpts),
		clientutil.WithPersistentConfig(),
	}

	// index Plugins by the ClusterName field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.Plugin{}, greenhouseapis.PluginClusterNameField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		plugin, ok := rawObj.(*greenhousev1alpha1.Plugin)
		if plugin.Spec.ClusterName == "" || !ok {
			return nil
		}
		return []string{plugin.Spec.ClusterName}
	}); err != nil {
		return err
	}

	labelSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      deliveryToolLabel,
				Operator: metav1.LabelSelectorOpExists,
			},
			{
				Key:      deliveryToolLabel,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{deliveryToolFlux},
			},
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
				workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](30*time.Second, 1*time.Hour),
				&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)}),
			MaxConcurrentReconciles: 3,
		}).
		For(&greenhousev1alpha1.Plugin{}, builder.WithPredicates(
			clientutil.LabelSelectorPredicate(labelSelector),
		)).
		// If a PluginDefinition was changed, reconcile relevant Plugins.
		Watches(&greenhousev1alpha1.PluginDefinition{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForPluginDefinition),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForCluster)).
		Watches(&greenhousev1alpha1.Team{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsInNamespace), builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *FluxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Plugin{}, r, r.setConditions())
}

func (r *FluxReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		_, ok := resource.(*greenhousev1alpha1.Plugin)
		if !ok {
			logger.Error(errors.New("resource is not a Plugin"), "status setup failed")
			return
		}

		//readyCondition := computeReadyCondition(plugin.Status.StatusConditions)
		//plugin.SetCondition(readyCondition)
	}
}

func (r *FluxReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *FluxReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin := resource.(*greenhousev1alpha1.Plugin) //nolint:errcheck

	// Check if the deliveryToolLabel label exists and has the value "flux"
	if value, ok := plugin.Labels[deliveryToolLabel]; !ok || value != deliveryToolFlux {
		return ctrl.Result{}, "", nil
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *FluxReconciler) enqueueAllPluginsForPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client, client.MatchingLabels{greenhouseapis.LabelKeyPluginDefinition: o.GetName()})
}

// enqueueAllPluginsForCluster enqueues all Plugins which have .spec.clusterName set to the name of the given Cluster.
func (r *FluxReconciler) enqueueAllPluginsForCluster(ctx context.Context, o client.Object) []ctrl.Request {
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(greenhouseapis.PluginClusterNameField, o.GetName()),
		Namespace:     o.GetNamespace(),
	}
	return listPluginsAsReconcileRequests(ctx, r.Client, listOpts)
}

func (r *FluxReconciler) enqueueAllPluginsInNamespace(ctx context.Context, o client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client, client.InNamespace(o.GetNamespace()))
}
