// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"
	"strings"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/ocimirror"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type helmer struct {
	k8sClient     client.Client
	recorder      events.EventRecorder
	pluginDef     common.GenericPluginDefinition
	namespaceName string
}

// initializeConditions sets the provided conditions to Unknown if they do not already exist.
func initializeConditions(resource common.GenericPluginDefinition, conditionTypes ...greenhousemetav1alpha1.ConditionType) {
	conditions := resource.GetConditions()
	for _, condType := range conditionTypes {
		if cond := conditions.GetConditionByType(condType); cond == nil {
			resource.SetCondition(
				greenhousemetav1alpha1.UnknownCondition(condType, greenhousev1alpha1.PluginDefinitionProgressingReason, "reconciliation in progress"),
			)
		}
	}
}

func setReadyCondition(resource common.GenericPluginDefinition) {
	if resource.GetPluginDefinitionSpec().HelmChart == nil {
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "No HelmChart defined"))
		return
	}

	conditions := resource.GetConditions()
	helmChartCondition := conditions.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
	switch {
	case helmChartCondition == nil:
		resource.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "HelmChart status unknown"))
		return
	case !helmChartCondition.IsTrue():
		resource.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousemetav1alpha1.ReadyCondition,
			helmChartCondition.Reason,
			helmChartCondition.Message))
		return
	}

	imageReplicationCondition := conditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
	switch {
	case imageReplicationCondition == nil:
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "PluginDefinition is ready"))
	case imageReplicationCondition.IsFalse():
		resource.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousemetav1alpha1.ReadyCondition,
			imageReplicationCondition.Reason,
			imageReplicationCondition.Message))
	default:
		resource.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousemetav1alpha1.ReadyCondition, "", "PluginDefinition is ready"))
	}
}

// setHelmChartReadyCondition checks the HelmChart status and sets the HelmChartReady condition on the given object.
func (h *helmer) setHelmChartReadyCondition(ctx context.Context, fluxObj lifecycle.CatalogObject) {
	if err := h.k8sClient.Get(ctx, client.ObjectKeyFromObject(fluxObj), fluxObj); err != nil {
		h.pluginDef.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmChartReadyCondition, "", "unable to fetch HelmRepository status"))
		return
	}
	readyCondition := meta.FindStatusCondition(fluxObj.GetConditions(), fluxmeta.ReadyCondition)
	switch {
	case readyCondition == nil:
		h.pluginDef.SetCondition(greenhousemetav1alpha1.UnknownCondition(
			greenhousev1alpha1.HelmChartReadyCondition, "", "HelmChart status pending"))
	case readyCondition.Status == metav1.ConditionTrue:
		h.pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmChartReadyCondition, greenhousemetav1alpha1.ConditionReason(readyCondition.Reason), readyCondition.Message))
	default:
		h.pluginDef.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmChartReadyCondition,
			greenhousemetav1alpha1.ConditionReason(readyCondition.Reason),
			readyCondition.Message))
	}
}

func (h *helmer) createUpdateHelmRepository(ctx context.Context) (*sourcev1.HelmRepository, error) {
	pluginDefSpec := h.pluginDef.GetPluginDefinitionSpec()
	repositoryURL := pluginDefSpec.HelmChart.Repository
	helmRepository := &sourcev1.HelmRepository{}
	helmRepository.SetName(flux.ChartURLToName(pluginDefSpec.HelmChart.Repository))
	helmRepository.SetNamespace(h.namespaceName)

	result, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, helmRepository, func() error {
		helmRepository.Spec.Type = flux.GetSourceRepositoryType(repositoryURL)
		helmRepository.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}
		helmRepository.Spec.URL = repositoryURL
		return controllerutil.SetOwnerReference(h.pluginDef, helmRepository, h.k8sClient.Scheme())
	})
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to create or update HelmRepository", "namespace", h.namespaceName, "name", helmRepository.Name)
		return nil, err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRepository", "namespace", h.namespaceName, "name", helmRepository.Name)
		h.recorder.Eventf(h.pluginDef, helmRepository, corev1.EventTypeNormal, "Created", "reconciling (Cluster-)PluginDefinition", "Created HelmRepository %s", helmRepository.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRepository", "namespace", h.namespaceName, "name", helmRepository.Name)
		h.recorder.Eventf(h.pluginDef, helmRepository, corev1.EventTypeNormal, "Updated", "reconciling (Cluster-)PluginDefinition", "Updated HelmRepository %s", helmRepository.Name)
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmRepository", "namespace", h.namespaceName, "name", helmRepository.Name)
	}
	return helmRepository, nil
}

func (h *helmer) createUpdateHelmChart(ctx context.Context, helmRepo *sourcev1.HelmRepository) (*sourcev1.HelmChart, error) {
	pluginDefSpec := h.pluginDef.GetPluginDefinitionSpec()
	helmChart := &sourcev1.HelmChart{}
	helmChart.SetName(h.pluginDef.FluxHelmChartResourceName())
	helmChart.SetNamespace(h.namespaceName)
	result, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, helmChart, func() error {
		helmChart.Spec = sourcev1.HelmChartSpec{
			Chart:             pluginDefSpec.HelmChart.Name,
			Interval:          metav1.Duration{Duration: 5 * time.Minute},
			ReconcileStrategy: sourcev1.ReconcileStrategyChartVersion,
			SourceRef: sourcev1.LocalHelmChartSourceReference{
				Kind: sourcev1.HelmRepositoryKind,
				Name: helmRepo.Name,
			},
			Version: pluginDefSpec.HelmChart.Version,
		}
		return controllerutil.SetControllerReference(h.pluginDef, helmChart, h.k8sClient.Scheme())
	})
	if err != nil {
		return nil, err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmChart", "namespace", h.namespaceName, "name", helmChart.Name)
		h.recorder.Eventf(h.pluginDef, helmChart, corev1.EventTypeNormal, "Created", "reconciling (Cluster-)PluginDefinition", "Created HelmChart %s", helmChart.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmChart", "namespace", h.namespaceName, "name", helmChart.Name)
		h.recorder.Eventf(h.pluginDef, helmChart, corev1.EventTypeNormal, "Updated", "reconciling (Cluster-)PluginDefinition", "Updated HelmChart %s", helmChart.Name)
	case controllerutil.OperationResultNone:
		log.FromContext(ctx).Info("No changes to helmChart", "namespace", h.namespaceName, "name", helmChart.Name)
	}
	return helmChart, nil
}

// ensureChartReplication triggers replication for the Helm chart OCI artifact to the configured mirror registry.
func ensureChartReplication(ctx context.Context, k8sClient client.Client, pluginDef common.GenericPluginDefinition, namespaceName string) error {
	logger := log.FromContext(ctx)

	failReplication := func(err error, msg string) error {
		logger.Error(err, msg)
		pluginDef.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.OCIReplicationReadyCondition,
			greenhousev1alpha1.OCIReplicationFailedReason,
			msg+": "+err.Error()))
		return err
	}

	mirrorConfig, err := ocimirror.GetRegistryMirrorConfig(ctx, k8sClient, namespaceName)
	if err != nil {
		return failReplication(err, "failed to get registry mirror config")
	}

	if mirrorConfig == nil || mirrorConfig.PrimaryMirror == "" {
		pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.OCIReplicationReadyCondition,
			greenhousev1alpha1.OCIReplicationNotConfiguredReason,
			"OCI replication is not configured"))
		return nil
	}

	helmChart := pluginDef.GetPluginDefinitionSpec().HelmChart
	if !strings.HasPrefix(helmChart.Repository, "oci://") {
		logger.V(1).Info("chart repository is not OCI, skipping replication", "repository", helmChart.Repository)
		pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.OCIReplicationReadyCondition,
			greenhousev1alpha1.OCIReplicationNotConfiguredReason,
			"Chart replication only applies to OCI repositories"))
		return nil
	}
	chartRef := strings.TrimPrefix(helmChart.Repository, "oci://") + "/" + helmChart.Name + ":" + helmChart.Version

	// Skip replication if the current chart version was already replicated.
	conditions := pluginDef.GetConditions()
	replicationCond := conditions.GetConditionByType(greenhousev1alpha1.OCIReplicationReadyCondition)
	if replicationCond != nil && replicationCond.IsTrue() &&
		replicationCond.Reason == greenhousev1alpha1.OCIReplicationSucceededReason &&
		strings.Contains(replicationCond.Message, chartRef) {
		logger.V(1).Info("chart already replicated, skipping", "chart", chartRef)
		return nil
	}

	replicator, err := ocimirror.NewOCIReplicator(ctx, k8sClient, mirrorConfig, namespaceName)
	if err != nil {
		return failReplication(err, "failed to create OCI replicator")
	}

	mirroredRef := replicator.BuildMirroredOCIRef(chartRef)
	if mirroredRef == "" {
		logger.Info("no mirror configured for chart registry, skipping replication", "chart", chartRef)
		pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.OCIReplicationReadyCondition,
			greenhousev1alpha1.OCIReplicationNotConfiguredReason,
			"No mirror configured for chart registry"))
		return nil
	}

	if err := replicator.TriggerReplication(ctx, mirroredRef); err != nil {
		return failReplication(err, "chart replication failed")
	}

	pluginDef.SetCondition(greenhousemetav1alpha1.TrueCondition(
		greenhousev1alpha1.OCIReplicationReadyCondition,
		greenhousev1alpha1.OCIReplicationSucceededReason,
		"Chart replicated successfully: "+chartRef))
	return nil
}
