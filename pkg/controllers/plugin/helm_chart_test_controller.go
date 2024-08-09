// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"reflect"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HelmChartTestReconciler struct {
	client.Client
	recorder        record.EventRecorder
	kubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status,verbs=get;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list

// SetupWithManager sets up the controller with the Manager.
func (r *HelmChartTestReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.kubeClientOpts = []clientutil.KubeClientOption{
		clientutil.WithRuntimeOptions(r.kubeRuntimeOpts),
		clientutil.WithPersistentConfig(),
	}

	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Plugin{}).
		Complete(r)
}

func (r *HelmChartTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// logic goes here
	fmt.Printf("Reconciling Plugin from HelmChartTestReconciler: %s\n", req.NamespacedName)
	var plugin greenhousev1alpha1.Plugin
	if err := r.Get(ctx, req.NamespacedName, &plugin); err != nil {
		fmt.Printf("Error getting plugin: %v", err)
	}

	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) {
		fmt.Printf("Plugin status is empty. Skipping reconcile")
		return ctrl.Result{}, nil
	}

	if plugin.Status.HelmReleaseStatus.Status != "deployed" {
		// Helm Chart Test cannot be done as the Helm Chart is not yet deployed.
		return ctrl.Result{}, nil
	}

	pluginStatus := initPluginStatus(&plugin)

	clusterAccessReadyCondition, restClientGetter := r.initClientGetter(ctx, plugin, plugin.Status)
	pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
	if !clusterAccessReadyCondition.IsTrue() {
		return ctrl.Result{}, fmt.Errorf("cannot access cluster: %s", clusterAccessReadyCondition.Message)
	}

	err := helm.HelmChartTest(ctx, restClientGetter, &plugin)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// TODO: This is a duplicate. Move this as a "function" instead of a "method" to a common file in the same package
func (r *HelmChartTestReconciler) initClientGetter(
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
		restClientGetter, err = clientutil.NewRestClientGetterForInCluster(plugin.GetReleaseNamespace(), r.kubeClientOpts...)
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
	restClientGetter, err = clientutil.NewRestClientGetterFromSecret(&secret, plugin.GetReleaseNamespace(), r.kubeClientOpts...)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("cannot access cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}
	clusterAccessReadyCondition.Status = metav1.ConditionTrue
	clusterAccessReadyCondition.Message = ""
	return clusterAccessReadyCondition, restClientGetter
}
