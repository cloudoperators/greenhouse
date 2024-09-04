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
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	chartTestRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_plugin_chart_test_runs_total",
			Help: "Total number of Helm Chart test runs with their results",
		},
		[]string{"cluster", "plugin", "namespace", "result"})
)

func init() {
	metrics.Registry.MustRegister(chartTestRunsTotal)
}

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

	// Nothing to do when the status of the plugin is empty and when the plugin does not have a Helm Chart
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) || plugin.Status.HelmChart == nil {
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

	clusterAccessReadyCondition, restClientGetter := initClientGetter(ctx, r.Client, r.kubeClientOpts, plugin, plugin.Status)
	pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
	if !clusterAccessReadyCondition.IsTrue() {
		return ctrl.Result{}, fmt.Errorf("cannot access cluster: %s", clusterAccessReadyCondition.Message)
	}

	hasHelmChartTest, err := helm.HelmChartTest(ctx, restClientGetter, &plugin)
	prometheusLabels := prometheus.Labels{
		"cluster":   plugin.Spec.ClusterName,
		"plugin":    plugin.Name,
		"namespace": plugin.Namespace,
	}
	if err != nil {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionFalse
		errStr := fmt.Sprintf("Helm Chart testing failed: %s. To debug, please run `helm test %s`command in your remote cluster %s.", err.Error(), plugin.Name, plugin.Spec.ClusterName)
		errStr = strings.ReplaceAll(errStr, "\n", "")
		errStr = strings.ReplaceAll(errStr, "\t", " ")
		errStr = strings.ReplaceAll(errStr, "*", "")
		noHelmChartTestFailuresCondition.Message = errStr

		prometheusLabels["result"] = "Error"
		chartTestRunsTotal.With(prometheusLabels).Inc()

		return ctrl.Result{}, err
	}

	if !hasHelmChartTest {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionTrue
		noHelmChartTestFailuresCondition.Message = "No Helm Chart Tests defined by the PluginDefinition"

		prometheusLabels["result"] = "NoTests"
		chartTestRunsTotal.With(prometheusLabels).Inc()
	} else {
		noHelmChartTestFailuresCondition.Status = metav1.ConditionTrue
		noHelmChartTestFailuresCondition.Message = "Helm Chart Test is successful"

		prometheusLabels["result"] = "Success"
		chartTestRunsTotal.With(prometheusLabels).Inc()
	}

	return ctrl.Result{}, nil
}
