// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

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
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	var plugin greenhousev1alpha1.Plugin
	if err := r.Get(ctx, req.NamespacedName, &plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Nothing to do when the status of the plugin is empty
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) {
		return ctrl.Result{}, nil
	}

	// Helm Chart Test cannot be done as the Helm Chart deployment is not successful
	if plugin.Status.HelmReleaseStatus.Status != "deployed" {
		return ctrl.Result{Requeue: true}, nil
	}

	pluginStatus := initPluginStatus(&plugin)

	noHelmChartTestFailuresCondition := *pluginStatus.GetConditionByType(greenhousev1alpha1.NoHelmChartTestFailuresCondition)

	defer func() {
		_, err := clientutil.PatchStatus(ctx, r.Client, &plugin, func() error {
			plugin.Status.StatusConditions.SetConditions(noHelmChartTestFailuresCondition)
			return nil
		})
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to set status")
		}
	}()

	clusterAccessReadyCondition, restClientGetter := r.initClientGetter(ctx, plugin, plugin.Status)
	pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
	if !clusterAccessReadyCondition.IsTrue() {
		return ctrl.Result{}, fmt.Errorf("cannot access cluster: %s", clusterAccessReadyCondition.Message)
	}

	hasHelmChartTest, err := helm.HelmChartTest(ctx, restClientGetter, &plugin)
	if err != nil {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionFalse
		errStr := fmt.Sprintf("Helm Chart testing failed: %s. To debug, please run `helm test %s`command in your remote cluster %s.", err.Error(), plugin.Name, plugin.Spec.ClusterName)
		errStr = strings.ReplaceAll(errStr, "\n", "")
		errStr = strings.ReplaceAll(errStr, "\t", " ")
		errStr = strings.ReplaceAll(errStr, "*", "")
		noHelmChartTestFailuresCondition.Message = errStr

		return ctrl.Result{}, err
	}

	if !hasHelmChartTest {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionTrue
		noHelmChartTestFailuresCondition.Message = "Helm Chart Test is not defined"
	} else {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionTrue
		noHelmChartTestFailuresCondition.Message = "Helm Chart Test is successful"
	}

	return ctrl.Result{}, nil
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