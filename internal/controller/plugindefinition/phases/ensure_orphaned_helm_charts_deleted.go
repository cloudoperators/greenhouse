// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"fmt"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// ensureOrphanedHelmChartsDeleted lists HelmChart resources labelled with the owner PluginDefinition
// name and deletes every one whose name differs from the currently desired FluxHelmChartResourceName.
// Using a label selector avoids scanning all HelmCharts in the namespace on every reconcile.
// The controller-reference UID check provides a secondary guard to prevent touching charts that
// were labelled identically by an unrelated controller.
func (p *Phase) ensureOrphanedHelmChartsDeleted() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		currentName := p.PluginDef.FluxHelmChartResourceName()
		logger := log.FromContext(ctx)

		helmChartList := &sourcev1.HelmChartList{}
		if err := p.Client.List(ctx, helmChartList,
			client.InNamespace(p.NamespaceName),
			client.MatchingLabels{greenhouseapis.LabelKeyPluginDefinition: p.PluginDef.GetName()},
		); err != nil {
			return lifecycle.Break(), fmt.Errorf("failed to list HelmCharts in namespace %s: %w", p.NamespaceName, err)
		}

		for i := range helmChartList.Items {
			chart := &helmChartList.Items[i]
			// Use metav1.GetControllerOf to reliably retrieve the controller owner reference
			// (avoids relying on GetObjectKind().GroupVersionKind().Kind which can be empty).
			controllerRef := metav1.GetControllerOf(chart)
			if controllerRef == nil || controllerRef.UID != p.PluginDef.GetUID() {
				continue
			}
			// Keep the current (desired) HelmChart; delete everything else.
			if chart.Name == currentName {
				continue
			}
			logger.Info("deleting orphaned HelmChart", "namespace", chart.Namespace, "name", chart.Name)
			if err := p.Client.Delete(ctx, chart); err != nil {
				// Ignore NotFound — the chart may have been deleted concurrently.
				if apierrors.IsNotFound(err) {
					continue
				}
				return lifecycle.Break(), fmt.Errorf("failed to delete orphaned HelmChart %s/%s: %w", chart.Namespace, chart.Name, err)
			}
			p.Recorder.Eventf(p.PluginDef, chart, corev1.EventTypeNormal, "Deleted", "reconciling (Cluster-)PluginDefinition", "Deleted orphaned HelmChart %s", chart.Name)
		}
		return lifecycle.Continue(), nil
	}
}
