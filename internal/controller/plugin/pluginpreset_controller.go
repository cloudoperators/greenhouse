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
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

// presetExposedConditions contains the conditions that are exposed in the PluginPreset's StatusConditions.
var presetExposedConditions = []greenhouseapis.ConditionType{
	greenhouseapis.ReadyCondition,
	greenhousev1alpha1.PluginSkippedCondition,
	greenhousev1alpha1.PluginFailedCondition,
	greenhouseapis.ClusterListEmpty,
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
		pluginPreset.SetCondition(readyCondition)
	}
}

func (r *PluginPresetReconciler) EnsureCreated(ctx context.Context, resource lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginPreset := resource.(*greenhousev1alpha1.PluginPreset) //nolint:errcheck

	_ = log.FromContext(ctx)

	initPluginPresetStatus(pluginPreset)

	clusters, err := r.listClusters(ctx, pluginPreset)
	if err != nil {
		pluginPreset.SetCondition(greenhouseapis.TrueCondition(greenhouseapis.ClusterListEmpty, "", fmt.Sprintf("Invalid ClusterSelector: %v", err.Error())))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	switch {
	case len(clusters.Items) == 0:
		pluginPreset.SetCondition(greenhouseapis.TrueCondition(greenhouseapis.ClusterListEmpty, "", "No cluster matches ClusterSelector"))
	default:
		pluginPreset.SetCondition(greenhouseapis.FalseCondition(greenhouseapis.ClusterListEmpty, "", ""))
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

	// Cleanup the plugins that are managed by this PluginPreset.
	plugins, err := r.listPlugins(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	allErrs := make([]error, 0)
	for _, plugin := range plugins.Items {
		if err := r.Client.Delete(ctx, &plugin); err != nil && !apierrors.IsNotFound(err) {
			allErrs = append(allErrs, err)
		}
	}

	// If there are still plugins left, requeue the deletion.
	if len(allErrs) > 0 {
		return ctrl.Result{}, lifecycle.Pending, fmt.Errorf("failed to delete plugins for %s/%s: %w", pluginPreset.Namespace, pluginPreset.Name, errors.Join(allErrs...))
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

// reconcilePluginPreset reconciles the PluginPreset by creating or updating the Plugins for the given clusters.
// It skips reconciliation for Plugins that do not have the labels of the PluginPreset.
func (r *PluginPresetReconciler) reconcilePluginPreset(ctx context.Context, preset *greenhousev1alpha1.PluginPreset, clusters *greenhousev1alpha1.ClusterList) error {
	var allErrs = make([]error, 0)
	var skippedPlugins = make([]string, 0)
	var failedPlugins = make([]string, 0)

	pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
	err := r.Get(ctx, client.ObjectKey{Name: preset.Spec.Plugin.PluginDefinition}, pluginDefinition)
	if err != nil {
		allErrs = append(allErrs, err)
		return utilerrors.NewAggregate(allErrs)
	}

	for _, cluster := range clusters.Items {
		plugin := &greenhousev1alpha1.Plugin{}
		err := r.Get(ctx, client.ObjectKey{Namespace: preset.GetNamespace(), Name: generatePluginName(preset, &cluster)}, plugin)

		switch {
		case !cluster.DeletionTimestamp.IsZero():
			continue
		case err == nil:
			// The Plugin exists but does not contain the labels of the PluginPreset. This Plugin is not managed by the PluginPreset and must not be touched.
			if shouldSkipPlugin(plugin, preset, pluginDefinition, cluster.Name) {
				skippedPlugins = append(skippedPlugins, plugin.Name)
				continue
			}

		case apierrors.IsNotFound(err):
			plugin = &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      generatePluginName(preset, &cluster),
					Namespace: preset.GetNamespace(),
				},
			}
		default:
			return err
		}

		_, err = clientutil.CreateOrPatch(ctx, r.Client, plugin, func() error {
			// Label the plugin with the managed resource label to identify it as managed by the PluginPreset.
			plugin.SetLabels(map[string]string{greenhouseapis.LabelKeyPluginPreset: preset.Name})
			// Set the owner reference to the PluginPreset. This is used to trigger reconciliation, if the managed Plugin is modified.
			if err := controllerutil.SetControllerReference(preset, plugin, r.Scheme()); err != nil {
				return err
			}
			plugin.Spec = preset.Spec.Plugin
			// Set the cluster name to the name of the cluster. The PluginSpec contained in the PluginPreset does not have a cluster name.
			plugin.Spec.ClusterName = cluster.GetName()

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
		preset.SetCondition(greenhouseapis.TrueCondition(greenhousev1alpha1.PluginSkippedCondition, "", "Skipped existing plugins: "+strings.Join(skippedPlugins, ", ")))
	default:
		preset.SetCondition(greenhouseapis.FalseCondition(greenhousev1alpha1.PluginSkippedCondition, "", ""))
	}
	switch {
	case len(failedPlugins) > 0:
		preset.SetCondition(greenhouseapis.TrueCondition(greenhousev1alpha1.PluginFailedCondition, greenhousev1alpha1.PluginReconcileFailed, strings.Join(failedPlugins, "; ")))
	default:
		preset.SetCondition(greenhouseapis.FalseCondition(greenhousev1alpha1.PluginFailedCondition, "", ""))
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
		pluginReadyCondition := plugin.Status.GetConditionByType(greenhouseapis.ReadyCondition)

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
	preset.Status.AvailablePlugins = len(pluginStatuses)
	preset.Status.ReadyPlugins = readyPluginsCount
	preset.Status.FailedPlugins = failedPluginsCount
	return nil
}

func isPluginManagedByPreset(plugin *greenhousev1alpha1.Plugin, presetName string) bool {
	return plugin.Labels[greenhouseapis.LabelKeyPluginPreset] == presetName
}

func shouldSkipPlugin(plugin *greenhousev1alpha1.Plugin, preset *greenhousev1alpha1.PluginPreset, definition *greenhousev1alpha1.PluginDefinition, clusterName string) bool {
	if !isPluginManagedByPreset(plugin, preset.Name) {
		return true
	}

	// need to reconcile when plugin labels has been changed
	for _, override := range preset.Spec.ClusterOptionOverrides {
		if override.ClusterName != clusterName {
			continue
		}

		for _, overrideOptionValue := range override.Overrides {
			if !slices.ContainsFunc(plugin.Spec.OptionValues, func(item greenhousev1alpha1.PluginOptionValue) bool {
				return equalPluginOptions(overrideOptionValue, item)
			}) {
				return false
			}
		}
	}

	// need to reconcile when plugin does not have option which exists in plugin preset
	for _, presetOptionValue := range preset.Spec.Plugin.OptionValues {
		if !slices.ContainsFunc(plugin.Spec.OptionValues, func(item greenhousev1alpha1.PluginOptionValue) bool {
			return equalPluginOptions(presetOptionValue, item)
		}) {
			return false
		}
	}

	for _, pluginOption := range plugin.Spec.OptionValues {
		if strings.HasPrefix(pluginOption.Name, "global.greenhouse") {
			// pluginOption is a global option, nothing to do
			continue
		}

		if slices.ContainsFunc(preset.Spec.Plugin.OptionValues, func(item greenhousev1alpha1.PluginOptionValue) bool {
			return equalPluginOptions(item, pluginOption)
		}) {
			// optionValue is set by the PluginPreset, nothing to doen plugin does not have option which exists in plugin p
			continue
		}
		if slices.ContainsFunc(definition.Spec.Options, func(item greenhousev1alpha1.PluginOption) bool {
			if item.Default == nil {
				return false
			}
			if pluginOption.ValueFrom != nil {
				// PluginDefinition does not support valueFrom for default values
				return false
			}
			return item.Name == pluginOption.Name && string(item.Default.Raw) == string(pluginOption.Value.Raw)
		}) {
			// optionValue is set by the PluginDefinition, nothing to do
			continue
		}
		// the optionValue is not a global option, not set by the PluginPreset and not set by the PluginDefinition
		// need to reconcile to get the managed Plugin back into the desired state
		return false
	}
	// all options are global options or set by the PluginPreset or the PluginDefinition
	return true
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
	return fmt.Sprintf("%s-%s", p.Name, cluster.GetName())
}

func initPluginPresetStatus(p *greenhousev1alpha1.PluginPreset) {
	for _, ct := range presetExposedConditions {
		if p.Status.GetConditionByType(ct) == nil {
			p.SetCondition(greenhouseapis.UnknownCondition(ct, "", ""))
		}
	}
}

// computeReadyCondition computes the ReadyCondition based on the PluginPreset's StatusConditions.
func (r *PluginPresetReconciler) computeReadyCondition(
	conditions greenhouseapis.StatusConditions,
) (readyCondition greenhouseapis.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhouseapis.ReadyCondition)

	if conditions.GetConditionByType(greenhousev1alpha1.PluginFailedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Plugin reconciliation failed"
		return readyCondition
	}

	if conditions.GetConditionByType(greenhousev1alpha1.PluginSkippedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Existing plugins skipped"
		return readyCondition
	}

	if conditions.GetConditionByType(greenhouseapis.ClusterListEmpty).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "No cluster matches ClusterSelector"
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
			if err := r.Client.Delete(ctx, &p); err != nil && !apierrors.IsNotFound(err) {
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

// equalPluginOptions compares two PluginOptionValue objects.
func equalPluginOptions(a, b greenhousev1alpha1.PluginOptionValue) bool {
	if a.Name != b.Name {
		return false
	}
	valueNil := a.Value == nil && b.Value == nil
	valueFromNil := a.ValueFrom == nil && b.ValueFrom == nil
	switch {
	case valueNil && valueFromNil:
		return true

	case !valueNil:
		return a.ValueJSON() == b.ValueJSON()

	case !valueFromNil:
		return a.ValueFrom.Secret.Name == b.ValueFrom.Secret.Name &&
			a.ValueFrom.Secret.Key == b.ValueFrom.Secret.Key
	default:
		return false
	}
}
