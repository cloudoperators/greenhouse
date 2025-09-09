// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"bufio"
	"context"
	"fmt"
	"reflect"
	"strings"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
)

func (r *PluginReconciler) reconcileHelmChartTest(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (*reconcileResult, error) {
	// Nothing to do when the status of the plugin is empty and when the plugin does not have a Helm Chart
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) || plugin.Status.HelmChart == nil {
		return nil, nil
	}

	// Helm Chart Test cannot be done as the Helm Chart deployment is not successful
	if plugin.Status.HelmReleaseStatus.Status != "deployed" {
		return nil, nil
	}

	restClientGetter, err := InitClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin)
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

	hasHelmChartTest, testPodLogs, err := helm.ChartTest(ctx, restClientGetter, plugin)
	if err != nil {
		failedTestPodLogs := extractErrorsFromTestPodLogs(testPodLogs)
		if failedTestPodLogs != "" {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "", failedTestPodLogs))
		} else {
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "", err.Error()))
		}
		IncrementHelmChartTestRunsTotal(plugin, "Error")
		return nil, err
	}

	if !hasHelmChartTest {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "",
			"No Helm Chart Tests defined by the PluginDefinition"))

		IncrementHelmChartTestRunsTotal(plugin, "NoTests")
	} else {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmChartTestSucceededCondition, "",
			"Helm Chart Test is successful"))

		IncrementHelmChartTestRunsTotal(plugin, "Success")
	}

	return nil, nil
}

func extractErrorsFromTestPodLogs(testPodLogs string) string {
	var errors []string
	var errorBlock strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(testPodLogs))

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "not ok"):
			if errorBlock.Len() > 0 {
				errors = append(errors, errorBlock.String())
				errorBlock.Reset()
			}
			errorBlock.WriteString(line + "\n")
		case strings.HasPrefix(line, "ok"):
			// Ignore lines starting with "ok"
			continue
		default:
			if errorBlock.Len() > 0 {
				// Append additional lines related to the failure
				if strings.HasPrefix(line, "#") {
					errorBlock.WriteString(line + "\n")
				}
			}
		}
	}

	if errorBlock.Len() > 0 {
		errors = append(errors, errorBlock.String())
	}

	// Join the error messages with an empty line separator
	return strings.Join(errors, "\n")
}
