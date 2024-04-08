// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

const helmReleaseSecretType = "helm.sh/release.v1" //nolint:gosec

// exposedConditions are the conditions that are exposed in the StatusConditions of the Plugin.
var exposedConditions = []greenhousev1alpha1.ConditionType{
	greenhousev1alpha1.ReadyCondition,
	greenhousev1alpha1.ClusterAccessReadyCondition,
	greenhousev1alpha1.HelmDriftDetectedCondition,
	greenhousev1alpha1.HelmReconcileFailedCondition,
	greenhousev1alpha1.StatusUpToDateCondition}

// HelmReconciler reconciles a Plugin object.
type HelmReconciler struct {
	client.Client
	KubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status;,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch;update

// These broad permissions are required as the controller manages Helm charts which contain arbitrary Kubernetes resources.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*

// SetupWithManager sets up the controller with the Manager.
func (r *HelmReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.kubeClientOpts = []clientutil.KubeClientOption{
		clientutil.WithRuntimeOptions(r.KubeRuntimeOpts),
		clientutil.WithPersistentConfig(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewMaxOfRateLimiter(
				workqueue.NewItemExponentialFailureRateLimiter(30*time.Second, 1*time.Hour),
				&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)})}).
		For(&greenhousev1alpha1.Plugin{}).
		// If the release was (manually) modified the secret would have been modified. Reconcile it.
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(enqueuePluginForReleaseSecret),
			builder.WithPredicates(clientutil.PredicateFilterBySecretType(helmReleaseSecretType), predicate.GenerationChangedPredicate{}),
		).
		// If a PluginDefinition was changed, reconcile relevant Plugins.
		Watches(&greenhousev1alpha1.PluginDefinition{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForPlugin),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPlugins),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&greenhousev1alpha1.Team{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsInNamespace), builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *HelmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var plugin = new(greenhousev1alpha1.Plugin)
	if err := r.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pluginStatus := initPluginStatus(plugin)

	defer func() {
		if statusErr := r.setStatus(ctx, plugin, pluginStatus); statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "failed to set status")
		}
	}()

	clusterAccessReadyCondition, restClientGetter := r.initClientGetter(ctx, *plugin, pluginStatus)
	pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
	if !clusterAccessReadyCondition.IsTrue() {
		return ctrl.Result{}, fmt.Errorf("cannot access cluster: %s", clusterAccessReadyCondition.Message)
	}

	// Cleanup Helm release.
	if plugin.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(plugin, greenhouseapis.FinalizerCleanupHelmRelease) {
		isDeleted, err := helm.UninstallHelmRelease(ctx, restClientGetter, plugin)
		if err != nil {
			c := greenhousev1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
			pluginStatus.StatusConditions.SetConditions(c)
			return ctrl.Result{}, err
		}
		if !isDeleted {
			// Ensure we're called again for some corner cases esp. where the actual deletion takes unusually long (hooks) yet the watch won't catch it.
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
		pluginStatus.StatusConditions.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))

		err = clientutil.RemoveFinalizer(ctx, r.Client, plugin, greenhouseapis.FinalizerCleanupHelmRelease)
		return ctrl.Result{}, err
	}

	if err := clientutil.EnsureFinalizer(ctx, r.Client, plugin, greenhouseapis.FinalizerCleanupHelmRelease); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: https://github.com/cloudoperators/greenhouse/issues/489
	helmReconcileFailedCondition, pluginDefinition := r.getPlugin(ctx, plugin)
	pluginStatus.StatusConditions.SetConditions(helmReconcileFailedCondition)
	if pluginDefinition == nil {
		return ctrl.Result{}, fmt.Errorf("pluginDefinition not found: %s", helmReconcileFailedCondition.Message)
	}

	driftDetectedCondition, reconcileFailedCondition := r.reconcileHelmRelease(ctx, restClientGetter, plugin, pluginDefinition, pluginStatus)
	pluginStatus.StatusConditions.SetConditions(driftDetectedCondition, reconcileFailedCondition)
	if reconcileFailedCondition.IsTrue() {
		return ctrl.Result{}, fmt.Errorf("helm reconcile failed: %s", helmReconcileFailedCondition.Message)
	}
	statusReconcileCompleteCondition := r.reconcileStatus(ctx, restClientGetter, plugin, pluginDefinition, &pluginStatus)
	pluginStatus.StatusConditions.SetConditions(statusReconcileCompleteCondition)

	return ctrl.Result{}, nil
}

func initPluginStatus(plugin *greenhousev1alpha1.Plugin) greenhousev1alpha1.PluginStatus {
	pluginStatus := plugin.Status.DeepCopy()
	for _, t := range exposedConditions {
		if pluginStatus.GetConditionByType(t) == nil {
			pluginStatus.SetConditions(greenhousev1alpha1.UnknownCondition(t, "", ""))
		}
	}
	if pluginStatus.HelmReleaseStatus == nil {
		pluginStatus.HelmReleaseStatus = &greenhousev1alpha1.HelmReleaseStatus{Status: "unknown"}
	}
	return *pluginStatus
}

// initClientGetter returns a RestClientGetter for the given Plugin.
// If the Plugin has a clusterName set, the RestClientGetter is initialized from the cluster secret.
// Otherwise, the RestClientGetter is initialized with in-cluster config
func (r *HelmReconciler) initClientGetter(
	ctx context.Context,
	plugin greenhousev1alpha1.Plugin,
	pluginStatus greenhousev1alpha1.PluginStatus,
) (
	clusterAccessReadyCondition greenhousev1alpha1.Condition,
	restClientGetter genericclioptions.RESTClientGetter,
) {

	clusterAccessReadyCondition = *pluginStatus.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
	clusterAccessReadyCondition.Status = metav1.ConditionTrue

	var err error

	// early return if spec.clusterName is not set
	if plugin.Spec.ClusterName == "" {
		restClientGetter, err = clientutil.NewRestClientGetterForInCluster(plugin.GetNamespace(), r.kubeClientOpts...)
		if err != nil {
			clusterAccessReadyCondition.Status = metav1.ConditionFalse
			clusterAccessReadyCondition.Message = fmt.Sprintf("cannot access greenhouse cluster: %s", err.Error())
			return clusterAccessReadyCondition, nil
		}
		return clusterAccessReadyCondition, restClientGetter
	}

	cluster := new(greenhousev1alpha1.Cluster)
	err = r.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("Failed to get cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}

	readyConditionInCluster := cluster.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)
	if readyConditionInCluster == nil || readyConditionInCluster.Status != metav1.ConditionTrue {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("cluster %s is not ready", plugin.Spec.ClusterName)
		return clusterAccessReadyCondition, nil
	}

	// get restclientGetter from cluster if clusterName is set
	secret := corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, &secret)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("Failed to get secret for cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}
	restClientGetter, err = clientutil.NewRestClientGetterFromSecret(&secret, plugin.GetNamespace(), r.kubeClientOpts...)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("cannot access cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}
	clusterAccessReadyCondition.Status = metav1.ConditionTrue
	clusterAccessReadyCondition.Message = ""
	return clusterAccessReadyCondition, restClientGetter
}

func (r *HelmReconciler) setStatus(ctx context.Context, plugin *greenhousev1alpha1.Plugin, pluginStatus greenhousev1alpha1.PluginStatus) error {
	readyCondition := r.computeReadyCondition(pluginStatus.StatusConditions)
	pluginStatus.StatusConditions.SetConditions(readyCondition)
	_, err := clientutil.PatchStatus(ctx, r.Client, plugin, func() error {
		plugin.Status = pluginStatus
		return nil
	})
	return err
}

func (r *HelmReconciler) getPlugin(
	ctx context.Context,
	plugin *greenhousev1alpha1.Plugin,
) (
	helmReconcileFailedCondition greenhousev1alpha1.Condition,
	pluginDefinition *greenhousev1alpha1.PluginDefinition,
) {

	var err error
	pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	helmReconcileFailedCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", "")

	if err = r.Get(ctx, types.NamespacedName{Namespace: plugin.GetNamespace(), Name: plugin.Spec.PluginDefinition}, pluginDefinition); err != nil {
		helmReconcileFailedCondition.Status = metav1.ConditionTrue
		helmReconcileFailedCondition.Reason = greenhousev1alpha1.PluginDefinitionNotFoundReason
		if apierrors.IsNotFound(err) {
			helmReconcileFailedCondition.Message = fmt.Sprintf("PluginDefinition %s does not exist", plugin.Spec.PluginDefinition)
			return helmReconcileFailedCondition, nil
		}
		helmReconcileFailedCondition.Message = fmt.Sprintf("Failed to get pluginDefinition %s: %s", plugin.Spec.PluginDefinition, err.Error())
		return helmReconcileFailedCondition, nil
	}
	helmReconcileFailedCondition.Status = metav1.ConditionFalse
	helmReconcileFailedCondition.Message = ""
	helmReconcileFailedCondition.Reason = ""
	return helmReconcileFailedCondition, pluginDefinition
}

func (r *HelmReconciler) reconcileHelmRelease(
	ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinition *greenhousev1alpha1.PluginDefinition,
	pluginStatus greenhousev1alpha1.PluginStatus,
) (driftDetectedCondition, reconcileFailedCondition greenhousev1alpha1.Condition) {

	driftDetectedCondition = *pluginStatus.GetConditionByType(greenhousev1alpha1.HelmDriftDetectedCondition)
	reconcileFailedCondition = *pluginStatus.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)

	// Not a HelmChart pluginDefinition. Ignore it.
	if pluginDefinition.Spec.HelmChart == nil {
		reconcileFailedCondition.Status = metav1.ConditionFalse
		reconcileFailedCondition.Message = "PluginDefinition is not backed by HelmChart"
		return driftDetectedCondition, reconcileFailedCondition
	}

	// Validate before attempting the installation/upgrade.
	// Any error is reflected in the status of the Plugin.
	if _, err := helm.TemplateHelmChartFromPlugin(ctx, r.Client, restClientGetter, pluginDefinition, plugin); err != nil {
		reconcileFailedCondition.Status = metav1.ConditionTrue
		reconcileFailedCondition.Message = fmt.Sprintf("Helm template failed: %s", err.Error())
		return driftDetectedCondition, reconcileFailedCondition
	}

	// Check whether the deployed resources match the ones we expect.
	diffObjects, isHelmDrift, err := helm.DiffChartToDeployedResources(ctx, r.Client, restClientGetter, pluginDefinition, plugin)
	if err != nil {
		reconcileFailedCondition.Status = metav1.ConditionTrue
		reconcileFailedCondition.Message = fmt.Sprintf("Helm diff failed: %s", err.Error())
		return driftDetectedCondition, reconcileFailedCondition
	}

	switch {
	case isHelmDrift: // drift was detected
		driftDetectedCondition.Status = metav1.ConditionTrue
		driftDetectedCondition.LastTransitionTime = metav1.Now()
		log.FromContext(ctx).Info("drift between deployed resources and manifest detected", "resources", diffObjects.String())
	case len(diffObjects) > 0: // diff detected
		driftDetectedCondition.Status = metav1.ConditionFalse
		driftDetectedCondition.LastTransitionTime = metav1.Now()
		log.FromContext(ctx).Info("diff between deployed release and manifest detected", "resources", diffObjects.String())
	default: // no diff detected and no drift detected
		driftDetectedCondition.Status = metav1.ConditionFalse
		driftDetectedCondition.LastTransitionTime = metav1.Now()

		reconcileFailedCondition.Status = metav1.ConditionFalse
		reconcileFailedCondition.Message = "Release for plugin is up-to-date"
		// TODO: remove unnecessary log?
		log.FromContext(ctx).Info("release for plugin is up-to-date")
		return driftDetectedCondition, reconcileFailedCondition
	}

	if err := helm.InstallOrUpgradeHelmChartFromPlugin(ctx, r.Client, restClientGetter, pluginDefinition, plugin); err != nil {
		reconcileFailedCondition.Status = metav1.ConditionTrue
		reconcileFailedCondition.Message = fmt.Sprintf("Helm install/upgrade failed: %s", err.Error())
		return driftDetectedCondition, reconcileFailedCondition
	}
	reconcileFailedCondition.Status = metav1.ConditionFalse
	reconcileFailedCondition.Message = "Helm install/upgrade successful"
	return driftDetectedCondition, reconcileFailedCondition
}

func (r *HelmReconciler) reconcileStatus(ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinition *greenhousev1alpha1.PluginDefinition,
	pluginStatus *greenhousev1alpha1.PluginStatus,
) (
	statusReconcileCondition greenhousev1alpha1.Condition,
) {

	var (
		pluginVersion   string
		exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
		releaseStatus   = &greenhousev1alpha1.HelmReleaseStatus{
			Status:        "unknown",
			FirstDeployed: metav1.Time{},
			LastDeployed:  metav1.Time{},
		}
	)

	statusReconcileCondition = *pluginStatus.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition)
	statusReconcileCondition.Status = metav1.ConditionTrue
	// Collect status from the Helm release.
	if helmRelease, err := helm.GetReleaseForHelmChartFromPlugin(ctx, restClientGetter, plugin); err == nil {
		// Ensure the status is always reported.
		if serviceList, err := getExposedServicesForPluginFromHelmRelease(restClientGetter, helmRelease, plugin); err == nil {
			exposedServices = serviceList
		} else {
			statusReconcileCondition.Status = metav1.ConditionFalse
			statusReconcileCondition.Message = fmt.Sprintf("failed to get exposed services: %s", err.Error())
		}

		// Get the release status.
		if latestReleaseInfo := helmRelease.Info; latestReleaseInfo != nil {
			releaseStatus.Status = latestReleaseInfo.Status.String()
			releaseStatus.FirstDeployed = metav1.NewTime(latestReleaseInfo.FirstDeployed.Time)
			releaseStatus.LastDeployed = metav1.NewTime(latestReleaseInfo.LastDeployed.Time)
			if latestReleaseInfo.Status == release.StatusDeployed {
				pluginVersion = latestReleaseInfo.Description
			}
		}
	} else {
		statusReconcileCondition.Status = metav1.ConditionFalse
		statusReconcileCondition.Message = fmt.Sprintf("failed to get Helm release: %s", err.Error())
	}
	var (
		uiApplication      *greenhousev1alpha1.UIApplicationReference
		helmChartReference *greenhousev1alpha1.HelmChartReference
	)
	// Ensure the status is always reported.
	uiApplication = pluginDefinition.Spec.UIApplication
	// only set the helm chart reference if the pluginVersion matches the pluginDefinition version or the release status is unknown
	if pluginVersion == pluginDefinition.Spec.Version || releaseStatus.Status == "unknown" {
		helmChartReference = pluginDefinition.Spec.HelmChart
	} else {
		helmChartReference = plugin.Status.HelmChart
	}

	pluginStatus.HelmReleaseStatus = releaseStatus
	pluginStatus.Version = pluginVersion
	pluginStatus.UIApplication = uiApplication
	pluginStatus.HelmChart = helmChartReference
	pluginStatus.Weight = pluginDefinition.Spec.Weight
	pluginStatus.Description = pluginDefinition.Spec.Description
	pluginStatus.ExposedServices = exposedServices

	return statusReconcileCondition
}

func (r *HelmReconciler) computeReadyCondition(
	conditions greenhousev1alpha1.StatusConditions,
) (readyCondition greenhousev1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)

	if conditions.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "cluster access not ready"
		return readyCondition
	}
	if conditions.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm reconcile failed"
		return readyCondition
	}
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

func (r *HelmReconciler) enqueueAllPlugins(ctx context.Context, _ client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client)
}

func (r *HelmReconciler) enqueueAllPluginsInNamespace(ctx context.Context, o client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client, client.InNamespace(o.GetNamespace()))
}

func (r *HelmReconciler) enqueueAllPluginsForPlugin(ctx context.Context, o client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client, client.MatchingLabels{greenhouseapis.LabelKeyPlugin: o.GetName()})
}

func listPluginsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var pluginList = new(greenhousev1alpha1.PluginList)
	if err := c.List(ctx, pluginList, listOpts...); err != nil {
		return nil
	}
	res := make([]ctrl.Request, len(pluginList.Items))
	for idx, plugin := range pluginList.Items {
		res[idx] = ctrl.Request{NamespacedName: client.ObjectKeyFromObject(plugin.DeepCopy())}
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
		Labels: map[string]string{
			greenhouseapis.LabelKeyExposeService: "true",
		},
	})
	if err != nil {
		return nil, err
	}
	var exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
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
		}
	}
	return exposedServices, nil
}
