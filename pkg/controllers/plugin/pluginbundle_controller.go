// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// PluginBundleReconciler reconciles a PluginBundle object
type PluginBundleReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginbundles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginbundles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginbundles/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=pluginconfigs,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *PluginBundleReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.PluginBundle{}).
		Complete(r)
}

func (r *PluginBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var pluginBundle = new(greenhousev1alpha1.PluginBundle)
	if err := r.Get(ctx, req.NamespacedName, pluginBundle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcilePluginBundle(ctx, pluginBundle); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PluginBundleReconciler) reconcilePluginBundle(ctx context.Context, pluginBundle *greenhousev1alpha1.PluginBundle) error {
	lblSelector, err := v1.LabelSelectorAsSelector(&pluginBundle.Spec.ClusterSelector)
	if err != nil {
		return err
	}
	var selectedClusters = new(greenhousev1alpha1.ClusterList)
	if err := r.List(ctx, selectedClusters, client.InNamespace(pluginBundle.GetNamespace()), client.MatchingLabelsSelector{Selector: lblSelector}); err != nil {
		return err
	}

	var allErrs = make([]error, 0)
	for _, pluginConfigSpec := range pluginBundle.Spec.Plugins {
		for _, cluster := range selectedClusters.Items {
			var pluginConfigForCluster = new(greenhousev1alpha1.Plugin)
			pluginConfigForCluster.Namespace = pluginBundle.GetNamespace()
			pluginConfigForCluster.Name = makeNameForPluginConfigFromPluginBundle(&pluginConfigSpec, &cluster)
			result, err := clientutil.CreateOrPatch(ctx, r.Client, pluginConfigForCluster, func() error {
				pluginConfigForCluster.Spec = pluginConfigSpec
				pluginConfigForCluster.Spec.ClusterName = cluster.GetName()
				return nil
			})
			switch result {
			// TODO: Handle the result. Log and emit event.
			}
			allErrs = append(allErrs, err)
		}
	}
	return utilerrors.NewAggregate(allErrs)
}

func makeNameForPluginConfigFromPluginBundle(pluginSpec *greenhousev1alpha1.PluginSpec, cluster *greenhousev1alpha1.Cluster) string {
	return fmt.Sprintf("%s-%s", pluginSpec.PluginDefinition, cluster.GetName())
}
