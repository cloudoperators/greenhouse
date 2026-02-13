// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/kustomize"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const (
	maxHistory = 10
	secretKind = "Secret"
)

func (r *PluginReconciler) EnsureFluxDeleted(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (ctrl.Result, lifecycle.ReconcileResult, error) {
	// suspend the HelmRelease first to delete the Flux HelmRelease without removing the Helm release from the target cluster
	if plugin.Spec.DeletionPolicy == greenhouseapis.DeletionPolicyRetain {
		if _, err := r.EnsureFluxSuspended(ctx, plugin); err != nil {
			c := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
			plugin.SetCondition(c)
			util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonSuspendFailed)
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}

	if err := r.Delete(ctx, &helmv2.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: plugin.Name, Namespace: plugin.Namespace}}); client.IgnoreNotFound(err) != nil {
		c := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
		plugin.SetCondition(c)
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		return ctrl.Result{}, lifecycle.Failed, err
	}

	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginReconciler) EnsureFluxCreated(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginDefinition, err := common.GetPluginDefinitionFromPlugin(ctx, r.Client, plugin.Spec.PluginDefinitionRef, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.PluginDefinitionNotFoundReason, err.Error()))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonPluginDefinitionNotFound)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("%s not found: %s", plugin.Spec.PluginDefinitionRef.Kind, err.Error())
	}
	pluginDefinitionSpec := pluginDefinition.GetPluginDefinitionSpec()

	namespace := flux.HelmRepositoryDefaultNamespace
	if plugin.Spec.PluginDefinitionRef.Kind == greenhousev1alpha1.PluginDefinitionKind {
		namespace = plugin.GetNamespace()
	}

	if pluginDefinitionSpec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRelease creation", "plugin", plugin.Name)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "PluginDefinition is not backed by HelmChart"))
		// Update status for UI Applications.
		plugin.Status.UIApplication = pluginDefinitionSpec.UIApplication
		return ctrl.Result{}, lifecycle.Success, nil
	}

	helmChart, err := getPluginHelmChart(ctx, r.Client, pluginDefinition, namespace)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", fmt.Sprintf("Failed to load helm chart for %s/%s", plugin.Spec.PluginDefinitionRef.Kind, plugin.Spec.PluginDefinitionRef.Name)))
		return ctrl.Result{}, lifecycle.Failed, errors.New("helm chart not found for " + plugin.Spec.PluginDefinitionRef.Kind + "/" + plugin.Spec.PluginDefinitionRef.Name)
	}

	optionValues, err := computeReleaseValues(ctx, r.Client, plugin, r.ExpressionEvaluationEnabled, r.IntegrationEnabled)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.OptionValueResolutionFailedReason, err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	if err := r.ensureHelmRelease(ctx, plugin, *pluginDefinitionSpec, helmChart, optionValues); err != nil {
		log.FromContext(ctx).Error(err, "failed to ensure HelmRelease for Plugin", "name", plugin.Name, "namespace", plugin.Namespace)
		return ctrl.Result{}, lifecycle.Failed, err
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginReconciler) EnsureFluxSuspended(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (ctrl.Result, error) {
	release := &helmv2.HelmRelease{}
	release.SetName(plugin.Name)
	release.SetNamespace(plugin.Namespace)

	err := r.Get(ctx, client.ObjectKeyFromObject(release), release)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, release, func() error {
		release.Spec.Suspend = true
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	switch result {
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes, Plugin's HelmRelease already suspended", "name", release.Name)
	default:
		log.FromContext(ctx).Info("Suspend applied to Plugin's HelmRelease", "name", release.Name)
	}
	return ctrl.Result{}, nil
}

func (r *PluginReconciler) ensureHelmRelease(
	ctx context.Context,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	helmChart *sourcev1.HelmChart,
	optionValues []greenhousev1alpha1.PluginOptionValue,
) error {

	release := &helmv2.HelmRelease{}
	release.SetName(plugin.Name)
	release.SetNamespace(plugin.Namespace)

	mirrorConfig, err := common.GetRegistryMirrorConfig(ctx, r.Client, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", "Failed to read registry mirror configuration"))
		return fmt.Errorf("failed to read registry mirror configuration for Plugin %s: %w", plugin.Name, err)
	}

	values, err := generateHelmValues(ctx, optionValues)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.PluginOptionValueInvalidReason, err.Error()))
		return fmt.Errorf("failed to generate HelmRelease values for Plugin %s: %w", plugin.Name, err)
	}

	result, err := controllerutil.CreateOrPatch(ctx, r.Client, release, func() error {
		builder := flux.NewHelmReleaseSpecBuilder().
			WithHelmChartRef(&helmv2.CrossNamespaceSourceReference{
				Kind:      sourcev1.HelmChartKind,
				Name:      helmChart.Name,
				Namespace: helmChart.Namespace,
			}).
			WithInterval(flux.DefaultInterval).
			WithTimeout(flux.DefaultTimeout).
			WithMaxHistory(maxHistory).
			WithReleaseName(plugin.GetReleaseName()).
			WithInstall(&helmv2.Install{
				CreateNamespace: true,
				Remediation: &helmv2.InstallRemediation{
					Retries: 3,
				},
			}).
			WithUpgrade(&helmv2.Upgrade{
				Remediation: &helmv2.UpgradeRemediation{
					Retries: 3,
				},
			}).
			WithTest(&helmv2.Test{
				Enable: false,
			}).
			WithDriftDetection(configureDriftDetection(plugin.Spec.IgnoreDifferences)).
			WithSuspend(false).
			WithKubeConfig(&fluxmeta.SecretKeyReference{
				Name: plugin.Spec.ClusterName,
				Key:  greenhouseapis.GreenHouseKubeConfigKey,
			}).
			WithDependsOn(resolvePluginDependencies(plugin.Spec.WaitFor, plugin.Spec.ClusterName)).
			WithValues(values).
			WithValuesFrom(addValueReferences(plugin)).
			WithStorageNamespace(plugin.Spec.ReleaseNamespace).
			WithTargetNamespace(plugin.Spec.ReleaseNamespace)

		if mirrorConfig != nil && len(mirrorConfig.RegistryMirrors) > 0 {
			restClientGetter, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
			if err != nil {
				return fmt.Errorf("failed to init client getter for Plugin %s: %w", plugin.Name, err)
			}

			helmRelease, err := helm.TemplateHelmChartFromPluginOptionValues(ctx, r.Client, restClientGetter, &pluginDefinitionSpec, plugin, optionValues)
			if err != nil {
				return fmt.Errorf("failed to template helm chart for Plugin %s: %w", plugin.Name, err)
			}

			postRenderer := createRegistryMirrorPostRenderer(mirrorConfig, helmRelease.Manifest)
			if postRenderer != nil {
				builder = builder.WithPostRenderers([]helmv2.PostRenderer{*postRenderer})
			}
		}

		spec, err := builder.Build()
		if err != nil {
			return fmt.Errorf("failed to create HelmRelease for plugin %s: %w", plugin.Name, err)
		}
		release.Spec = spec

		val, _ := lifecycle.ReconcileAnnotationValue(plugin)
		common.EnsureAnnotation(release, fluxmeta.ReconcileRequestAnnotation, val)
		common.EnsureAnnotation(release, helmv2.ResetRequestAnnotation, val)

		return controllerutil.SetControllerReference(plugin, release, r.Scheme())
	})
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "Failed to create/update Helm release: "+err.Error()))
		return err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRelease", "name", release.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRelease", "name", release.Name)
	}

	ready := meta.FindStatusCondition(release.Status.Conditions, fluxmeta.ReadyCondition)
	if ready != nil && ready.ObservedGeneration == release.Generation {
		if ready.Status == metav1.ConditionTrue {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition,
				greenhousemetav1alpha1.ConditionReason(ready.Reason), ready.Message))
		} else {
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition,
				greenhousemetav1alpha1.ConditionReason(ready.Reason), ready.Message))
		}
	}

	return nil
}

func (r *PluginReconciler) computeReadyConditionFlux(ctx context.Context, plugin *greenhousev1alpha1.Plugin) greenhousemetav1alpha1.Condition {
	readyCondition := *plugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	restClientGetter, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
	if err != nil {
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "cluster access not ready"
		return readyCondition
	}

	pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(ctx, r.Client, plugin.Spec.PluginDefinitionRef, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.PluginDefinitionNotFoundReason, err.Error()))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonPluginDefinitionNotFound)
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm reconcile failed"
		return readyCondition
	}

	r.reconcilePluginStatus(ctx, restClientGetter, plugin, *pluginDefinitionSpec, &plugin.Status)

	if err := r.reconcileTechnicalLabels(ctx, plugin); err != nil {
		log.FromContext(ctx).Error(err, "failed to reconcile technical labels")
	}

	// If the Helm reconcile failed, the Plugin is not up to date / ready
	helmReconcileFailedCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
	if helmReconcileFailedCondition.IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm reconcile failed"
		return readyCondition
	}
	if helmReconcileFailedCondition.IsUnknown() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Reconciling"
		return readyCondition
	}
	// In other cases, the Plugin is ready
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

func (r *PluginReconciler) reconcilePluginStatus(ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	pluginStatus *greenhousev1alpha1.PluginStatus,
) {

	var (
		pluginVersion   string
		exposedServices = make(map[string]greenhousev1alpha1.Service)
		releaseStatus   = &greenhousev1alpha1.HelmReleaseStatus{
			Status:        "unknown",
			FirstDeployed: metav1.Time{},
			LastDeployed:  metav1.Time{},
		}
	)

	// Collect status from the Helm release.
	helmRelease := &helmv2.HelmRelease{}
	err := r.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: plugin.Namespace}, helmRelease)
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get Helm release: "+err.Error()))
	} else {
		helmSDKRelease, err := helm.GetReleaseForHelmChartFromPlugin(ctx, restClientGetter, plugin)
		if err != nil {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get Helm SDK release: "+err.Error()))
		} else {
			serviceList, err := getExposedServicesForPluginFromHelmRelease(restClientGetter, helmSDKRelease, plugin)
			if err != nil {
				plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
					greenhousev1alpha1.StatusUpToDateCondition, "", "failed to get exposed services: "+err.Error()))
			} else {
				exposedServices = serviceList
				plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.StatusUpToDateCondition, "", ""))
			}
		}

		// Get the latest successfully deployed release to set the dates.
		latestSnapshot := helmRelease.Status.History.Latest()
		if latestSnapshot != nil {
			releaseStatus.FirstDeployed = latestSnapshot.FirstDeployed
			releaseStatus.LastDeployed = latestSnapshot.LastDeployed
		}

		// HelmRelease Ready condition is the best representation of the release status.
		ready := meta.FindStatusCondition(helmRelease.Status.Conditions, fluxmeta.ReadyCondition)
		isReadyCurrent := ready != nil && ready.ObservedGeneration == helmRelease.Generation

		switch {
		case helmRelease.Spec.Suspend:
			releaseStatus.Status = "suspended"
		case isReadyCurrent && ready.Status == metav1.ConditionTrue:
			// If the current release is successfully deployed, get the status from history.
			if latestSnapshot != nil {
				releaseStatus.Status = latestSnapshot.Status
			} else {
				releaseStatus.Status = "deployed"
			}
			pluginVersion = pluginDefinitionSpec.Version
		case isReadyCurrent && ready.Status == metav1.ConditionUnknown:
			switch helmRelease.Status.LastAttemptedReleaseAction {
			case helmv2.ReleaseActionInstall:
				releaseStatus.Status = "pending-install"
			case helmv2.ReleaseActionUpgrade:
				releaseStatus.Status = "pending-upgrade"
			default:
				releaseStatus.Status = "progressing"
			}
		case isReadyCurrent && ready.Status == metav1.ConditionFalse:
			releaseStatus.Status = "failed"
		default:
			releaseStatus.Status = "progressing"
		}

		switch {
		case len(plugin.Spec.WaitFor) == 0:
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.WaitingForDependenciesCondition, "", ""))
		case isReadyCurrent && ready.Status == metav1.ConditionTrue:
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.WaitingForDependenciesCondition, "", ""))
		case isReadyCurrent && ready.Status == metav1.ConditionFalse && ready.Reason == helmv2.DependencyNotReadyReason:
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.WaitingForDependenciesCondition,
				greenhousemetav1alpha1.ConditionReason(ready.Reason), ready.Message))
		}

		// Check if retries are exhausted for install or upgrade operations.
		installExhausted := helmRelease.Spec.Install.GetRemediation().RetriesExhausted(helmRelease)
		upgradeExhausted := helmRelease.Spec.Upgrade.GetRemediation().RetriesExhausted(helmRelease)
		if installExhausted || upgradeExhausted {
			msg := fmt.Sprintf("install failures: %d, upgrade failures: %d (max retries: %d)",
				helmRelease.Status.InstallFailures, helmRelease.Status.UpgradeFailures,
				helmRelease.Spec.Install.GetRemediation().GetRetries())
			plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
				greenhousev1alpha1.RetriesExhaustedCondition, "RetriesExhausted", msg))
		} else {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.RetriesExhaustedCondition, "", ""))
		}

		oldChecksum := ""
		newChecksum := ""
		if plugin.Status.HelmReleaseStatus != nil && plugin.Status.HelmReleaseStatus.PluginOptionChecksum != "" {
			oldChecksum = plugin.Status.HelmReleaseStatus.PluginOptionChecksum
		}
		if plugin.Spec.OptionValues != nil {
			newChecksum, err = helm.CalculatePluginOptionChecksum(ctx, r.Client, plugin)
			if err != nil {
				releaseStatus.PluginOptionChecksum = ""
			} else {
				releaseStatus.PluginOptionChecksum = newChecksum
			}
		}
		if oldChecksum != "" {
			r.reconcileTrackingResources(ctx, plugin, oldChecksum, newChecksum)
		}
	}

	var (
		uiApplication      *greenhousev1alpha1.UIApplicationReference
		helmChartReference *greenhousev1alpha1.HelmChartReference
	)
	// Ensure the status is always reported.
	uiApplication = pluginDefinitionSpec.UIApplication
	// Only set the helm chart reference if the helm release has been applied successfully or the release status is unknown.
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

// computeReleaseValues resolves Expressions and ValueFromRefs in the Plugin's option values
// and inserts the Greenhouse values
func computeReleaseValues(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, expressionEvaluation, integrationEnabled bool) ([]greenhousev1alpha1.PluginOptionValue, error) {
	optionValues, err := helm.GetPluginOptionValuesForPlugin(ctx, c, plugin)
	if err != nil {
		return nil, err
	}
	trackedObjects := make([]string, 0)
	// initialize CEL resolver
	var celResolver *helm.CELResolver
	if expressionEvaluation {
		celResolver, err = helm.NewCELResolver(optionValues)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize CEL resolver: %w", err)
		}
	}
	for i, v := range optionValues {
		switch {
		case v.Value != nil:
			// noop, direct values are already set
			continue

		case v.Expression != nil:
			if !expressionEvaluation {
				// skip expression evaluation if not enabled
				continue
			}
			resolvedOptionValue, err := celResolver.ResolveExpression(v, expressionEvaluation)
			if err != nil {
				return nil, err
			}
			optionValues[i] = *resolvedOptionValue

		case v.ValueFrom != nil && v.ValueFrom.Ref != nil:
			// skip if integration flag is not enabled
			if !integrationEnabled {
				continue
			}
			//TODO: handle external references
			resolvedOptionValue, objectTrackers, err := ResolveValueFromRef(ctx, c, plugin, v)
			if err != nil {
				return nil, err
			}
			trackedObjects = append(trackedObjects, objectTrackers...)
			optionValues[i] = *resolvedOptionValue

		case v.ValueFrom != nil && v.ValueFrom.Secret != nil:
			// noop, secret refs are not resolved here
			continue
		default:
			return nil, fmt.Errorf("option value %s has no value or valueFrom set", v.Name)
		}
	}

	// update tracking information for plugin integrations
	if integrationEnabled {
		// remove tracking annotations from resources that are no longer being tracked
		if err := removeUntrackedObjectAnnotations(ctx, c, plugin, trackedObjects); err != nil {
			// log err, will retry on next reconciliation
			log.FromContext(ctx).Error(err, "failed to remove untracked object annotations", "namespace", plugin.Namespace, "plugin", plugin.Name)
		}
		if len(trackedObjects) > 0 {
			plugin.Status.TrackedObjects = trackedObjects
		} else {
			// clear tracked objects if there are none
			plugin.Status.TrackedObjects = nil
		}
	}

	return optionValues, nil
}

// generateHelmValues generates the Helm values in JSON format to be used with a Flux HelmRelease.
func generateHelmValues(ctx context.Context, optionValues []greenhousev1alpha1.PluginOptionValue) ([]byte, error) {
	o := make([]greenhousev1alpha1.PluginOptionValue, len(optionValues))
	for _, v := range optionValues {
		if v.ValueFrom != nil && v.ValueFrom.Secret != nil {
			// remove all option values that are set from a secret, as these have a nil value
			continue
		}
		o = append(o, v)
	}

	jsonValue, err := helm.ConvertFlatValuesToHelmValues(o)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plugin option values to JSON: %w", err)
	}

	byteValue, err := json.Marshal(jsonValue)
	if err != nil {
		log.FromContext(ctx).Error(err, "Unable to marshal values for plugin")
		return nil, err
	}
	return byteValue, nil
}

// configureDriftDetection configures drift detection for the HelmRelease based on the provided ignore differences.
// The mode is set to enable, and ignore rules are added for each specified ignore difference.
func configureDriftDetection(ignoreDifferences []greenhousev1alpha1.IgnoreDifference) *helmv2.DriftDetection {
	driftDetection := &helmv2.DriftDetection{
		Mode: helmv2.DriftDetectionEnabled,
	}
	for _, ignore := range ignoreDifferences {
		driftDetection.Ignore = append(driftDetection.Ignore, helmv2.IgnoreRule{
			Target: &kustomize.Selector{
				Group:   ignore.Group,
				Version: ignore.Version,
				Kind:    ignore.Kind,
				Name:    ignore.Name,
			},
			Paths: ignore.Paths,
		})
	}
	return driftDetection
}

func addValueReferences(plugin *greenhousev1alpha1.Plugin) []helmv2.ValuesReference {
	var valuesFrom []helmv2.ValuesReference
	for _, value := range plugin.Spec.OptionValues {
		if value.ValueFrom != nil && value.ValueFrom.Secret != nil {
			valuesFrom = append(valuesFrom, helmv2.ValuesReference{
				Kind:       secretKind,
				Name:       value.ValueFrom.Secret.Name,
				ValuesKey:  value.ValueFrom.Secret.Key,
				TargetPath: value.Name,
			})
		}
	}
	return valuesFrom
}

// reconcileTrackingResources triggers reconciliation on resources that are tracking this plugin.
// When a plugin's option values change (detected by checksum change), this function annotates
// all resources that reference this plugin to trigger their reconciliation.
func (r *PluginReconciler) reconcileTrackingResources(ctx context.Context, plugin *greenhousev1alpha1.Plugin, oldChecksum, newChecksum string) {
	if oldChecksum == newChecksum {
		// No changes, skip reconciliation
		return
	}

	// Get the list of trackers from plugin annotations
	trackerIDs := getTrackerIDsFromAnnotations(plugin)
	if len(trackerIDs) == 0 {
		return
	}

	// Trigger reconciliation for each tracking resource
	for _, trackerID := range trackerIDs {
		if err := r.triggerReconcileForTracker(ctx, plugin, trackerID); err != nil {
			log.FromContext(ctx).Error(err, "failed to trigger reconciliation for tracking resource", "trackerID", trackerID)
		}
	}
}

// triggerReconcileForTracker triggers reconciliation for a single tracking resource.
func (r *PluginReconciler) triggerReconcileForTracker(ctx context.Context, plugin *greenhousev1alpha1.Plugin, trackerID string) error {
	// Parse the tracker ID
	kind, name, err := parseTrackingID(trackerID)
	if err != nil {
		log.FromContext(ctx).Error(err, "invalid tracker ID format", "trackerID", trackerID)
		return err
	}

	// Skip self-references
	if name == plugin.GetName() {
		return nil
	}

	// Build GVK and key for the tracking resource
	gvk := buildGVK(kind)
	key := types.NamespacedName{
		Name:      name,
		Namespace: plugin.GetNamespace(),
	}

	// Update the resource with reconcile annotation
	err = updateResourceWithAnnotation(ctx, r.Client, gvk, key)

	if err != nil {
		log.FromContext(ctx).Error(err, "failed to annotate tracking object with reconcile request",
			"kind", kind,
			"namespace", plugin.GetNamespace(),
			"name", name)
		return err
	}

	return nil
}

func getPluginHelmChart(ctx context.Context, c client.Client, pluginDef common.GenericPluginDefinition, namespace string) (*sourcev1.HelmChart, error) {
	helmChartResourceName := pluginDef.FluxHelmChartResourceName()
	helmChart := &sourcev1.HelmChart{}
	err := c.Get(ctx, types.NamespacedName{Name: helmChartResourceName, Namespace: namespace}, helmChart)
	if err != nil {
		return nil, err
	}
	return helmChart, nil
}
