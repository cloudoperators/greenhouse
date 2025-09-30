// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/util"
)

const (
	maxHistory = 10
	secretKind = "Secret"
)

func (r *PluginReconciler) EnsureFluxDeleted(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (ctrl.Result, lifecycle.ReconcileResult, error) {
	if err := r.Delete(ctx, &helmv2.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: plugin.Name, Namespace: plugin.Namespace}}); err != nil {
		c := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.HelmUninstallFailedReason, err.Error())
		plugin.SetCondition(c)
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonClusterAccessFailed)
		return ctrl.Result{}, lifecycle.Failed, err
	}

	plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.HelmReconcileFailedCondition, "", ""))
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *PluginReconciler) EnsureFluxCreated(ctx context.Context, plugin *greenhousev1alpha1.Plugin) (ctrl.Result, lifecycle.ReconcileResult, error) {
	pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(ctx, r.Client, plugin.Spec.PluginDefinitionRef, plugin.GetNamespace())
	if err != nil {
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, greenhousev1alpha1.PluginDefinitionNotFoundReason, err.Error()))
		util.UpdatePluginReconcileTotalMetric(plugin, util.MetricResultError, util.MetricReasonPluginDefinitionNotFound)
		return ctrl.Result{}, lifecycle.Failed, fmt.Errorf("%s not found: %s", plugin.Spec.PluginDefinitionRef.Kind, err.Error())
	}

	namespace := flux.HelmRepositoryDefaultNamespace
	if plugin.Spec.PluginDefinitionRef.Kind == greenhousev1alpha1.PluginDefinitionKind {
		namespace = plugin.GetNamespace()
	}

	if pluginDefinitionSpec.HelmChart == nil {
		log.FromContext(ctx).Info("No HelmChart defined in PluginDefinition, skipping HelmRelease creation", "plugin", plugin.Name)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.HelmReconcileFailedCondition, "", "PluginDefinition is not backed by HelmChart"))
		return ctrl.Result{}, lifecycle.Success, nil
	}

	helmRepository, err := flux.FindHelmRepositoryByURL(ctx, r.Client, pluginDefinitionSpec.HelmChart.Repository, namespace)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, errors.New("helm repository not found")
	}

	release := &helmv2.HelmRelease{}
	release.SetName(plugin.Name)
	release.SetNamespace(plugin.Namespace)

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, release, func() error {
		values, err := addValuesToHelmRelease(ctx, r.Client, plugin)
		if err != nil {
			return fmt.Errorf("failed to compute HelmRelease values for Plugin %s: %w", plugin.Name, err)
		}

		spec, err := flux.NewHelmReleaseSpecBuilder().
			WithChart(helmv2.HelmChartTemplateSpec{
				Chart:    pluginDefinitionSpec.HelmChart.Name,
				Interval: &metav1.Duration{Duration: flux.DefaultInterval},
				Version:  pluginDefinitionSpec.HelmChart.Version,
				SourceRef: helmv2.CrossNamespaceObjectReference{
					Kind:      sourcecontroller.HelmRepositoryKind,
					Name:      helmRepository.Name,
					Namespace: helmRepository.Namespace,
				},
			}).
			WithInterval(flux.DefaultInterval).
			WithTimeout(flux.DefaultTimeout).
			WithMaxHistory(maxHistory).
			WithReleaseName(plugin.GetReleaseName()).
			WithInstall(&helmv2.Install{
				CreateNamespace: true,
				Remediation: &helmv2.InstallRemediation{
					Retries: 3,
				},
			}).
			WithUpgrade(&helmv2.Upgrade{
				Remediation: &helmv2.UpgradeRemediation{
					Retries: 3,
				},
			}).
			WithTest(&helmv2.Test{
				Enable: false,
			}).
			WithDriftDetection(&helmv2.DriftDetection{
				Mode: helmv2.DriftDetectionEnabled,
			}).
			WithSuspend(release.Spec.Suspend).
			WithKubeConfig(fluxmeta.SecretKeyReference{
				Name: plugin.Spec.ClusterName,
				Key:  greenhouseapis.GreenHouseKubeConfigKey,
			}).
			WithValues(values).
			WithValuesFrom(r.addValueReferences(plugin)).
			WithTargetNamespace(plugin.Spec.ReleaseNamespace).Build()
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to create HelmRelease for plugin", "plugin", plugin.Name)
			return fmt.Errorf("failed to create HelmRelease for plugin %s: %w", plugin.Name, err)
		}
		release.Spec = spec
		return nil
	})
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}
	switch result {
	case controllerutil.OperationResultCreated:
		log.FromContext(ctx).Info("Created helmRelease", "name", release.Name)
	case controllerutil.OperationResultUpdated:
		log.FromContext(ctx).Info("Updated helmRelease", "name", release.Name)
	}

	return ctrl.Result{}, lifecycle.Success, nil
}

func addValuesToHelmRelease(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]byte, error) {
	optionValues, err := helm.GetPluginOptionValuesForPlugin(ctx, c, plugin)
	if err != nil {
		return nil, err
	}

	optionValues, err = helm.ResolveTemplatedValues(ctx, optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve templated values: %w", err)
	}

	// remove all option values that are set from a secret, as these have a nil value
	optionValues = slices.DeleteFunc(optionValues, func(v greenhousev1alpha1.PluginOptionValue) bool {
		return v.ValueFrom != nil && v.ValueFrom.Secret != nil
	})

	jsonValue, err := helm.ConvertFlatValuesToHelmValues(optionValues)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plugin option values to JSON: %w", err)
	}

	byteValue, err := json.Marshal(jsonValue)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "Unable to marshal values for plugin", "plugin", plugin.Name)
		return nil, err
	}
	return byteValue, nil
}

func (r *PluginReconciler) addValueReferences(plugin *greenhousev1alpha1.Plugin) []helmv2.ValuesReference {
	var valuesFrom []helmv2.ValuesReference
	for _, value := range plugin.Spec.OptionValues {
		if value.ValueFrom != nil {
			valuesFrom = append(valuesFrom, helmv2.ValuesReference{
				Kind:       secretKind,
				Name:       value.ValueFrom.Secret.Name,
				ValuesKey:  value.ValueFrom.Secret.Key,
				TargetPath: value.Name,
			})
		}
	}
	return valuesFrom
}
