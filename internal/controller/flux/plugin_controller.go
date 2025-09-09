// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	helmcontroller "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	pluginController "github.com/cloudoperators/greenhouse/internal/controller/plugin"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/util"
)

const (
	maxHistory = 10
	secretKind = "Secret"

	PluginDefinitionVersionAnnotation = "greenhouse.sap/pd-version"
	StatusUnknown                     = "Unknown"
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

	labelSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      greenhouseapis.GreenhouseHelmDeliveryToolLabel,
				Operator: metav1.LabelSelectorOpExists,
			},
			{
				Key:      greenhouseapis.GreenhouseHelmDeliveryToolLabel,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{greenhouseapis.GreenhouseHelmDeliveryToolFlux},
			},
		},
	}

	labelSelectorPredicate, err := predicate.LabelSelectorPredicate(labelSelector)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Plugin{}, builder.WithPredicates(labelSelectorPredicate)).
		Owns(&helmcontroller.HelmRelease{}, builder.WithPredicates(clientutil.PredicateHelmReleaseWithStatusReadyChange())).
		// If a PluginDefinition was changed, reconcile relevant Plugins.
		Watches(&greenhousev1alpha1.ClusterPluginDefinition{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForPluginDefinition),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, labelSelectorPredicate)).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsForCluster), builder.WithPredicates(labelSelectorPredicate)).
		Watches(&greenhousev1alpha1.Team{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginsInNamespace), builder.WithPredicates(predicate.GenerationChangedPredicate{}, labelSelectorPredicate)).
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
		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, plugin)
		plugin.Status.SetConditions(readyCondition, ownerLabelCondition)
	}
}

func (r *FluxReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin := resource.(*greenhousev1alpha1.Plugin) //nolint:errcheck

	if err := r.Delete(ctx, &helmcontroller.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: plugin.Name, Namespace: plugin.Namespace}}); err != nil {
		c := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
		plugin.SetCondition(c)
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("cannot access cluster: %s", err.Error())
	}

	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *FluxReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	plugin, ok := resource.(*greenhousev1alpha1.Plugin)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.New("resource is not a Plugin")
	}

	// ignore plugins that are not managed by Flux
	if plugin.GetLabels() != nil && plugin.GetLabels()[greenhouseapis.GreenhouseHelmDeliveryToolLabel] != greenhouseapis.GreenhouseHelmDeliveryToolFlux {
		return ctrl.Result{}, lifecycle.Pending, nil
	}

	pluginController.InitPluginStatus(plugin)

	pluginDef := r.getPluginDef(ctx, plugin)
	if pluginDef == nil {
		return ctrl.Result{}, lifecycle.Failed, errors.New("plugin definition not found")
	}

	namespace := flux.HelmRepositoryDefaultNamespace
	if pluginDef.Namespace != "" {
		namespace = pluginDef.Namespace
	}

	if pluginDef.Spec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRelease creation", "plugin", plugin.Name)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "PluginDefinition is not backed by HelmChart"))
		return ctrl.Result{}, lifecycle.Success, nil
	}

	helmRepository, err := flux.FindHelmRepositoryByURL(ctx, r.Client, pluginDef.Spec.HelmChart.Repository, namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, errors.New("helm repository not found")
	}

	if err := r.ensureHelmRelease(ctx, plugin, pluginDef.Spec, helmRepository); err != nil {
		log.FromContext(ctx).Error(err, "failed to ensure HelmRelease for Plugin", "name", plugin.Name, "namespace", plugin.Namespace)
		return ctrl.Result{}, lifecycle.Failed, err
	}

	r.reconcilePluginStatus(ctx, plugin, pluginDef, &plugin.Status)

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *FluxReconciler) reconcilePluginStatus(ctx context.Context,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition,
	pluginStatus *greenhousev1alpha1.PluginStatus,
) {

	var (
		pluginVersion   string
		exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
		releaseStatus   = &greenhousev1alpha1.HelmReleaseStatus{
			Status:        StatusUnknown,
			FirstDeployed: metav1.Time{},
			LastDeployed:  metav1.Time{},
			Diff:          pluginStatus.HelmReleaseStatus.Diff,
		}
	)

	// Collect status from the Helm release.
	helmRelease := &helmcontroller.HelmRelease{}
	err := r.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: plugin.Namespace}, helmRelease)
	if err == nil {
		serviceList, err := getExposedServicesForPluginFromHelmRelease(ctx, r.Client, helmRelease, plugin)
		if err == nil {
			exposedServices = serviceList
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.StatusUpToDateCondition, "", ""))
		} else {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get exposed services: "+err.Error()))
		}

		// Get the release status.
		latestSnapshot := helmRelease.Status.History.Latest()
		if latestSnapshot != nil {
			releaseStatus.FirstDeployed = latestSnapshot.FirstDeployed
			releaseStatus.LastDeployed = latestSnapshot.LastDeployed
		}
		releasedCondition := meta.FindStatusCondition(helmRelease.Status.Conditions, helmcontroller.ReleasedCondition)
		if releasedCondition != nil && releasedCondition.Status == metav1.ConditionTrue &&
			releasedCondition.ObservedGeneration >= helmRelease.Generation {
			if v := helmRelease.Annotations[PluginDefinitionVersionAnnotation]; v != "" {
				pluginVersion = v
			}
			if releaseStatus.LastDeployed.IsZero() {
				releaseStatus.LastDeployed = releasedCondition.LastTransitionTime
			}
		}
		releaseStatus.Status = getReleaseStatus(helmRelease)

		if plugin.Spec.OptionValues != nil {
			checksum, err := helm.CalculatePluginOptionChecksum(ctx, r.Client, plugin)
			if err == nil {
				releaseStatus.PluginOptionChecksum = checksum
			} else {
				releaseStatus.PluginOptionChecksum = ""
			}
		}
	} else {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get Helm release: "+err.Error()))
	}

	var (
		uiApplication      *greenhousev1alpha1.UIApplicationReference
		helmChartReference *greenhousev1alpha1.HelmChartReference
	)
	// Ensure the status is always reported.
	uiApplication = pluginDefinition.Spec.UIApplication
	// only set the helm chart reference if the pluginVersion matches the pluginDefinition version or the release status is unknown
	if pluginVersion == pluginDefinition.Spec.Version || releaseStatus.Status == StatusUnknown {
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
}

func getExposedServicesForPluginFromHelmRelease(ctx context.Context, c client.Client, hr *helmcontroller.HelmRelease, plugin *greenhousev1alpha1.Plugin) (map[string]greenhousev1alpha1.Service, error) {
	latest := hr.Status.History.Latest()
	if latest == nil {
		return nil, nil
	}
	if plugin.Spec.ClusterName == "" {
		return nil, errors.New("plugin does not have ClusterName")
	}

	storageNamespace := hr.GetStorageNamespace()
	releaseName := hr.GetReleaseName()

	var storageSecret corev1.Secret
	// TODO: change to remote kube client!
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: storageNamespace,
		Name:      fmt.Sprintf("sh.helm.release.v1.%s.v%d", releaseName, latest.Version),
	}, &storageSecret); err != nil {
		return nil, err
	}

	gzBytes, err := base64.StdEncoding.DecodeString(string(storageSecret.Data["release"]))
	if err != nil {
		return nil, err
	}
	gr, err := gzip.NewReader(strings.NewReader(string(gzBytes)))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	jsonBytes, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	var rel release.Release
	if err := json.Unmarshal(jsonBytes, &rel); err != nil {
		return nil, err
	}

	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(rel.Manifest)), 4096)

	var exposedServiceList []corev1.Service
	for {
		var obj map[string]any

		if err := dec.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode manifest doc: %w", err)
		}
		if len(obj) == 0 {
			continue
		}

		u := &unstructured.Unstructured{Object: obj}
		if u.GetKind() != "Service" {
			continue
		}

		// Check label expose:"true" on the manifest object
		labels := u.GetLabels()
		if labels[greenhouseapis.LabelKeyExposeService] != "true" {
			continue
		}

		var svc corev1.Service
		b, err := u.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("to Service: %w", err)
		}
		if err := json.Unmarshal(b, &svc); err != nil {
			return nil, fmt.Errorf("to Service: %w", err)
		}
		exposedServiceList = append(exposedServiceList, svc)
	}

	var exposedServices = make(map[string]greenhousev1alpha1.Service, 0)
	if len(exposedServiceList) == 0 {
		return exposedServices, nil
	}
	for _, svc := range exposedServiceList {
		svcPort, err := getPortForExposedService(svc)
		if err != nil {
			return nil, err
		}
		namespace := svc.Namespace
		if namespace == "" {
			namespace = rel.Namespace // default namespace to release namespace
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

func getReleaseStatus(helmRelease *helmcontroller.HelmRelease) string {
	conditions := helmRelease.Status.Conditions
	currentGen := helmRelease.Generation

	isCurrent := func(c *metav1.Condition) bool {
		return c != nil && c.ObservedGeneration >= currentGen
	}
	isTrue := func(c *metav1.Condition) bool {
		return isCurrent(c) && c.Status == metav1.ConditionTrue
	}

	stalled := meta.FindStatusCondition(conditions, fluxmeta.StalledCondition)
	if isTrue(stalled) {
		return fluxmeta.StalledCondition
	}
	ready := meta.FindStatusCondition(conditions, fluxmeta.ReadyCondition)
	if isTrue(ready) {
		return fluxmeta.ReadyCondition
	}
	reconciling := meta.FindStatusCondition(conditions, fluxmeta.ReconcilingCondition)
	// As per meta contract the ReadyCondition == False should be treated as Reconciling.
	if isTrue(reconciling) || isCurrent(ready) {
		return fluxmeta.ReconcilingCondition
	}
	return StatusUnknown
}

func looksLikeClusterAccessError(msg string) bool {
	msg = strings.ToLower(msg)
	patterns := []string{"kubeconfig", "cannot create kubernetes client", "failed to build rest",
		"unauthorized", "forbidden", "x509", "certificate", "dial tcp", "timeout", "connection refused"}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

func (r *FluxReconciler) enqueueAllPluginsForPluginDefinition(ctx context.Context, o client.Object) []ctrl.Request {
	// TODO: Once namespaced PluginDefinitions are supported, we need a logic here to handle the correct label key
	return pluginController.ListPluginsAsReconcileRequests(ctx, r.Client, client.MatchingLabels{greenhouseapis.LabelKeyClusterPluginDefinition: o.GetName()})
}

// enqueueAllPluginsForCluster enqueues all Plugins which have .spec.clusterName set to the name of the given Cluster.
func (r *FluxReconciler) enqueueAllPluginsForCluster(ctx context.Context, o client.Object) []ctrl.Request {
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(greenhouseapis.PluginClusterNameField, o.GetName()),
		Namespace:     o.GetNamespace(),
	}
	return pluginController.ListPluginsAsReconcileRequests(ctx, r.Client, listOpts)
}

func (r *FluxReconciler) enqueueAllPluginsInNamespace(ctx context.Context, o client.Object) []ctrl.Request {
	return pluginController.ListPluginsAsReconcileRequests(ctx, r.Client, client.InNamespace(o.GetNamespace()))
}

func (r *FluxReconciler) getPluginDef(ctx context.Context, plugin *greenhousev1alpha1.Plugin) *greenhousev1alpha1.ClusterPluginDefinition {
	pluginDef := new(greenhousev1alpha1.ClusterPluginDefinition)
	if err := r.Get(ctx, types.NamespacedName{Name: plugin.Spec.PluginDefinition}, pluginDef); err != nil {
		log.FromContext(ctx).Error(err, "Unable to find pluginDefinition for ", "plugin", plugin.Name, "namespace", plugin.Namespace)
		return nil
	}
	return pluginDef
}

func (r *FluxReconciler) ensureHelmRelease(
	ctx context.Context,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	helmRepository *sourcecontroller.HelmRepository,
) error {

	helmRelease := &helmcontroller.HelmRelease{}
	helmRelease.SetName(plugin.Name)
	helmRelease.SetNamespace(plugin.Namespace)

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, helmRelease, func() error {
		values, err := addValuesToHelmRelease(ctx, r.Client, plugin)
		if err != nil {
			return fmt.Errorf("failed to compute HelmRelease values for Plugin %s: %w", plugin.Name, err)
		}

		spec, err := flux.NewHelmReleaseSpecBuilder().
			WithChart(helmcontroller.HelmChartTemplateSpec{
				Chart:    pluginDefinitionSpec.HelmChart.Name,
				Interval: &metav1.Duration{Duration: flux.DefaultInterval},
				Version:  pluginDefinitionSpec.HelmChart.Version,
				SourceRef: helmcontroller.CrossNamespaceObjectReference{
					Kind:      sourcecontroller.HelmRepositoryKind,
					Name:      helmRepository.Name,
					Namespace: helmRepository.Namespace,
				},
			}).
			WithInterval(flux.DefaultInterval).
			WithTimeout(flux.DefaultTimeout).
			WithMaxHistory(maxHistory).
			WithReleaseName(plugin.GetReleaseName()).
			WithInstall(&helmcontroller.Install{
				CreateNamespace: true,
				Remediation: &helmcontroller.InstallRemediation{
					Retries: 3,
				},
			}).
			WithUpgrade(&helmcontroller.Upgrade{
				Remediation: &helmcontroller.UpgradeRemediation{
					Retries: 3,
				},
			}).
			WithTest(&helmcontroller.Test{
				Enable: false,
			}).
			WithDriftDetection(&helmcontroller.DriftDetection{
				Mode: helmcontroller.DriftDetectionEnabled,
			}).
			WithKubeConfig(fluxmeta.SecretKeyReference{
				Name: plugin.Spec.ClusterName,
				Key:  greenhouseapis.GreenHouseKubeConfigKey,
			}).
			WithValues(values).
			WithValuesFrom(r.addValueReferences(plugin)).
			WithTargetNamespace(plugin.Spec.ReleaseNamespace).Build()
		if err != nil {
			return fmt.Errorf("failed to create HelmRelease for plugin %s: %w", plugin.Name, err)
		}
		helmRelease.Spec = spec

		// Set PluginDefinition.Spec.Version in the release.
		if helmRelease.Annotations == nil {
			helmRelease.Annotations = map[string]string{}
		}
		helmRelease.Annotations[PluginDefinitionVersionAnnotation] = pluginDefinitionSpec.Version

		return controllerutil.SetControllerReference(plugin, helmRelease, r.Scheme())
	})
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.StatusUpToDateCondition, "", "failed to create/update Helm release: "+err.Error()))
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRelease", "name", helmRelease.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRelease", "name", helmRelease.Name)
	}

	helmReleaseReady := meta.FindStatusCondition(helmRelease.Status.Conditions, fluxmeta.ReadyCondition)
	if helmReleaseReady != nil && helmRelease.Status.ObservedGeneration >= helmRelease.Generation {
		if helmReleaseReady.Status == metav1.ConditionTrue {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition,
				greenhousemetav1alpha1.ConditionReason(helmReleaseReady.Reason), helmReleaseReady.Message))

			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
		} else {
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition,
				greenhousemetav1alpha1.ConditionReason(helmReleaseReady.Reason), helmReleaseReady.Message))

			// Approximate access to the cluster based on reason and message returned by flux helmcontroller.
			if helmReleaseReady.Reason == helmcontroller.TestFailedReason ||
				helmReleaseReady.Reason == helmcontroller.ArtifactFailedReason ||
				!looksLikeClusterAccessError(helmReleaseReady.Message) {
				plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
			} else {
				plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ClusterAccessReadyCondition,
					greenhousemetav1alpha1.ConditionReason(helmReleaseReady.Reason), helmReleaseReady.Message))
			}
		}
	}

	return nil
}

func addValuesToHelmRelease(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]byte, error) {
	optionValues, err := helm.GetPluginOptionValuesForPlugin(ctx, c, plugin)
	if err != nil {
		return nil, err
	}

	optionValues, err = helm.ResolveTemplatedValues(ctx, c, plugin, optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve templated values: %w", err)
	}

	// remove all option values that are set from a secret, as these have a nil value
	optionValues = slices.DeleteFunc(optionValues, func(v greenhousev1alpha1.PluginOptionValue) bool {
		return v.ValueFrom != nil && v.ValueFrom.Secret != nil
	})

	jsonValue, err := helm.ConvertFlatValuesToHelmValues(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plugin option values to JSON: %w", err)
	}

	byteValue, err := json.Marshal(jsonValue)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Unable to marshal values for plugin", "plugin", plugin.Name)
		return nil, err
	}
	return byteValue, nil
}

func (r *FluxReconciler) addValueReferences(plugin *greenhousev1alpha1.Plugin) []helmcontroller.ValuesReference {
	var valuesFrom []helmcontroller.ValuesReference
	for _, value := range plugin.Spec.OptionValues {
		if value.ValueFrom != nil {
			valuesFrom = append(valuesFrom, helmcontroller.ValuesReference{
				Kind:       secretKind,
				Name:       value.ValueFrom.Secret.Name,
				ValuesKey:  value.ValueFrom.Secret.Key,
				TargetPath: value.Name,
			})
		}
	}
	return valuesFrom
}
