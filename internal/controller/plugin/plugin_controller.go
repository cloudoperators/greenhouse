// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/features"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/util"
)

const helmReleaseSecretType = "helm.sh/release.v1" //nolint:gosec

// PluginReconciler reconciles a Plugin object.
type PluginReconciler struct {
	client.Client
	KubeRuntimeOpts             clientutil.RuntimeOptions
	kubeClientOpts              []clientutil.KubeClientOption
	ExpressionEvaluationEnabled bool
	DefaultDeploymentTool       *string
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusterplugindefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status;,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch;update
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/finalizers,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=helmcharts,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=ocirepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// These broad permissions are required as the controller manages Helm charts which contain arbitrary Kubernetes resources.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*

// SetupWithManager sets up the controller with the Manager.
func (r *PluginReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
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

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
				workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](30*time.Second, 1*time.Hour),
				&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)}),
			MaxConcurrentReconciles: 10,
		}).
		For(&greenhousev1alpha1.Plugin{}).
		// Reconcile on owned flux HelmRelease changes.
		Owns(&helmv2.HelmRelease{}).
		// If the release was (manually) modified the secret would have been modified. Reconcile it.
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(enqueuePluginForReleaseSecret),
			builder.WithPredicates(clientutil.PredicateFilterBySecretTypes(helmReleaseSecretType)),
		).
		// If a ClusterPluginDefinition was changed, reconcile relevant Plugins.
		Watches(
			&greenhousev1alpha1.ClusterPluginDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForClusterPluginDefinition),
		).
		// If a PluginDefinition was changed, reconcile relevant Plugins.
		Watches(
			&greenhousev1alpha1.PluginDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForPluginDefinition),
		).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(
			&greenhousev1alpha1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForCluster),
		).
		Watches(
			&greenhousev1alpha1.Team{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsInNamespace),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}

func (r *PluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Plugin{}, r, r.setConditions())
}

// getDeploymentTool determines the deployment tool to use for a plugin.
// It applies the precedence: label override > feature flag default > "helm" fallback.
func (r *PluginReconciler) getDeploymentTool(plugin *greenhousev1alpha1.Plugin) string {
	// Check if explicitly set via label
	if label, exists := plugin.GetLabels()[greenhouseapis.GreenhouseHelmDeliveryToolLabel]; exists {
		return label
	}

	// Use feature flag default if configured
	if r.DefaultDeploymentTool != nil {
		return *r.DefaultDeploymentTool
	}

	// Final fallback
	return features.DefaultDeploymentToolValue
}

// suspendFluxResourcesIfNeeded suspends any existing Flux HelmRelease resources for a plugin
// when switching from Flux to Helm deployment.
func (r *PluginReconciler) suspendFluxResourcesIfNeeded(ctx context.Context, plugin *greenhousev1alpha1.Plugin) error {
	helmRelease := &helmv2.HelmRelease{}
	err := r.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: plugin.Namespace}, helmRelease)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	if helmRelease.Spec.Suspend {
		return nil
	}

	log.FromContext(ctx).Info("suspending Flux HelmRelease as deployment tool switched to helm")
	helmRelease.Spec.Suspend = true
	if err := r.Update(ctx, helmRelease); err != nil {
		return fmt.Errorf("failed to suspend HelmRelease: %w", err)
	}

	return nil
}

func (r *PluginReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		plugin, ok := resource.(*greenhousev1alpha1.Plugin)
		if !ok {
			logger.Error(errors.New("resource is not a Plugin"), "status setup failed")
			return
		}

		var readyCondition greenhousemetav1alpha1.Condition
		deploymentTool := r.getDeploymentTool(plugin)
		if deploymentTool == greenhouseapis.GreenhouseHelmDeliveryToolFlux {
			readyCondition = r.computeReadyConditionFlux(ctx, plugin)
		} else {
			readyCondition = computeReadyCondition(plugin.Status.StatusConditions)
		}
		UpdatePluginReadyMetric(plugin, readyCondition.Status == metav1.ConditionTrue)

		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, plugin)
		util.UpdateOwnedByLabelMissingMetric(plugin, ownerLabelCondition.IsFalse())

		plugin.Status.SetConditions(readyCondition, ownerLabelCondition)

		if err := r.reconcileTechnicalLabels(ctx, plugin); err != nil {
			logger.Error(err, "failed to reconcile technical labels")
		}
	}
}

// reconcileTechnicalLabels ensures the plugin has the correct technical labels based on its status.
func (r *PluginReconciler) reconcileTechnicalLabels(ctx context.Context, plugin *greenhousev1alpha1.Plugin) error {
	_, err := clientutil.Patch(ctx, r.Client, plugin, func() error {
		labels := plugin.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		if plugin.Status.UIApplication != nil {
			labels[greenhouseapis.LabelKeyUIPlugin] = "true"
		} else {
			delete(labels, greenhouseapis.LabelKeyUIPlugin)
		}

		if len(plugin.Status.ExposedServices) > 0 {
			labels[greenhouseapis.LabelKeyPluginExposesService] = "true"
		} else {
			delete(labels, greenhouseapis.LabelKeyPluginExposesService)
		}

		plugin.SetLabels(labels)

		return nil
	})

	return err
}

func (r *PluginReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin := resource.(*greenhousev1alpha1.Plugin) //nolint:errcheck

	deploymentTool := r.getDeploymentTool(plugin)
	if deploymentTool == greenhouseapis.GreenhouseHelmDeliveryToolFlux {
		return r.EnsureFluxDeleted(ctx, plugin)
	}

	// if DeletionPolicy is Retain, skip Helm release deletion
	if plugin.Spec.DeletionPolicy == greenhouseapis.DeletionPolicyRetain {
		log.FromContext(ctx).Info("skipping Helm release deletion due to DeletionPolicy=Retain")
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))
		return ctrl.Result{}, lifecycle.Success, nil
	}

	restClientGetter, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
	if err != nil {
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("cannot access cluster: %s", err.Error())
	}

	isDeleted, err := helm.UninstallHelmRelease(ctx, restClientGetter, plugin)
	if err != nil {
		c := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
		plugin.SetCondition(c)
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonUninstallHelmFailed)
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if !isDeleted {
		// Ensure we're called again for some corner cases esp. where the actual deletion takes unusually long (hooks) yet the watch won't catch it.
		return ctrl.Result{RequeueAfter: time.Minute}, lifecycle.Pending, nil
	}

	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin := resource.(*greenhousev1alpha1.Plugin) //nolint:errcheck
	InitPluginStatus(plugin)

	deploymentTool := r.getDeploymentTool(plugin)
	if deploymentTool == greenhouseapis.GreenhouseHelmDeliveryToolFlux {
		return r.EnsureFluxCreated(ctx, plugin)
	}

	if err := r.suspendFluxResourcesIfNeeded(ctx, plugin); err != nil {
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("failed to suspend flux resources: %w", err)
	}

	restClientGetter, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
	if err != nil {
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("cannot access cluster: %s", err.Error())
	}

	// Check if we should continue with reconciliation or requeue if cluster is scheduled for deletion
	result, err := shouldReconcileOrRequeue(ctx, r.Client, plugin)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	if result != nil {
		return ctrl.Result{RequeueAfter: result.requeueAfter}, lifecycle.Pending, nil
	}

	pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(ctx, r.Client, plugin.Spec.PluginDefinitionRef, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.PluginDefinitionNotFoundReason, err.Error()))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonPluginDefinitionNotFound)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("%s not found: %s", plugin.Spec.PluginDefinitionRef.Kind, err.Error())
	}

	reconcileErr := r.reconcileHelmRelease(ctx, restClientGetter, plugin, *pluginDefinitionSpec)

	// PluginStatus, WorkloadStatus and ChartTest should be reconciled regardless of Helm reconciliation result.
	r.reconcileStatus(ctx, restClientGetter, plugin, *pluginDefinitionSpec, &plugin.Status)

	workloadStatusResult, workloadStatusErr := r.reconcilePluginWorkloadStatus(ctx, restClientGetter, plugin, *pluginDefinitionSpec)

	helmChartTestResult, helmChartTestErr := r.reconcileHelmChartTest(ctx, plugin)

	if reconcileErr != nil {
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("helm reconcile failed: %s", reconcileErr.Error())
	}
	if workloadStatusErr != nil {
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("workload status reconcile failed: %s", workloadStatusErr.Error())
	}
	if helmChartTestErr != nil {
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("helm chart test reconcile failed: %s", helmChartTestErr.Error())
	}
	if workloadStatusResult != nil {
		return ctrl.Result{RequeueAfter: workloadStatusResult.requeueAfter}, lifecycle.Pending, nil
	}
	if helmChartTestResult != nil {
		return ctrl.Result{RequeueAfter: helmChartTestResult.requeueAfter}, lifecycle.Pending, nil
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginReconciler) EnsureSuspended(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, error) {
	plugin := resource.(*greenhousev1alpha1.Plugin) //nolint:errcheck
	deploymentTool := r.getDeploymentTool(plugin)
	if deploymentTool == greenhouseapis.GreenhouseHelmDeliveryToolFlux {
		return r.EnsureFluxSuspended(ctx, plugin)
	}
	return ctrl.Result{}, nil
}

func (r *PluginReconciler) reconcileHelmRelease(
	ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
) error {

	// Not a HelmChart pluginDefinition. Ignore it.
	if pluginDefinitionSpec.HelmChart == nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "PluginDefinition is not backed by HelmChart"))
		return nil
	}

	// Validate before attempting the installation/upgrade.
	// Any error is reflected in the status of the Plugin.
	if _, err := helm.TemplateHelmChartFromPlugin(ctx, r.Client, restClientGetter, pluginDefinitionSpec, plugin); err != nil {
		errorMessage := "Helm template failed: " + err.Error()
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", errorMessage))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonTemplateFailed)
		return errors.New(errorMessage)
	}

	// Check whether the deployed resources match the ones we expect.
	diffObjects, isHelmDrift, err := helm.DiffChartToDeployedResources(ctx, r.Client, restClientGetter, pluginDefinitionSpec, plugin)
	if err != nil {
		errorMessage := "Helm diff failed: " + err.Error()
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", errorMessage))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonDiffFailed)
		return errors.New(errorMessage)
	}

	switch {
	case isHelmDrift: // drift was detected
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmDriftDetectedCondition, "", ""))
		log.FromContext(ctx).Info("drift between deployed resources and manifest detected", "resources", diffObjects.String())
	case len(diffObjects) > 0: // diff detected
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmDriftDetectedCondition, "", ""))
		log.FromContext(ctx).Info("diff between deployed release and manifest detected", "resources", diffObjects.String())
	default: // no diff detected and no drift detected
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmDriftDetectedCondition, "", ""))
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "Release for plugin is up-to-date"))

		// TODO: remove unnecessary log?
		log.FromContext(ctx).Info("release for plugin is up-to-date")
		return nil
	}

	plugin.Status.HelmReleaseStatus.Diff = diffObjects.String()

	if err := helm.InstallOrUpgradeHelmChartFromPlugin(ctx, r.Client, restClientGetter, pluginDefinitionSpec, plugin); err != nil {
		errorMessage := "Helm install/upgrade failed: " + err.Error()
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", errorMessage))
		return errors.New(errorMessage)
	}

	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
		greenhousev1alpha1.HelmReconcileFailedCondition, "", "Helm install/upgrade successful"))
	util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultSuccess, util.MetricReasonEmpty)
	return nil
}

func (r *PluginReconciler) reconcileStatus(ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	pluginStatus *greenhousev1alpha1.PluginStatus,
) {

	var (
		pluginVersion   string
		exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
		releaseStatus   = &greenhousev1alpha1.HelmReleaseStatus{
			Status:        "unknown",
			FirstDeployed: metav1.Time{},
			LastDeployed:  metav1.Time{},
			Diff:          pluginStatus.HelmReleaseStatus.Diff,
		}
	)

	// Collect status from the Helm release.
	helmRelease, err := helm.GetReleaseForHelmChartFromPlugin(ctx, restClientGetter, plugin)
	if err == nil {
		serviceList, err := getAllExposedServicesForPlugin(restClientGetter, helmRelease, plugin)
		if err != nil {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get exposed services: "+err.Error()))
		} else {
			exposedServices = serviceList
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.StatusUpToDateCondition, "", ""))
		}

		// Get the release status.
		if latestReleaseInfo := helmRelease.Info; latestReleaseInfo != nil {
			releaseStatus.Status = latestReleaseInfo.Status.String()
			releaseStatus.FirstDeployed = metav1.NewTime(latestReleaseInfo.FirstDeployed.Time)
			releaseStatus.LastDeployed = metav1.NewTime(latestReleaseInfo.LastDeployed.Time)
			if latestReleaseInfo.Status == release.StatusDeployed {
				pluginVersion = latestReleaseInfo.Description
			}
			if plugin.Spec.OptionValues != nil {
				checksum, err := helm.CalculatePluginOptionChecksum(ctx, r.Client, plugin)
				if err == nil {
					releaseStatus.PluginOptionChecksum = checksum
				} else {
					releaseStatus.PluginOptionChecksum = ""
				}
			}
		}
	} else {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get Helm release: "+err.Error()))
	}

	var (
		uiApplication      *greenhousev1alpha1.UIApplicationReference
		helmChartReference *greenhousev1alpha1.HelmChartReference
	)
	// Ensure the status is always reported.
	uiApplication = pluginDefinitionSpec.UIApplication
	// only set the helm chart reference if the pluginVersion matches the pluginDefinition version or the release status is unknown
	if pluginVersion == pluginDefinitionSpec.Version || releaseStatus.Status == "unknown" {
		helmChartReference = pluginDefinitionSpec.HelmChart
	} else {
		helmChartReference = plugin.Status.HelmChart
	}

	pluginStatus.HelmReleaseStatus = releaseStatus
	pluginStatus.Version = pluginVersion
	pluginStatus.UIApplication = uiApplication
	pluginStatus.HelmChart = helmChartReference
	pluginStatus.Weight = pluginDefinitionSpec.Weight
	pluginStatus.Description = pluginDefinitionSpec.Description
	pluginStatus.ExposedServices = exposedServices
}

// enqueueAllPluginsForCluster enqueues all Plugins which have .spec.clusterName set to the name of the given Cluster.
func (r *PluginReconciler) enqueueAllPluginsForCluster(ctx context.Context, o client.Object) []ctrl.Request {
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(greenhouseapis.PluginClusterNameField, o.GetName()),
		Namespace:     o.GetNamespace(),
	}
	return ListPluginsAsReconcileRequests(ctx, r.Client, listOpts)
}

func (r *PluginReconciler) enqueueAllPluginsInNamespace(ctx context.Context, o client.Object) []ctrl.Request {
	return ListPluginsAsReconcileRequests(ctx, r.Client, client.InNamespace(o.GetNamespace()))
}

func (r *PluginReconciler) enqueueAllPluginsForClusterPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
	return ListPluginsAsReconcileRequests(
		ctx,
		r.Client,
		client.MatchingLabels{greenhouseapis.LabelKeyClusterPluginDefinition: o.GetName()},
	)
}

func (r *PluginReconciler) enqueueAllPluginsForPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
	return ListPluginsAsReconcileRequests(
		ctx,
		r.Client,
		client.MatchingLabels{greenhouseapis.LabelKeyPluginDefinition: o.GetName()},
		client.InNamespace(o.GetNamespace()),
	)
}

func ListPluginsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var pluginList = new(greenhousev1alpha1.PluginList)
	if err := c.List(ctx, pluginList, listOpts...); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to list Plugins for reconcile requests")
		return nil
	}
	res := make([]ctrl.Request, len(pluginList.Items))
	for idx, plugin := range pluginList.Items {
		namespacedName := types.NamespacedName{Namespace: plugin.GetNamespace(), Name: plugin.GetName()}
		res[idx] = ctrl.Request{NamespacedName: namespacedName}
	}
	return res
}

func enqueuePluginForReleaseSecret(_ context.Context, o client.Object) []ctrl.Request {
	secret, ok := o.(*corev1.Secret)
	if !ok || secret.Type != helmReleaseSecretType {
		return nil
	}
	if name, ok := secret.GetLabels()["name"]; ok {
		return []ctrl.Request{{NamespacedName: types.NamespacedName{Namespace: o.GetNamespace(), Name: name}}}
	}
	return nil
}

// getExposedServicesForPluginFromHelmRelease returns a map of exposed services for a plugin from a Helm release.
// The exposed services are collected from Helm release manifest and not from the template to make sure they are deployed.
func getExposedServicesForPluginFromHelmRelease(restClientGetter genericclioptions.RESTClientGetter, helmRelease *release.Release, plugin *greenhousev1alpha1.Plugin) (map[string]greenhousev1alpha1.Service, error) {
	// Collect exposed services from the manifest.
	exposedServiceList, err := helm.ObjectMapFromRelease(restClientGetter, helmRelease, &helm.ManifestObjectFilter{
		APIVersion: "v1",
		Kind:       "Service",
		Annotations: map[string]string{
			greenhouseapis.AnnotationKeyExpose: "true",
		},
	})
	if err != nil {
		return nil, err
	}
	var exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
	if len(exposedServiceList) == 0 {
		return exposedServices, nil
	}
	if plugin.Spec.ClusterName == "" {
		return nil, errors.New("plugin does not have ClusterName")
	}
	for _, svc := range exposedServiceList {
		svcPort, err := getPortForExposedService(svc.Object)
		if err != nil {
			return nil, err
		}
		namespace := svc.Namespace
		if namespace == "" {
			namespace = helmRelease.Namespace // default namespace to release namespace
		}
		exposedURL := common.URLForExposedServiceInPlugin(svc.Name, plugin)
		exposedServices[exposedURL] = greenhousev1alpha1.Service{
			Namespace: namespace,
			Name:      svc.Name,
			Protocol:  svcPort.AppProtocol,
			Port:      svcPort.Port,
			Type:      greenhousev1alpha1.ServiceTypeService,
		}
	}
	return exposedServices, nil
}

// getExposedIngressesForPluginFromHelmRelease discovers ingress resources from a Helm release that are marked for exposure.
// It searches for ingresses with the annotation "greenhouse.sap/expose": "true" and extracts their URLs.
//
// For each discovered ingress:
// - Automatically detects HTTPS protocol based on TLS configuration
// - Selects the host using greenhouse.sap/exposed-host annotation or defaults to the first host rule
// - Constructs complete URLs with protocol (http:// or https://)
// - Returns them as Service entries with type ServiceTypeIngress (port set to 0 as not applicable)
//
// The function filters ingresses from the Helm release manifest (not templates) to ensure they are actually deployed.
func getExposedIngressesForPluginFromHelmRelease(restClientGetter genericclioptions.RESTClientGetter, helmRelease *release.Release) (map[string]greenhousev1alpha1.Service, error) {
	exposedIngressList, err := helm.ObjectMapFromRelease(restClientGetter, helmRelease, &helm.ManifestObjectFilter{
		APIVersion: "v1",
		Kind:       "Ingress",
		Annotations: map[string]string{
			greenhouseapis.AnnotationKeyExpose: "true",
		},
	})
	if err != nil {
		return nil, err
	}

	var exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
	for _, ingress := range exposedIngressList {
		fullURL, err := getURLForExposedIngress(ingress.Object)
		if err != nil {
			return nil, err
		}

		namespace := ingress.Namespace
		if namespace == "" {
			namespace = helmRelease.Namespace
		}

		exposedServices[fullURL] = greenhousev1alpha1.Service{
			Namespace: namespace,
			Name:      ingress.Name,
			Type:      greenhousev1alpha1.ServiceTypeIngress,
		}
	}
	return exposedServices, nil
}

// getAllExposedServicesForPlugin aggregates all exposed services and ingresses for a plugin from its Helm release.
// This function combines two types of exposures into a unified map for Plugin.status.exposedServices:
//
// 1. Services (ServiceTypeService): Exposed via service-proxy for internal routing
//   - Discovered from services with "greenhouse.sap/expose" annotation
//   - URLs are generated using the service-proxy pattern
//
// 2. Ingresses (ServiceTypeIngress): Exposed directly via external URLs
//   - Discovered from ingresses with "greenhouse.sap/expose" annotation
//   - URLs are the actual ingress hostnames with proper protocol (http/https)
//   - Protocol automatically detected from TLS configuration
//
// If either service or ingress discovery encounters an error, the entire operation fails.
func getAllExposedServicesForPlugin(restClientGetter genericclioptions.RESTClientGetter, helmRelease *release.Release, plugin *greenhousev1alpha1.Plugin) (map[string]greenhousev1alpha1.Service, error) {
	var errorMessages []string
	exposedServices := make(map[string]greenhousev1alpha1.Service)

	serviceList, err := getExposedServicesForPluginFromHelmRelease(restClientGetter, helmRelease, plugin)
	if err != nil {
		errorMessages = append(errorMessages, "services: "+err.Error())
	} else {
		maps.Copy(exposedServices, serviceList)
	}

	ingressList, err := getExposedIngressesForPluginFromHelmRelease(restClientGetter, helmRelease)
	if err != nil {
		errorMessages = append(errorMessages, "ingresses: "+err.Error())
	} else {
		maps.Copy(exposedServices, ingressList)
	}

	if len(errorMessages) > 0 {
		return nil, fmt.Errorf("failed to get exposed resources: %s", strings.Join(errorMessages, "; "))
	}

	return exposedServices, nil
}
