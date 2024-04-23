// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Owns(&greenhousev1alpha1.Plugin{}).
		// Clusters and teams are passed as values to each Helm operation. Reconcile on change.
		Watches(&greenhousev1alpha1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllPluginPresetsInNamespace),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *PluginPresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var pluginPreset = new(greenhousev1alpha1.PluginPreset)
	if err := r.Get(ctx, req.NamespacedName, pluginPreset); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

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
			return ctrl.Result{}, fmt.Errorf("failed to delete plugins for %s/%s: %v", pluginPreset.Namespace, pluginPreset.Name, errors.Join(allErrs...))
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
	// TODO: GC plugins that are not in the list of clusters anymore
	plugins, err := r.listPlugins(ctx, pluginPreset)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.cleanupPlugins(ctx, clusters, plugins); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcilePluginPreset(ctx, pluginPreset, clusters); err != nil {
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

func (r *PluginPresetReconciler) reconcilePluginPreset(ctx context.Context, pb *greenhousev1alpha1.PluginPreset, clusters *greenhousev1alpha1.ClusterList) error {
	var allErrs = make([]error, 0)

	for _, cluster := range clusters.Items {
		plugin := &greenhousev1alpha1.Plugin{
			ObjectMeta: v1.ObjectMeta{
				Name:      generatePluginName(pb.Spec.Plugin.PluginDefinition, &cluster),
				Namespace: pb.GetNamespace(),
			},
		}
		result, err := clientutil.CreateOrPatch(ctx, r.Client, plugin, func() error {
			// Label the plugin with the managed resource label to identify it as managed by the PluginPreset.
			plugin.SetLabels(map[string]string{greenhouseapis.LabelKeyPluginPreset: pb.Name})
			// Set the owner reference to the PluginPreset. This is used to trigger reconciliation, if the managed Plugin is modified.
			if err := controllerutil.SetControllerReference(pb, plugin, r.Scheme()); err != nil {
				return err
			}
			plugin.Spec = pb.Spec.Plugin
			// Set the cluster name to the name of the cluster. The PluginSpec contained in the PluginPreset does not have a cluster name.
			plugin.Spec.ClusterName = cluster.GetName()
			return nil
		})
		switch result {
		// TODO: Handle the result. Log and emit event.
		}
		allErrs = append(allErrs, err)
	}
	return utilerrors.NewAggregate(allErrs)
}

// generatePluginName generates a name for a plugin based on the used PluginDefinition and the Cluster.
func generatePluginName(pluginDefinition string, cluster *greenhousev1alpha1.Cluster) string {
	return fmt.Sprintf("%s-%s", pluginDefinition, cluster.GetName())
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
	clusterSelector, err := v1.LabelSelectorAsSelector(&pb.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	var clusters = new(greenhousev1alpha1.ClusterList)
	if err := r.List(ctx, clusters, client.InNamespace(pb.GetNamespace()), client.MatchingLabelsSelector{Selector: clusterSelector}); err != nil {
		return nil, err
	}
	return clusters, nil
}

// enqueueAllPluginPresets returns a list of reconcile requests for all PluginPresets in the same namespace as obj.
func (r *PluginPresetReconciler) enqueueAllPluginPresetsInNamespace(ctx context.Context, obj client.Object) []ctrl.Request {
	return listPluginsAsReconcileRequests(ctx, r.Client, client.InNamespace(obj.GetNamespace()))
}

// listPluginsAsReconcileRequests returns a list of reconcile requests for all PluginPresets that match the given list options.
func listPluginPresetAsReconcileEvents(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var allPluginPresets = new(greenhousev1alpha1.PluginPresetList)
	if err := c.List(ctx, allPluginPresets); err != nil {
		return nil
	}
	requests := make([]ctrl.Request, len(allPluginPresets.Items))
	for i, pluginPreset := range allPluginPresets.Items {
		requests[i] = ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pluginPreset.DeepCopy())}
	}
	return requests
}
