// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/helm"
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

func (r *PluginReconciler) reconcileHelmChartTest(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (*reconcileResult, error) {
	// Nothing to do when the status of the plugin is empty and when the plugin does not have a Helm Chart
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) || plugin.Status.HelmChart == nil {
		return nil, nil
	}

	if plugin.Spec.Disabled {
		return nil, nil
	}

	// Helm Chart Test cannot be done as the Helm Chart deployment is not successful
	if plugin.Status.HelmReleaseStatus.Status != "deployed" {
		return nil, nil
	}

	restClientGetter, err := initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
	if err != nil {
		return nil, fmt.Errorf("cannot access cluster: %s", err.Error())
	}

	// Check if we should continue with reconciliation or requeue if cluster is scheduled for deletion
	result, err := shouldReconcileOrRequeue(ctx, r.Client, plugin)
	if err != nil {
		return nil, err
	}
	if result != nil {
		return &reconcileResult{requeueAfter: result.requeueAfter}, nil
	}

	hasHelmChartTest, err := helm.ChartTest(restClientGetter, plugin)
	prometheusLabels := prometheus.Labels{
		"cluster":   plugin.Spec.ClusterName,
		"plugin":    plugin.Name,
		"namespace": plugin.Namespace,
	}
	if err != nil {
		errStr := fmt.Sprintf("Helm Chart testing failed: %s. To debug, please run `helm test %s`command in your remote cluster %s.", err.Error(), plugin.Name, plugin.Spec.ClusterName)
		errStr = strings.ReplaceAll(errStr, "\n", "")
		errStr = strings.ReplaceAll(errStr, "\t", " ")
		errStr = strings.ReplaceAll(errStr, "*", "")
		plugin.SetCondition(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "", errStr))

		prometheusLabels["result"] = "Error"
		chartTestRunsTotal.With(prometheusLabels).Inc()

		return nil, err
	}

	if !hasHelmChartTest {
		plugin.SetCondition(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "",
			"No Helm Chart Tests defined by the PluginDefinition"))

		prometheusLabels["result"] = "NoTests"
		chartTestRunsTotal.With(prometheusLabels).Inc()
	} else {
		plugin.SetCondition(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "",
			"Helm Chart Test is successful"))

		prometheusLabels["result"] = "Success"
		chartTestRunsTotal.With(prometheusLabels).Inc()
	}

	return nil, nil
}
