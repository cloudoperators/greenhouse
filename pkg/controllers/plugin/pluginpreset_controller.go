// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

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

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// presetExposedConditions contains the conditions that are exposed in the PluginPreset's StatusConditions.
var presetExposedConditions = []greenhousev1alpha1.ConditionType{
	greenhousev1alpha1.ReadyCondition,
	greenhousev1alpha1.PluginSkippedCondition,
	greenhousev1alpha1.PluginFailedCondition,
	greenhousev1alpha1.ClusterListEmpty,
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

// SetupWithManager sets up the controller with the Manager.
func (r *PluginPresetReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginPreset{}).
		Owns(&greenhousev1alpha1.Plugin{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginPresetsInNamespace),
			builder.WithPredicates(predicate.LabelChangedPredicate{})).
		Complete(r)
}

func (r *PluginPresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var pluginPreset = new(greenhousev1alpha1.PluginPreset)
	if err := r.Get(ctx, req.NamespacedName, pluginPreset); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pluginPresetStatus := initPluginPresetStatus(pluginPreset)

	defer func() {
		if statusErr := r.setStatus(ctx, pluginPreset, pluginPresetStatus); statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "failed to set status")
		}
	}()

	if pluginPreset.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(pluginPreset, greenhouseapis.FinalizerCleanupPluginPreset) {
		// Cleanup the plugins that are managed by this PluginPreset.
		plugins, err := r.listPlugins(ctx, pluginPreset)
		if err != nil {
			return ctrl.Result{}, err
		}
		allErrs := make([]error, 0)
		for _, plugin := range plugins.Items {
			if err := r.Client.Delete(ctx, &plugin); err != nil && !apierrors.IsNotFound(err) {
				allErrs = append(allErrs, err)
			}
		}

		// If there are still plugins left, requeue the deletion.
		if len(allErrs) > 0 {
			return ctrl.Result{}, fmt.Errorf("failed to delete plugins for %s/%s: %w", pluginPreset.Namespace, pluginPreset.Name, errors.Join(allErrs...))
		}

		// Remove the finalizer to allow for deletion.
		if err := clientutil.RemoveFinalizer(ctx, r.Client, pluginPreset, greenhouseapis.FinalizerCleanupPluginPreset); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err := clientutil.EnsureFinalizer(ctx, r.Client, pluginPreset, greenhouseapis.FinalizerCleanupPluginPreset); err != nil {
		return ctrl.Result{}, err
	}

	clusters, err := r.listClusters(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, err
	}

	switch {
	case len(clusters.Items) == 0:
		pluginPresetStatus.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ClusterListEmpty, "", "No cluster matches ClusterSelector"))
	default:
		pluginPresetStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ClusterListEmpty, "", ""))
	}

	plugins, err := r.listPlugins(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.cleanupPlugins(ctx, clusters, plugins); err != nil {
		return ctrl.Result{}, err
	}

	skippedCondition, failedCondition, err := r.reconcilePluginPreset(ctx, pluginPreset, clusters, pluginPresetStatus)
	pluginPresetStatus.SetConditions(skippedCondition, failedCondition)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// cleanupPlugins deletes all plugins where the clusterName is not included in the ClusterList.
func (r *PluginPresetReconciler) cleanupPlugins(ctx context.Context, clusters *greenhousev1alpha1.ClusterList, plugins *greenhousev1alpha1.PluginList) error {
	var allErrs = make([]error, 0)
	for _, plugin := range plugins.Items {
		if !slices.ContainsFunc(clusters.Items, func(cluster greenhousev1alpha1.Cluster) bool {
			return cluster.GetName() == plugin.Spec.ClusterName
		}) {
			if err := r.Client.Delete(ctx, &plugin); err != nil && !apierrors.IsNotFound(err) {
				allErrs = append(allErrs, err)
			}
		}
	}
	return errors.Join(allErrs...)
}

// reconcilePluginPreset reconciles the PluginPreset by creating or updating the Plugins for the given clusters.
// It skips reconciliation for Plugins that do not have the labels of the PluginPreset.
func (r *PluginPresetReconciler) reconcilePluginPreset(ctx context.Context, preset *greenhousev1alpha1.PluginPreset, clusters *greenhousev1alpha1.ClusterList, status greenhousev1alpha1.PluginPresetStatus) (skippedCondition, failedCondition greenhousev1alpha1.Condition, err error) {
	var allErrs = make([]error, 0)
	var skippedPlugins = make([]string, 0)
	var failedPlugins = make([]string, 0)

	failedCondition = *status.GetConditionByType(greenhousev1alpha1.PluginFailedCondition)
	skippedCondition = *status.GetConditionByType(greenhousev1alpha1.PluginSkippedCondition)

	for _, cluster := range clusters.Items {
		plugin := &greenhousev1alpha1.Plugin{}
		err := r.Get(ctx, client.ObjectKey{Namespace: preset.GetNamespace(), Name: generatePluginName(preset, &cluster)}, plugin)

		switch {
		case err == nil:
			// The Plugin exists but does not contain the labels of the PluginPreset. This Plugin is not managed by the PluginPreset and must not be touched.
			if plugin.Labels[greenhouseapis.LabelKeyPluginPreset] != preset.Name {
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
			return skippedCondition, failedCondition, err
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
			failedPlugins = append(failedPlugins, plugin.Name)
			allErrs = append(allErrs, err)
		}
	}
	switch {
	case len(skippedPlugins) > 0:
		skippedCondition.Status = metav1.ConditionTrue
		skippedCondition.Message = "Skipped existing plugins: " + strings.Join(skippedPlugins, ", ")
	default:
		skippedCondition.Status = metav1.ConditionFalse
		skippedCondition.Message = ""
	}
	switch {
	case len(failedPlugins) > 0:
		failedCondition.Status = metav1.ConditionTrue
		failedCondition.Message = "Failed to reconcile plugins: " + strings.Join(failedPlugins, ", ")
	default:
		failedCondition.Status = metav1.ConditionFalse
		failedCondition.Message = ""
	}
	return skippedCondition, failedCondition, utilerrors.NewAggregate(allErrs)
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
			return true
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

func initPluginPresetStatus(p *greenhousev1alpha1.PluginPreset) greenhousev1alpha1.PluginPresetStatus {
	presetStatus := p.Status.DeepCopy()
	for _, t := range presetExposedConditions {
		if presetStatus.GetConditionByType(t) == nil {
			presetStatus.SetConditions(greenhousev1alpha1.UnknownCondition(t, "", ""))
		}
	}
	return *presetStatus
}

func (r *PluginPresetReconciler) setStatus(ctx context.Context, p *greenhousev1alpha1.PluginPreset, status greenhousev1alpha1.PluginPresetStatus) error {
	readyCondition := r.computeReadyCondition(status.StatusConditions)
	status.StatusConditions.SetConditions(readyCondition)
	_, err := clientutil.PatchStatus(ctx, r.Client, p, func() error {
		p.Status = status
		return nil
	})
	return err
}

// computeReadyCondition computes the ReadyCondition based on the PluginPreset's StatusConditions.
func (r *PluginPresetReconciler) computeReadyCondition(
	conditions greenhousev1alpha1.StatusConditions,
) (readyCondition greenhousev1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)

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

	if conditions.GetConditionByType(greenhousev1alpha1.ClusterListEmpty).IsTrue() {
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
