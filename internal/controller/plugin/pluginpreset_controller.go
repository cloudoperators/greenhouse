// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/util"
)

// presetExposedConditions contains the conditions that are exposed in the PluginPreset's StatusConditions.
var presetExposedConditions = []greenhousemetav1alpha1.ConditionType{
	greenhousemetav1alpha1.ReadyCondition,
	greenhousev1alpha1.PluginSkippedCondition,
	greenhousev1alpha1.PluginFailedCondition,
	greenhousev1alpha1.AllPluginsReadyCondition,
	greenhousemetav1alpha1.ClusterListEmpty,
	greenhousemetav1alpha1.OwnerLabelSetCondition,
}

// PluginPresetReconciler reconciles a PluginPreset object
type PluginPresetReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginpresets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginpresets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginpresets/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get

// SetupWithManager sets up the controller with the Manager.
func (r *PluginPresetReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginPreset{}).
		Owns(&greenhousev1alpha1.Plugin{}, builder.WithPredicates(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				clientutil.PredicatePluginWithStatusReadyChange(),
			))).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginPresetsInNamespace),
			builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Complete(r)
}

func (r *PluginPresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.PluginPreset{}, r, r.setConditions())
}

func (r *PluginPresetReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		pluginPreset, ok := resource.(*greenhousev1alpha1.PluginPreset)
		if !ok {
			logger.Error(errors.New("resource is not a PluginPreset"), "status setup failed")
			return
		}

		readyCondition := r.computeReadyCondition(pluginPreset.Status.StatusConditions)
		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, pluginPreset)
		util.UpdateOwnedByLabelMissingMetric(pluginPreset, ownerLabelCondition.IsFalse())
		pluginPreset.Status.SetConditions(readyCondition, ownerLabelCondition)
	}
}

func (r *PluginPresetReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginPreset := resource.(*greenhousev1alpha1.PluginPreset) //nolint:errcheck

	_ = log.FromContext(ctx)

	initPluginPresetStatus(pluginPreset)

	clusters, err := r.listClusters(ctx, pluginPreset)
	if err != nil {
		pluginPreset.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ClusterListEmpty, "", fmt.Sprintf("Invalid ClusterSelector: %v", err.Error())))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	switch {
	case len(clusters.Items) == 0:
		pluginPreset.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ClusterListEmpty, "", "No cluster matches ClusterSelector"))
	default:
		pluginPreset.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ClusterListEmpty, "", ""))
	}

	if err := r.cleanupPlugins(ctx, pluginPreset, clusters); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	clusters = clientutil.FilterClustersBeingDeleted(clusters)

	err = r.reconcilePluginPreset(ctx, pluginPreset, clusters)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	err = r.reconcilePluginStatuses(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginPresetReconciler) EnsureDeleted(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginPreset := resource.(*greenhousev1alpha1.PluginPreset) //nolint:errcheck

	plugins, err := r.listPlugins(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	switch pluginPreset.Spec.DeletionPolicy {
	case greenhouseapis.DeletionPolicyRetain:
		// Remove the owner reference from all managed Plugins to retain them when deleting the Preset.
		allErrs := make([]error, 0)
		for _, plugin := range plugins.Items {
			if isPluginManagedByPreset(&plugin, pluginPreset.Name) {
				// Remove the owner reference from the Plugin.
				_, err := clientutil.Patch(ctx, r.Client, &plugin, func() error {
					if len(plugin.OwnerReferences) > 0 {
						return controllerutil.RemoveOwnerReference(pluginPreset, &plugin, r.Scheme())
					}
					return nil
				})
				if err != nil {
					allErrs = append(allErrs, err)
				}
			}
		}
		if len(allErrs) > 0 {
			return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("failed to process retained plugins for %s/%s: %w", pluginPreset.Namespace, pluginPreset.Name, errors.Join(allErrs...))
		}
	default:
		// Cleanup the plugins that are managed by this PluginPreset.
		allErrs := make([]error, 0)
		for _, plugin := range plugins.Items {
			if err := r.Delete(ctx, &plugin); err != nil && !apierrors.IsNotFound(err) {
				allErrs = append(allErrs, err)
			}
		}
		// If the deletion of one or more Plugins failed, requeue the deletion.
		if len(allErrs) > 0 {
			return ctrl.Result{}, lifecycle.Pending, fmt.Errorf("failed to delete plugins for %s/%s: %w", pluginPreset.Namespace, pluginPreset.Name, errors.Join(allErrs...))
		}
		// If there were Plugins left, requeue the deletion
		if len(plugins.Items) > 0 {
			return ctrl.Result{}, lifecycle.Pending, nil
		}
	}
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginPresetReconciler) EnsureSuspended(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// reconcilePluginPreset reconciles the PluginPreset by creating or updating the Plugins for the given clusters.
// It skips reconciliation for Plugins that do not have the labels of the PluginPreset.
func (r *PluginPresetReconciler) reconcilePluginPreset(ctx context.Context, preset *greenhousev1alpha1.PluginPreset, clusters *greenhousev1alpha1.ClusterList) error {
	var allErrs = make([]error, 0)
	var skippedPlugins = make([]string, 0)
	var failedPlugins = make([]string, 0)

	for _, cluster := range clusters.Items {
		plugin := &greenhousev1alpha1.Plugin{}
		err := r.Get(ctx, client.ObjectKey{Namespace: preset.GetNamespace(), Name: generatePluginName(preset, &cluster)}, plugin)

		switch {
		case !cluster.DeletionTimestamp.IsZero():
			continue
		case err == nil:
			// The Plugin exists but does not contain the labels of the PluginPreset. This Plugin is not managed by the PluginPreset and must not be touched.
			if !isPluginManagedByPreset(plugin, preset.Name) {
				skippedPlugins = append(skippedPlugins, plugin.Name)
				continue
			}

		case apierrors.IsNotFound(err):
			plugin = &greenhousev1alpha1.Plugin{}
			plugin.SetName(generatePluginName(preset, &cluster))
			plugin.SetNamespace(preset.GetNamespace())
		default:
			return err
		}

		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, plugin, func() error {
			// Label the plugin with the managed resource label to identify it as managed by the PluginPreset.
			// Keep any existing labels.
			labels := plugin.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[greenhouseapis.LabelKeyPluginPreset] = preset.Name
			plugin.SetLabels(labels)
			// Set the owner reference to the PluginPreset. This is used to trigger reconciliation, if the managed Plugin is modified.
			if err := controllerutil.SetControllerReference(preset, plugin, r.Scheme()); err != nil {
				return err
			}

			// copy from preset to plugin spec
			if preset.Spec.Plugin.PluginDefinitionRef == (greenhousev1alpha1.PluginDefinitionReference{}) {
				plugin.Spec.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
					Name: preset.Spec.Plugin.PluginDefinition, //nolint:staticcheck
					Kind: greenhousev1alpha1.PluginDefinitionKind,
				}
			} else {
				plugin.Spec.PluginDefinitionRef = preset.Spec.Plugin.PluginDefinitionRef
			}
			plugin.Spec.PluginDefinitionRef = preset.Spec.Plugin.PluginDefinitionRef
			plugin.Spec.DisplayName = preset.Spec.Plugin.DisplayName
			plugin.Spec.ReleaseNamespace = preset.Spec.Plugin.ReleaseNamespace
			plugin.Spec.OptionValues = preset.Spec.Plugin.OptionValues

			// set back existing/computed values
			plugin.Spec.ReleaseName = getReleaseName(plugin, preset)
			// Set the cluster name to the name of the cluster. The PluginSpec contained in the PluginPreset does not have a cluster name.
			plugin.Spec.ClusterName = cluster.GetName()
			// Copy over the plugin dependencies
			plugin.Spec.WaitFor = preset.Spec.WaitFor
			// transport plugin preset labels to plugin
			plugin = (lifecycle.NewPropagator(preset, plugin).ApplyLabels()).(*greenhousev1alpha1.Plugin) //nolint:errcheck
			// overrides options based on preset definition
			overridesPluginOptionValues(plugin, preset)
			return nil
		})
		if err != nil {
			errorMessage := err.Error()
			var e *apierrors.StatusError
			if errors.As(err, &e) && e.ErrStatus.Details != nil && len(e.ErrStatus.Details.Causes) > 0 {
				// Extract the reason for failed Plugin.
				errorMessage = e.ErrStatus.Details.Causes[0].Message
			}
			failedPlugins = append(failedPlugins, plugin.Name+": "+errorMessage)
			allErrs = append(allErrs, err)
		}
	}
	switch {
	case len(skippedPlugins) > 0:
		preset.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.PluginSkippedCondition, "", "Skipped existing plugins: "+strings.Join(skippedPlugins, ", ")))
	default:
		preset.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.PluginSkippedCondition, "", ""))
	}
	switch {
	case len(failedPlugins) > 0:
		preset.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.PluginFailedCondition, greenhousev1alpha1.PluginReconcileFailed, strings.Join(failedPlugins, "; ")))
	default:
		preset.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.PluginFailedCondition, "", ""))
	}
	return utilerrors.NewAggregate(allErrs)
}

// reconcilePluginStatuses updates plugin statuses in PluginPreset for every Plugin managed by the Preset.
func (r *PluginPresetReconciler) reconcilePluginStatuses(
	ctx context.Context, preset *greenhousev1alpha1.PluginPreset,
) error {

	// List all Plugins that are managed by the PluginPreset.
	plugins, err := r.listPlugins(ctx, preset)
	if err != nil {
		return err
	}

	pluginStatuses := make([]greenhousev1alpha1.ManagedPluginStatus, 0, len(plugins.Items))
	readyPluginsCount := 0
	failedPluginsCount := 0

	for _, plugin := range plugins.Items {
		pluginReadyCondition := plugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

		switch {
		case pluginReadyCondition == nil:
			// Plugin exists, but its Ready condition is not set - treat it as unavailable.
			continue
		case pluginReadyCondition.IsTrue():
			readyPluginsCount++
		default:
			failedPluginsCount++
		}

		currentPluginStatus := greenhousev1alpha1.ManagedPluginStatus{PluginName: plugin.Name, ReadyCondition: *pluginReadyCondition}
		pluginStatuses = append(pluginStatuses, currentPluginStatus)
	}

	preset.Status.PluginStatuses = pluginStatuses
	preset.Status.TotalPlugins = len(pluginStatuses)
	preset.Status.ReadyPlugins = readyPluginsCount
	preset.Status.FailedPlugins = failedPluginsCount

	// Set AllPluginsReadyCondition based on plugin readiness.
	allPluginsReady := len(pluginStatuses) > 0 && readyPluginsCount == len(pluginStatuses)
	if allPluginsReady {
		preset.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllPluginsReadyCondition, "", "All plugins are ready"))
	} else {
		preset.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.AllPluginsReadyCondition, "", fmt.Sprintf("%d of %d plugins are ready", readyPluginsCount, len(pluginStatuses))))
	}

	return nil
}

func isPluginManagedByPreset(plugin *greenhousev1alpha1.Plugin, presetName string) bool {
	return plugin.Labels[greenhouseapis.LabelKeyPluginPreset] == presetName
}

func overridesPluginOptionValues(plugin *greenhousev1alpha1.Plugin, preset *greenhousev1alpha1.PluginPreset) {
	index := slices.IndexFunc(preset.Spec.ClusterOptionOverrides, func(override greenhousev1alpha1.ClusterOptionOverride) bool {
		return override.ClusterName == plugin.Spec.ClusterName
	})

	// when plugin is running on different cluster then defined in
	if index == -1 {
		return
	}

	// overrides value
	for _, overrideValue := range preset.Spec.ClusterOptionOverrides[index].Overrides {
		valueIndex := slices.IndexFunc(plugin.Spec.OptionValues, func(value greenhousev1alpha1.PluginOptionValue) bool {
			return value.Name == overrideValue.Name
		})

		if valueIndex == -1 {
			plugin.Spec.OptionValues = append(plugin.Spec.OptionValues, overrideValue)
		} else {
			plugin.Spec.OptionValues[valueIndex] = overrideValue
		}
	}
}

// generatePluginName generates a name for a plugin based on the used PluginPreset's name and the Cluster.
func generatePluginName(p *greenhousev1alpha1.PluginPreset, cluster *greenhousev1alpha1.Cluster) string {
	return buildPluginName(p.Name, cluster.GetName())
}

// buildPluginName takes PluginPreset name and Cluster name to create a name for the Plugin.
func buildPluginName(pluginPresetName, clusterName string) string {
	return fmt.Sprintf("%s-%s", pluginPresetName, clusterName)
}

func initPluginPresetStatus(p *greenhousev1alpha1.PluginPreset) {
	for _, ct := range presetExposedConditions {
		if p.Status.GetConditionByType(ct) == nil {
			p.SetCondition(greenhousemetav1alpha1.UnknownCondition(ct, "", ""))
		}
	}
}

// computeReadyCondition computes the ReadyCondition based on the PluginPreset's StatusConditions.
func (r *PluginPresetReconciler) computeReadyCondition(
	conditions greenhousemetav1alpha1.StatusConditions,
) (readyCondition greenhousemetav1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	if conditions.GetConditionByType(greenhousev1alpha1.PluginFailedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Plugin reconciliation failed"
		return readyCondition
	}

	if conditions.GetConditionByType(greenhousemetav1alpha1.ClusterListEmpty).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "No cluster matches ClusterSelector"
		return readyCondition
	}

	allPluginsReadyCondition := conditions.GetConditionByType(greenhousev1alpha1.AllPluginsReadyCondition)
	if allPluginsReadyCondition != nil && allPluginsReadyCondition.IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = allPluginsReadyCondition.Message
		return readyCondition
	}

	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

// listPlugins returns the list of plugins for the given PluginPreset
func (r *PluginPresetReconciler) listPlugins(ctx context.Context, pb *greenhousev1alpha1.PluginPreset) (*greenhousev1alpha1.PluginList, error) {
	var plugins = new(greenhousev1alpha1.PluginList)
	if err := r.List(ctx, plugins, client.InNamespace(pb.GetNamespace()), client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: pb.Name}); err != nil {
		return nil, err
	}
	return plugins, nil
}

func (r *PluginPresetReconciler) listClusters(ctx context.Context, pb *greenhousev1alpha1.PluginPreset) (*greenhousev1alpha1.ClusterList, error) {
	clusterSelector, err := metav1.LabelSelectorAsSelector(&pb.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(greenhousev1alpha1.ClusterList)
	if err := r.List(ctx, clusters, client.InNamespace(pb.GetNamespace()), client.MatchingLabelsSelector{Selector: clusterSelector}); err != nil {
		return nil, err
	}
	return clusters, nil
}

// cleanupPlugins deletes all Plugins managed by the PluginDefinition, where the Cluster is not in the list of Clusters.
func (r *PluginPresetReconciler) cleanupPlugins(ctx context.Context, pb *greenhousev1alpha1.PluginPreset, cl *greenhousev1alpha1.ClusterList) (err error) {
	plugins, err := r.listPlugins(ctx, pb)
	if err != nil {
		return err
	}
	for _, p := range plugins.Items {
		validCluster := slices.ContainsFunc(cl.Items, func(c greenhousev1alpha1.Cluster) bool {
			return p.Spec.ClusterName == c.GetName()
		})
		if !validCluster {
			if err := r.Delete(ctx, &p); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			r.recorder.Eventf(&p, corev1.EventTypeNormal, "PluginDeleted", "Dangling Plugin %s deleted by PluginPreset %s", p.Name, pb.Name)
			ctrl.LoggerFrom(ctx).Info("Dangling Plugin deleted", "plugin", p.Name, "pluginPreset", pb.Name)
		}
	}
	return nil
}

// enqueueAllPluginPresetsInNamespace returns a list of reconcile requests for all PluginPresets in the same namespace as obj.
func (r *PluginPresetReconciler) enqueueAllPluginPresetsInNamespace(ctx context.Context, obj client.Object) []ctrl.Request {
	return listPluginPresetAsReconcileRequests(ctx, r.Client, client.InNamespace(obj.GetNamespace()))
}

// listPluginPresetsAsReconcileRequests returns a list of reconcile requests for all PluginPresets that match the given list options.
func listPluginPresetAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var allPluginPresets = new(greenhousev1alpha1.PluginPresetList)
	if err := c.List(ctx, allPluginPresets, listOpts...); err != nil {
		return nil
	}
	requests := make([]ctrl.Request, len(allPluginPresets.Items))
	for i, pluginPreset := range allPluginPresets.Items {
		requests[i] = ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pluginPreset.DeepCopy())}
	}
	return requests
}

// getReleaseName determines the release name for a plugin based on its current state and the preset.
func getReleaseName(plugin *greenhousev1alpha1.Plugin, preset *greenhousev1alpha1.PluginPreset) string {
	switch {
	case plugin.Spec.ReleaseName != "":
		// If the plugin already has a release name, keep it.
		return plugin.Spec.ReleaseName
	case plugin.Status.HelmReleaseStatus != nil && plugin.Spec.ReleaseName == "":
		// If the plugin has a HelmReleaseStatus but no release name, set the release name to the plugin name. This is to avoid validation errors when the plugin is created, since the plugin is already deployed with the plugin name as the release name.
		return plugin.Name
	default:
		return preset.Spec.Plugin.ReleaseName
	}
}
