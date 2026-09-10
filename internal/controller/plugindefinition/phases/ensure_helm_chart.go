// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureHelmChart() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		helmChart, err := p.CreateUpdateHelmChart(ctx, p.helmRepo)
		if err != nil {
			return lifecycle.Break(), err
		}
		p.setHelmChartReadyCondition(ctx, helmChart)
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) CreateUpdateHelmChart(ctx context.Context, helmRepo *sourcev1.HelmRepository) (*sourcev1.HelmChart, error) {
	pluginDefSpec := p.PluginDef.GetPluginDefinitionSpec()
	helmChart := &sourcev1.HelmChart{}
	helmChart.SetName(p.PluginDef.FluxHelmChartResourceName())
	helmChart.SetNamespace(p.NamespaceName)
	ownedBy, owned := p.PluginDef.GetLabels()[greenhouseapis.LabelKeyOwnedBy]

	result, err := controllerutil.CreateOrUpdate(ctx, p.Client, helmChart, func() error {
		labels := helmChart.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		if owned {
			labels[greenhouseapis.LabelKeyOwnedBy] = ownedBy
		}
		labels[greenhouseapis.LabelKeyPluginDefinition] = p.PluginDef.GetName()
		helmChart.SetLabels(labels)
		helmChart.Spec = sourcev1.HelmChartSpec{
			Chart:             pluginDefSpec.HelmChart.Name,
			Interval:          metav1.Duration{Duration: 24 * time.Hour},
			ReconcileStrategy: sourcev1.ReconcileStrategyChartVersion,
			SourceRef: sourcev1.LocalHelmChartSourceReference{
				Kind: sourcev1.HelmRepositoryKind,
				Name: helmRepo.Name,
			},
			Version: pluginDefSpec.HelmChart.Version,
		}
		// Trigger an immediate Flux reconcile instead of waiting for the sync interval.
		requestedAt, _ := lifecycle.ReconcileAnnotationValue(p.PluginDef)
		common.EnsureAnnotation(helmChart, fluxmeta.ReconcileRequestAnnotation, requestedAt)
		return controllerutil.SetControllerReference(p.PluginDef, helmChart, p.Client.Scheme())
	})
	if err != nil {
		return nil, err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmChart", "namespace", p.NamespaceName, "name", helmChart.Name)
		if p.Recorder != nil {
			p.Recorder.Eventf(p.PluginDef, helmChart, corev1.EventTypeNormal, "Created", "reconciling (Cluster-)PluginDefinition", "Created HelmChart %s", helmChart.Name)
		}
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmChart", "namespace", p.NamespaceName, "name", helmChart.Name)
		if p.Recorder != nil {
			p.Recorder.Eventf(p.PluginDef, helmChart, corev1.EventTypeNormal, "Updated", "reconciling (Cluster-)PluginDefinition", "Updated HelmChart %s", helmChart.Name)
		}
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmChart", "namespace", p.NamespaceName, "name", helmChart.Name)
	}
	return helmChart, nil
}

// setHelmChartReadyCondition checks the HelmChart status and sets the HelmChartReady condition on the given object.
func (p *Phase) setHelmChartReadyCondition(ctx context.Context, fluxObj lifecycle.CatalogObject) {
	if err := p.Client.Get(ctx, client.ObjectKeyFromObject(fluxObj), fluxObj); err != nil {
		p.PluginDef.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmChartReadyCondition, "", "unable to fetch HelmRepository status"))
		return
	}
	readyCondition := meta.FindStatusCondition(fluxObj.GetConditions(), fluxmeta.ReadyCondition)
	switch {
	case readyCondition == nil:
		p.PluginDef.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmChartReadyCondition, "", "HelmChart status pending"))
	case readyCondition.Status == metav1.ConditionTrue:
		p.PluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmChartReadyCondition, greenhousemetav1alpha1.ConditionReason(readyCondition.Reason), readyCondition.Message))
	default:
		p.PluginDef.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmChartReadyCondition,
			greenhousemetav1alpha1.ConditionReason(readyCondition.Reason),
			readyCondition.Message))
	}
}
