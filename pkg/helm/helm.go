// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

func init() {
	// Setting the name of the app for managedFields in the Kubernetes client
	kube.ManagedFieldsManager = greenhouseFieldManager
}

var (
	settings = cli.New()

	// IsHelmDebug is configured via a flag and enables extensive debug logging for Helm actions.
	IsHelmDebug bool
)

// driftDetectionInterval is the interval after which a drift detection is performed.
const driftDetectionInterval = 60 * time.Minute

// InstallOrUpgradeHelmChartFromPlugin installs a new or upgrades an existing Helm release for the given Plugin and PluginConfig.
func InstallOrUpgradeHelmChartFromPlugin(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) error {
	// Early return if the plugin is not helm based
	if plugin.Spec.HelmChart == nil {
		return fmt.Errorf("no helm chart defined in plugin.Spec.HelmChart for plugin %s", pluginConfig.Spec.Plugin)
	}
	latestRelease, isReleaseExists, err := isReleaseExistsForPluginConfig(ctx, restClientGetter, pluginConfig)
	if err != nil {
		return err
	}
	// A release does not exist. Install it.
	if !isReleaseExists {
		log.FromContext(ctx).Info("installing release for plugin config", "namespace", pluginConfig.Namespace, "name", pluginConfig.Name)
		_, err = installRelease(ctx, local, restClientGetter, plugin, pluginConfig, false)
		return err
	}
	helmChart, err := locateChartForPlugin(restClientGetter, plugin)
	if err != nil {
		return err
	}
	// Avoid attempts to upgrade a failed release and attempt to resurrect it.
	if latestRelease.Info != nil && latestRelease.Info.Status == release.StatusFailed {
		log.FromContext(ctx).Info("attempting to reset release status", "current status", latestRelease.Info.Status.String())
		if err := ResetHelmReleaseStatusToDeployed(ctx, restClientGetter, pluginConfig); err != nil {
			return err
		}
	}
	// Avoid upgrading a currently pending release.
	if releaseStatus, ok := isCanReleaseBeUpgraded(latestRelease); !ok {
		return fmt.Errorf("cannot upgrade release %s/%s in status %s", latestRelease.Namespace, latestRelease.Name, releaseStatus.String())
	}
	log.FromContext(ctx).Info("upgrading release", "namespace", pluginConfig.Namespace, "name", pluginConfig.Name)

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return err
	}

	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects()); err != nil {
		return err
	}
	return upgradeRelease(ctx, local, restClientGetter, plugin, pluginConfig)
}

// UninstallHelmRelease removes the Helm release for the given PluginConfig.
func UninstallHelmRelease(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, pluginConfig *greenhousev1alpha1.PluginConfig) (releaseNotFound bool, err error) {
	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return false, err
	}
	_, isReleaseExists, err := isReleaseExistsForPluginConfig(ctx, restClientGetter, pluginConfig)
	if err != nil {
		return false, err
	}
	settings.RESTClientGetter()
	if !isReleaseExists {
		return true, nil
	}
	uninstallAction := action.NewUninstall(cfg)
	uninstallAction.KeepHistory = false
	_, err = uninstallAction.Run(pluginConfig.Name)
	return false, err
}

// DiffChartToDeployedResources returns whether the Kubernetes objects, as specified in the Helm chart manifest, differ from the deployed state.
func DiffChartToDeployedResources(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) (diffs DiffObjectList, isDrift bool, err error) {
	// Shortcut: If the Helm chart was changed we can skip below templating and diffing.
	var pluginConfigStatusHelmChart string
	if pluginConfig.Status.HelmReleaseStatus != nil && pluginConfig.Status.HelmChart != nil {
		pluginConfigStatusHelmChart = pluginConfig.Status.HelmChart.String()
	}
	if plugin.Spec.HelmChart.String() != pluginConfigStatusHelmChart {
		log.FromContext(ctx).Info("observed helm chart differs from plugin helm chart", "plugin", plugin.Spec.HelmChart.String(), "pluginConfig", pluginConfigStatusHelmChart)
		return nil, true, nil
	}

	helmRelease, exists, err := isReleaseExistsForPluginConfig(ctx, restClientGetter, pluginConfig)
	switch {
	case err != nil:
		return nil, false, err
		// check if the release has the current plugin version set as description
		// this description is used to reconcile the version of the PluginConfig
	case exists && helmRelease.Info.Description != plugin.Spec.Version:
		log.FromContext(ctx).Info("deployed helm chart version differs from plugin helm chart", "plugin", helmRelease.Info.Description, "pluginConfig", plugin.Spec.Version)
		return nil, true, nil
	}

	manifest, err := TemplateHelmChartFromPlugin(ctx, local, restClientGetter, plugin, pluginConfig)
	if err != nil {
		return nil, false, err
	}

	diffObjects, err := diffAgainstRelease(restClientGetter, pluginConfig.GetNamespace(), manifest, helmRelease)
	if err != nil {
		return nil, false, err
	}

	if len(diffObjects) > 0 {
		log.FromContext(ctx).Info("diff between manifest and release detected", "resources", diffObjects.String())
		return diffObjects, false, nil
	}

	c := pluginConfig.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmDriftDetectedCondition)
	// Skip the drift detection if last DriftDetection Status Change or last Deployment was less than driftDetectionInterval ago
	if c != nil && c.Status != metav1.ConditionUnknown &&
		time.Since(c.LastTransitionTime.Time) < driftDetectionInterval &&
		time.Since(pluginConfig.Status.HelmReleaseStatus.LastDeployed.Time) < driftDetectionInterval {
		return nil, false, nil
	}

	diffObjects, err = diffAgainstLiveObjects(restClientGetter, pluginConfig.GetNamespace(), manifest)
	if err != nil {
		return nil, false, err
	}
	if len(diffObjects) == 0 {
		return nil, false, nil
	}
	log.FromContext(ctx).Info("drift between deployed resources and manifest detected", "resources", diffObjects.String())
	return diffObjects, true, nil
}

// ResetHelmReleaseStatusToDeployed resets the status of the release to deployed using a rollback.
func ResetHelmReleaseStatusToDeployed(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, pluginConfig *greenhousev1alpha1.PluginConfig) error {
	r, err := getLatestUpgradeableRelease(restClientGetter, pluginConfig)
	if err != nil {
		return err
	}

	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return err
	}
	rollbackAction := action.NewRollback(cfg)
	rollbackAction.Version = r.Version
	rollbackAction.DisableHooks = true
	rollbackAction.Wait = true
	rollbackAction.Timeout = 5 * time.Minute
	rollbackAction.MaxHistory = 5
	return rollbackAction.Run(r.Name)
}

// getLatestUpgradeableRelease returns the latest released that can be upgraded or an error.
func getLatestUpgradeableRelease(restClientGetter genericclioptions.RESTClientGetter, pluginConfig *greenhousev1alpha1.PluginConfig) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return nil, err
	}
	var latest *release.Release
	releases, err := action.NewHistory(cfg).Run(pluginConfig.Name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving releases: %w", err)
	}
	for _, r := range releases {
		if _, canUpgrade := isCanReleaseBeUpgraded(r); canUpgrade {
			if latest == nil {
				latest = r
				continue
			}
			if r.Version > latest.Version {
				latest = r
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no release found to rollback to for plugin config %s/%s", pluginConfig.Namespace, pluginConfig.Name)
	}
	return latest, nil
}

// isReleaseExistsForPluginConfig checks whether a Helm release exists for the given PluginConfig.
func isReleaseExistsForPluginConfig(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, pluginConfig *greenhousev1alpha1.PluginConfig) (*release.Release, bool, error) {
	helmRelease, err := GetReleaseForHelmChartFromPluginConfig(ctx, restClientGetter, pluginConfig)
	if err != nil {
		switch errors.Is(err, driver.ErrReleaseNotFound) {
		case true:
			return nil, false, nil
		default:
			return nil, false, err
		}
	}
	return helmRelease, true, nil
}

// GetReleaseForHelmChartFromPluginConfig returns the Helm release for the given PluginConfig or an error.
func GetReleaseForHelmChartFromPluginConfig(_ context.Context, restClientGetter genericclioptions.RESTClientGetter, pluginConfig *greenhousev1alpha1.PluginConfig) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return nil, err
	}
	return action.NewGet(cfg).Run(pluginConfig.Name)
}

// TemplateHelmChartFromPlugin returns the rendered manifest or an error.
func TemplateHelmChartFromPlugin(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) (string, error) {
	helmRelease, err := installRelease(ctx, local, restClientGetter, plugin, pluginConfig, true)
	if err != nil {
		return "", err
	}
	return helmRelease.Manifest, nil
}

type ChartLoaderFunc func(name string) (*chart.Chart, error)

var ChartLoader ChartLoaderFunc = loader.Load

func locateChartForPlugin(restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (*chart.Chart, error) {
	cfg, err := newHelmAction(restClientGetter, corev1.NamespaceAll)
	if err != nil {
		return nil, err
	}

	// FIXME: we need to instantiate a action to set the registry in the ChartPathOptions
	cpo := &action.NewShowWithConfig(action.ShowChart, cfg).ChartPathOptions

	chartName := configureChartPathOptions(cpo, plugin.Spec.HelmChart)
	chartPath, err := cpo.LocateChart(chartName, settings)
	if err != nil {
		return nil, err
	}
	return ChartLoader(chartPath)
}

// configureChartPathOptions configures the ChartPathOptions and chartName considering OCI repositories.
func configureChartPathOptions(cpo *action.ChartPathOptions, c *greenhousev1alpha1.HelmChartReference) string {
	cpo.RepoURL = c.Repository
	cpo.Version = c.Version
	chartName := c.Name
	// Handle OCI.
	if registry.IsOCI(c.Repository) {
		cpo.RepoURL = ""
		chartName = fmt.Sprintf("%s/%s", c.Repository, c.Name)
	}
	return chartName
}

func upgradeRelease(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) error {
	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return err
	}
	upgradeAction := action.NewUpgrade(cfg)
	upgradeAction.Namespace = pluginConfig.Namespace
	upgradeAction.DependencyUpdate = true
	upgradeAction.MaxHistory = 5
	upgradeAction.Description = plugin.Spec.Version
	chartName := configureChartPathOptions(&upgradeAction.ChartPathOptions, plugin.Spec.HelmChart)

	chartPath, err := upgradeAction.LocateChart(chartName, settings)
	if err != nil {
		return err
	}
	helmChart, err := ChartLoader(chartPath)
	if err != nil {
		return err
	}

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return err
	}

	helmValues, err := getValuesForHelmChart(ctx, local, helmChart, pluginConfig, false)
	if err != nil {
		return err
	}
	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects()); err != nil {
		return err
	}

	// Do the Kubernetes version check beforehand to reflect incompatibilities in the PluginConfig status before attempting an installation or upgrade.
	if err := verifyKubeVersionIsCompatible(helmChart, cfg.Capabilities); err != nil {
		return err
	}
	helmChart.Metadata.KubeVersion = ""
	_, err = upgradeAction.RunWithContext(ctx, pluginConfig.Name, helmChart, helmValues)
	return err
}

func installRelease(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig, isDryRun bool) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, pluginConfig.Namespace)
	if err != nil {
		return nil, err
	}
	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = pluginConfig.Name
	installAction.Namespace = pluginConfig.Namespace
	// Namespaces are only created by an Organization.
	installAction.CreateNamespace = false
	installAction.DependencyUpdate = true
	installAction.DryRun = isDryRun
	installAction.ClientOnly = isDryRun
	installAction.Description = plugin.Spec.Version
	chartName := configureChartPathOptions(&installAction.ChartPathOptions, plugin.Spec.HelmChart)

	chartPath, err := installAction.LocateChart(chartName, settings)
	if err != nil {
		return nil, err
	}
	helmChart, err := ChartLoader(chartPath)
	if err != nil {
		return nil, err
	}

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return nil, err
	}

	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects()); err != nil {
		return nil, err
	}
	helmValues, err := getValuesForHelmChart(ctx, local, helmChart, pluginConfig, isDryRun)
	if err != nil {
		return nil, err
	}

	// Do the Kubernetes version check beforehand to reflect incompatibilities in the PluginConfig status before attempting an installation or upgrade.
	if err := verifyKubeVersionIsCompatible(helmChart, cfg.Capabilities); err != nil {
		return nil, err
	}
	helmChart.Metadata.KubeVersion = ""
	return installAction.RunWithContext(ctx, helmChart, helmValues)
}

func newHelmAction(restClientGetter genericclioptions.RESTClientGetter, namespace string) (*action.Configuration, error) {
	cfg := &action.Configuration{}
	settings.SetNamespace(namespace)
	if err := cfg.Init(restClientGetter, namespace, "secrets", debug); err != nil {
		return nil, err
	}

	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(IsHelmDebug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	)
	if err != nil {
		return nil, err
	}
	cfg.RegistryClient = registryClient
	caps, err := getCapabilities(cfg)
	if err != nil {
		return nil, err
	}
	cfg.Capabilities = caps
	return cfg, nil
}

func debug(format string, v ...interface{}) {
	if IsHelmDebug {
		format = fmt.Sprintf("[debug] %s", format)
		log.FromContext(context.Background()).Info(fmt.Sprintf(format, v...))
	}
}

/*
convertFlatValuesToHelmValues shall converts flat values for a Helm chart yaml-compatible structure.
Example:
The input

	global.image.registry=foobar

is transformed to

	global:
	  image:
	    registry: foobar
*/
func convertFlatValuesToHelmValues(values []greenhousev1alpha1.PluginOptionValue) (map[string]interface{}, error) {
	if values == nil {
		return make(map[string]interface{}, 0), nil
	}
	helmValues := make(map[string]interface{}, 0)
	for _, v := range values {
		jsonVal, err := v.ValueJSON()
		if err != nil {
			return nil, err
		}
		if err := strvals.ParseJSON(fmt.Sprintf("%s=%s", v.Name, jsonVal), helmValues); err != nil {
			return nil, err
		}
	}
	return helmValues, nil
}

// Taken from: https://github.com/helm/helm/blob/v3.10.3/pkg/cli/values/options.go#L99-L116
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// getValuesForHelmChart returns a set of values to be used for Helm operations.
// The order is important as the values defined in the Helm chart can be overridden by the values defined in the PluginConfig.
func getValuesForHelmChart(ctx context.Context, c client.Client, helmChart *chart.Chart, pluginConfig *greenhousev1alpha1.PluginConfig, isDryRun bool) (map[string]interface{}, error) {
	// Copy the values from the Helm chart ensuring a non-nil map.
	helmValues := mergeMaps(make(map[string]interface{}, 0), helmChart.Values)
	// Get values defined in pluginconfig.
	pluginConfigValues, err := getValuesFromPluginConfig(ctx, c, pluginConfig)
	if err != nil {
		return nil, err
	}
	helmPluginConfigValues, err := convertFlatValuesToHelmValues(pluginConfigValues)
	if err != nil {
		return nil, err
	}
	helmValues = mergeMaps(helmValues, helmPluginConfigValues)
	return helmValues, nil
}

func getValuesFromPluginConfig(ctx context.Context, c client.Client, pluginConfig *greenhousev1alpha1.PluginConfig) ([]greenhousev1alpha1.PluginOptionValue, error) {
	namedValues := make([]greenhousev1alpha1.PluginOptionValue, len(pluginConfig.Spec.OptionValues))
	copy(namedValues, pluginConfig.Spec.OptionValues)
	for idx, val := range namedValues {
		// Values already provided on plain text don't need to be extracted.
		if val.ValueFrom == nil {
			continue
		}
		switch {
		// Retrieve value from secret.
		case val.ValueFrom.Secret != nil:
			valFromSecret, err := getValueFromSecret(ctx, c, pluginConfig.Namespace, val.ValueFrom.Secret.Name, val.ValueFrom.Secret.Key)
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(valFromSecret)
			if err != nil {
				return nil, err
			}
			namedValues[idx].Value = &apiextensionsv1.JSON{Raw: raw}
		}
	}
	return namedValues, nil
}

func getValueFromSecret(ctx context.Context, c client.Client, secretNamespace, secretName, secretKey string) (string, error) {
	var secret = new(corev1.Secret)
	if err := c.Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, secret); err != nil {
		return "", err
	}
	if secret.Data == nil {
		return "", fmt.Errorf("secret %s/%s is empty", secretNamespace, secretName)
	}
	valByte, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("secret %s/%s does not contain key %s", secretNamespace, secretName, secretKey)
	}
	return string(valByte), nil
}

func isCanReleaseBeUpgraded(r *release.Release) (release.Status, bool) {
	if r.Info == nil {
		return release.StatusUnknown, false
	}
	// The release must neither be pending nor failed.
	return r.Info.Status, !r.Info.Status.IsPending() && r.Info.Status != release.StatusFailed
}

func replaceCustomResourceDefinitions(ctx context.Context, c client.Client, crdList []chart.CRD) error {
	if len(crdList) == 0 {
		return nil
	}
	for _, crdFile := range crdList {
		if crdFile.File == nil || crdFile.File.Data == nil {
			continue
		}
		// Read the manifest to an object.
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(crdFile.File.Data, crd); err != nil {
			return err
		}

		// Attempt to get the CRD from the cluster.
		var curObj = new(apiextensionsv1.CustomResourceDefinition)
		if err := c.Get(ctx, types.NamespacedName{Namespace: "", Name: crd.GetName()}, curObj); err != nil {
			// Let Helm handle the installation if the CRD doesn't exist yet.
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}

		// An update is used intentionally instead of a patch as esp. the last-applied-configuration annotation
		// can exceed the maximum characters and might have been pruned.
		// The update requires carrying over the resourceVersion from the currently deployed object.
		// TODO: Check max. last-applied-configuration annotation and prune if necessary.
		crd.SetResourceVersion(curObj.GetResourceVersion())
		if err := c.Update(ctx, crd); err != nil {
			return err
		}
	}
	return nil
}
