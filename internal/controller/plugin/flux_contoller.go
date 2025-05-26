// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	"golang.org/x/time/rate"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	helmcontroller "github.com/fluxcd/helm-controller/api/v2"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/flux"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const (
	defaultNameSpace = "greenhouse"
	maxHistory       = 10
	secretKind       = "Secret"
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
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts/status,verbs=get
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories/status,verbs=get
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
		plugin, ok := resource.(*greenhousev1alpha1.Plugin)
		if !ok {
			logger.Error(errors.New("resource is not a Plugin"), "status setup failed")
			return
		}

		readyCondition := computeReadyCondition(plugin.Status.StatusConditions)
		plugin.SetCondition(readyCondition)
	}
}

func (r *FluxReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *FluxReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin, ok := resource.(*greenhousev1alpha1.Plugin)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("resource is not a Plugin")
	}
	var nS string

	// Check if the deliveryToolLabel label exists and has the value "flux"
	if value, ok := plugin.Labels[deliveryToolLabel]; !ok || value != deliveryToolFlux {
		return ctrl.Result{}, "", nil
	}

	initPluginStatus(plugin)

	pluginDef := r.getPluginDef(ctx, plugin)
	if pluginDef == nil {
		return ctrl.Result{}, lifecycle.Failed, errors.New("plugin definition not found")
	}

	if pluginDef.Namespace == "" {
		nS = defaultNameSpace
	} else {
		nS = pluginDef.Namespace
	}
	helmRepository := flux.FindHelmRepositoryByUrl(ctx, r.Client, nS, pluginDef.Spec.HelmChart.Repository)
	if helmRepository == nil {
		return ctrl.Result{}, lifecycle.Failed, errors.New("helm repository not found")
	}

	helmRelease := &helmcontroller.HelmRelease{
		Spec: helmcontroller.HelmReleaseSpec{
			Install: &helmcontroller.Install{
				Remediation: &helmcontroller.InstallRemediation{},
			},
			Upgrade: &helmcontroller.Upgrade{
				Remediation: &helmcontroller.UpgradeRemediation{},
			},
			DriftDetection: &helmcontroller.DriftDetection{},
			Test:           &helmcontroller.Test{},
			KubeConfig:     &meta.KubeConfigReference{},
			Values:         &v1.JSON{},
		},
	}
	helmRelease.Name = plugin.Name
	helmRelease.Namespace = plugin.Namespace

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, helmRelease, func() error {
		helmRelease.Spec.ReleaseName = plugin.Name
		helmRelease.Spec.TargetNamespace = plugin.Spec.ReleaseNamespace
		helmRelease.Spec.Chart = &helmcontroller.HelmChartTemplate{
			Spec: helmcontroller.HelmChartTemplateSpec{
				Chart:    pluginDef.Spec.HelmChart.Name,
				Interval: &metav1.Duration{Duration: 5 * time.Minute},
				Version:  pluginDef.Spec.HelmChart.Version,
				SourceRef: helmcontroller.CrossNamespaceObjectReference{
					Kind:      sourcecontroller.HelmRepositoryKind,
					Name:      helmRepository.Name,
					Namespace: helmRepository.Namespace,
				},
			},
		}
		helmRelease.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmRelease.Spec.Timeout = &metav1.Duration{Duration: 30 * time.Minute}
		helmRelease.Spec.MaxHistory = ptr.To[int](maxHistory)
		helmRelease.Spec.Install.CreateNamespace = true
		helmRelease.Spec.Install.Remediation.Retries = 3
		helmRelease.Spec.Upgrade.Remediation.Retries = 3
		helmRelease.Spec.DriftDetection.Mode = helmcontroller.DriftDetectionEnabled
		helmRelease.Spec.Test.Enable = false
		helmRelease.Spec.KubeConfig.SecretRef = meta.SecretKeyReference{
			Name: plugin.Spec.ClusterName,
			Key:  greenhouseapis.GreenHouseKubeConfigKey,
		}
		helmRelease.Spec.Values = r.addValuestoHelmRelease(plugin, pluginDef)
		helmRelease.Spec.ValuesFrom = r.addValueReferences(plugin)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRelease", "name", helmRelease.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRelease", "name", helmRelease.Name)
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, lifecycle.Success, nil
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

func (r *FluxReconciler) getPluginDef(ctx context.Context, plugin *greenhousev1alpha1.Plugin) *greenhousev1alpha1.PluginDefinition {
	pluginDef := new(greenhousev1alpha1.PluginDefinition)
	if err := r.Get(ctx, types.NamespacedName{Name: plugin.Spec.PluginDefinition}, pluginDef); err != nil {
		log.FromContext(ctx).Error(err, "Unable to find pluginDefinition for ", "plugin", plugin.Name, "namespace", plugin.Namespace)
		return nil
	}
	return pluginDef
}

func (r *FluxReconciler) addValuestoHelmRelease(plugin *greenhousev1alpha1.Plugin, pluginDef *greenhousev1alpha1.PluginDefinition) *v1.JSON {
	jsonValue := make(map[string]any)
	for _, value := range pluginDef.Spec.Options {
		if value.Default != nil {
			defValue, err := json.Marshal(value.Default)
			if err != nil {
				log.FromContext(context.Background()).Error(err, "Unable to marshal default value for plugin", "plugin", plugin.Name)
				continue
			}
			jsonValue[value.Name] = string(defValue)
		}
	}
	for _, value := range plugin.Spec.OptionValues {
		jsonValue[value.Name] = value.Value
	}
	byteValue, err := json.Marshal(jsonValue)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Unable to marshal values for plugin", "plugin", plugin.Name)
		return nil
	}
	return &v1.JSON{Raw: byteValue}
}

func (r *FluxReconciler) addValueReferences(plugin *greenhousev1alpha1.Plugin) []helmcontroller.ValuesReference {
	var valuesFrom []helmcontroller.ValuesReference
	for _, value := range plugin.Spec.OptionValues {
		if value.ValueFrom != nil {
			valuesFrom = append(valuesFrom, helmcontroller.ValuesReference{
				Kind:      secretKind,
				Name:      value.ValueFrom.Secret.Name,
				ValuesKey: value.ValueFrom.Secret.Key,
			})
		}
	}
	return valuesFrom
}
