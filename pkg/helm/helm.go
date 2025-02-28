// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
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
	"github.com/cloudoperators/greenhouse/pkg/metrics"
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

// InstallOrUpgradeHelmChartFromPlugin installs a new or upgrades an existing Helm release for the given PluginDefinition and Plugin.
func InstallOrUpgradeHelmChartFromPlugin(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition, plugin *greenhousev1alpha1.Plugin) error {
	// Early return if the pluginDefinition is not helm based
	if pluginDefinition.Spec.HelmChart == nil {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonHelmChartIsNotDefined)
		return fmt.Errorf("no helm chart defined in pluginDefinition.Spec.HelmChart for pluginDefinition %s", plugin.Spec.PluginDefinition)
	}
	latestRelease, isReleaseExists, err := isReleaseExistsForPlugin(ctx, restClientGetter, plugin)
	if err != nil {
		return err
	}
	// A release does not exist. Install it.
	if !isReleaseExists {
		log.FromContext(ctx).Info("installing release for plugin", "namespace", plugin.Spec.ReleaseNamespace, "name", plugin.Name)
		_, err = installRelease(ctx, local, restClientGetter, pluginDefinition, plugin, false)
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonInstallFailed)
		return err
	}
	helmChart, err := locateChartForPlugin(restClientGetter, pluginDefinition)
	if err != nil {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
		return err
	}
	// Avoid attempts to upgrade a failed release and attempt to resurrect it.
	if latestRelease.Info != nil && latestRelease.Info.Status == release.StatusFailed {
		log.FromContext(ctx).Info("attempting to reset release status", "current status", latestRelease.Info.Status.String())
		if err := ResetHelmReleaseStatusToDeployed(restClientGetter, plugin); err != nil {
			metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
			return err
		}
	}
	// Avoid upgrading a currently pending release.
	if releaseStatus, ok := isCanReleaseBeUpgraded(latestRelease); !ok {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
		return fmt.Errorf("cannot upgrade release %s/%s in status %s", latestRelease.Namespace, latestRelease.Name, releaseStatus.String())
	}
	log.FromContext(ctx).Info("upgrading release", "namespace", plugin.Spec.ReleaseNamespace, "name", plugin.Name)

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
		return err
	}

	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects(), true); err != nil {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
		return err
	}

	if err := upgradeRelease(ctx, local, restClientGetter, pluginDefinition, plugin); err != nil {
		metrics.UpdateMetrics(plugin, metrics.MetricResultError, metrics.MetricReasonUpgradeFailed)
		return err
	}

	return nil
}

// ChartTest executes Helm chart tests and logs test pod logs if a test fails.
func ChartTest(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (hasTestHook bool, testPodLogs string, err error) {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return hasTestHook, "", err
	}

	testAction := action.NewReleaseTesting(cfg)
	// Used for fetching logs from test pods
	testAction.Namespace = plugin.Spec.ReleaseNamespace
	results, err := testAction.Run(plugin.Name)
	if err != nil {
		release, getErr := getLatestRelease(cfg, plugin.Name)
		if getErr != nil {
			log.FromContext(ctx).Error(getErr, "Failed to get latest release", "plugin", plugin.Name)
		} else {
			var err2 error
			testPodLogs, err2 = printTestPodLogs(ctx, testAction, release)
			if err2 != nil {
				log.FromContext(ctx).Error(err2, "Failed to retrieve test pod logs", "plugin", plugin.Name)
			}
		}
		return hasTestHook, testPodLogs, err
	}

	if results != nil && results.Hooks != nil {
		hasTestHook = slices.ContainsFunc(results.Hooks, func(h *release.Hook) bool {
			return slices.Contains(h.Events, release.HookTest)
		})
	}

	return hasTestHook, "", nil
}

func getLatestRelease(cfg *action.Configuration, releaseName string) (*release.Release, error) {
	getAction := action.NewGet(cfg)
	return getAction.Run(releaseName)
}

func printTestPodLogs(ctx context.Context, testAction *action.ReleaseTesting, rel *release.Release) (string, error) {
	var logBuffer bytes.Buffer
	if err := testAction.GetPodLogs(&logBuffer, rel); err != nil {
		return "", fmt.Errorf("error fetching test pod logs for release %s in namespace %s: %w", rel.Name, rel.Namespace, err)
	}

	logContent := logBuffer.String()
	if logContent == "" {
		log.FromContext(ctx).Info("No logs found for test pods", "release", rel.Name, "namespace", rel.Namespace)
	}

	return logContent, nil
}

// UninstallHelmRelease removes the Helm release for the given Plugin.
func UninstallHelmRelease(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (releaseNotFound bool, err error) {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return false, err
	}
	_, isReleaseExists, err := isReleaseExistsForPlugin(ctx, restClientGetter, plugin)
	if err != nil {
		return false, err
	}
	settings.RESTClientGetter()
	if !isReleaseExists {
		return true, nil
	}
	uninstallAction := action.NewUninstall(cfg)
	uninstallAction.KeepHistory = false
	_, err = uninstallAction.Run(plugin.Name)
	return false, err
}

// DiffChartToDeployedResources returns whether the Kubernetes objects, as specified in the Helm chart manifest, differ from the deployed state.
func DiffChartToDeployedResources(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition, plugin *greenhousev1alpha1.Plugin) (diffs DiffObjectList, isDrift bool, err error) {
	// Shortcut: If the Helm chart was changed we can skip below templating and diffing.
	var pluginStatusHelmChart string
	if plugin.Status.HelmReleaseStatus != nil && plugin.Status.HelmChart != nil {
		pluginStatusHelmChart = plugin.Status.HelmChart.String()
	}
	if pluginDefinition.Spec.HelmChart.String() != pluginStatusHelmChart {
		log.FromContext(ctx).Info("observed helm chart differs from pluginDefinition helm chart", "pluginDefinition", pluginDefinition.Spec.HelmChart.String(), "plugin", pluginStatusHelmChart)
		return nil, true, nil
	}

	helmRelease, exists, err := isReleaseExistsForPlugin(ctx, restClientGetter, plugin)
	switch {
	case err != nil:
		return nil, false, err
	case !exists:
		// the release should exist if the Status.HelmReleaseStatus was set
		// early return if the release was deleted
		return nil, true, nil
		// check if the release has the current pluginDefinition version set as description
		// this description is used to reconcile the version of the Plugin
	case helmRelease.Info.Description != pluginDefinition.Spec.Version:
		log.FromContext(ctx).Info("deployed helm chart version differs from pluginDefinition helm chart", "pluginDefinition", helmRelease.Info.Description, "plugin", pluginDefinition.Spec.Version)
		return nil, true, nil
	}

	helmTemplateRelease, err := TemplateHelmChartFromPlugin(ctx, local, restClientGetter, pluginDefinition, plugin)
	if err != nil {
		return nil, false, err
	}

	diffObjects, err := diffAgainstRelease(restClientGetter, plugin.Spec.ReleaseNamespace, helmTemplateRelease, helmRelease)
	if err != nil {
		return nil, false, err
	}
	diffCrds, err := diffAgainstRemoteCRDs(restClientGetter, helmRelease)
	if err != nil {
		return nil, false, err
	}
	diffObjects = append(diffObjects, diffCrds...)
	if len(diffObjects) > 0 {
		log.FromContext(ctx).Info("diff between manifest and release detected", "resources", diffObjects.String())
		return diffObjects, false, nil
	}

	c := plugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmDriftDetectedCondition)
	// Skip the drift detection if last DriftDetection Status Change or last Deployment was less than driftDetectionInterval ago
	switch {
	case c == nil: // HelmDriftDetectedCondition is not set
		return nil, false, nil
	case time.Since(plugin.Status.HelmReleaseStatus.LastDeployed.Time) < driftDetectionInterval: // Skip as last deployment was less than driftDetectionInterval ago
		return nil, false, nil
	case c.Status != metav1.ConditionUnknown && time.Since(c.LastTransitionTime.Time) < driftDetectionInterval: // Skip as HelmDriftDetectedCondition transitioned less than driftDetectionInterval ago
		return nil, false, nil
	}

	// Skip the drift detection if nothing changed with plugin option values.
	if plugin.Status.HelmReleaseStatus.PluginOptionChecksum != "" {
		currentPluginOptionChecksum, err := CalculatePluginOptionChecksum(ctx, local, plugin)
		if err == nil && plugin.Status.HelmReleaseStatus.PluginOptionChecksum == currentPluginOptionChecksum {
			return nil, false, nil
		}
	}

	diffObjects, err = diffAgainstLiveObjects(restClientGetter, plugin.Spec.ReleaseNamespace, helmTemplateRelease.Manifest)
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
func ResetHelmReleaseStatusToDeployed(restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) error {
	r, err := getLatestUpgradeableRelease(restClientGetter, plugin)
	if err != nil {
		return err
	}

	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return err
	}
	rollbackAction := action.NewRollback(cfg)
	rollbackAction.Version = r.Version
	rollbackAction.DisableHooks = true
	rollbackAction.Wait = true
	rollbackAction.Timeout = GetHelmTimeout()
	rollbackAction.MaxHistory = 5
	return rollbackAction.Run(r.Name)
}

// getLatestUpgradeableRelease returns the latest released that can be upgraded or an error.
func getLatestUpgradeableRelease(restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return nil, err
	}
	var latest *release.Release
	releases, err := action.NewHistory(cfg).Run(plugin.Name)
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
		return nil, fmt.Errorf("no release found to rollback to for plugin %s/%s", plugin.Spec.ReleaseNamespace, plugin.Name)
	}
	return latest, nil
}

// isReleaseExistsForPlugin checks whether a Helm release exists for the given Plugin.
func isReleaseExistsForPlugin(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (*release.Release, bool, error) {
	helmRelease, err := GetReleaseForHelmChartFromPlugin(ctx, restClientGetter, plugin)
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

// GetReleaseForHelmChartFromPlugin returns the Helm release for the given Plugin or an error.
func GetReleaseForHelmChartFromPlugin(_ context.Context, restClientGetter genericclioptions.RESTClientGetter, plugin *greenhousev1alpha1.Plugin) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return nil, err
	}
	return action.NewGet(cfg).Run(plugin.Name)
}

// TemplateHelmChartFromPlugin returns the rendered manifest or an error.
func TemplateHelmChartFromPlugin(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition, plugin *greenhousev1alpha1.Plugin) (*release.Release, error) {
	helmRelease, err := installRelease(ctx, local, restClientGetter, pluginDefinition, plugin, true)
	if err != nil {
		return nil, err
	}
	return helmRelease, nil
}

type ChartLoaderFunc func(name string) (*chart.Chart, error)

var ChartLoader ChartLoaderFunc = loader.Load

func locateChartForPlugin(restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition) (*chart.Chart, error) {
	cfg, err := newHelmAction(restClientGetter, corev1.NamespaceAll)
	if err != nil {
		return nil, err
	}

	// FIXME: we need to instantiate a action to set the registry in the ChartPathOptions
	cpo := &action.NewShowWithConfig(action.ShowChart, cfg).ChartPathOptions

	chartName := configureChartPathOptions(cpo, pluginDefinition.Spec.HelmChart)
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

func upgradeRelease(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition, plugin *greenhousev1alpha1.Plugin) error {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return err
	}
	upgradeAction := action.NewUpgrade(cfg)
	upgradeAction.Namespace = plugin.Spec.ReleaseNamespace
	upgradeAction.DependencyUpdate = true
	upgradeAction.MaxHistory = 5
	upgradeAction.Timeout = GetHelmTimeout() // set a timeout for the upgrade to not be stuck in pending state
	upgradeAction.Description = pluginDefinition.Spec.Version

	helmChart, err := loadHelmChart(&upgradeAction.ChartPathOptions, pluginDefinition.Spec.HelmChart, settings)
	if err != nil {
		return err
	}

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return err
	}

	helmValues, err := getValuesForHelmChart(ctx, local, helmChart, plugin)
	if err != nil {
		return err
	}
	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects(), true); err != nil {
		return err
	}

	// Do the Kubernetes version check beforehand to reflect incompatibilities in the Plugin status before attempting an installation or upgrade.
	if err := verifyKubeVersionIsCompatible(helmChart, cfg.Capabilities); err != nil {
		return err
	}
	helmChart.Metadata.KubeVersion = ""
	_, err = upgradeAction.RunWithContext(ctx, plugin.Name, helmChart, helmValues)
	return err
}

func installRelease(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinition *greenhousev1alpha1.PluginDefinition, plugin *greenhousev1alpha1.Plugin, isDryRun bool) (*release.Release, error) {
	cfg, err := newHelmAction(restClientGetter, plugin.Spec.ReleaseNamespace)
	if err != nil {
		return nil, err
	}
	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = plugin.Name
	installAction.Namespace = plugin.Spec.ReleaseNamespace
	installAction.Timeout = GetHelmTimeout() // set a timeout for the installation to not be stuck in pending state
	installAction.CreateNamespace = true
	installAction.DependencyUpdate = true
	installAction.DryRun = isDryRun
	installAction.ClientOnly = isDryRun
	installAction.Description = pluginDefinition.Spec.Version

	helmChart, err := loadHelmChart(&installAction.ChartPathOptions, pluginDefinition.Spec.HelmChart, settings)
	if err != nil {
		return nil, err
	}

	c, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return nil, err
	}

	if err := replaceCustomResourceDefinitions(ctx, c, helmChart.CRDObjects(), false); err != nil {
		return nil, err
	}
	helmValues, err := getValuesForHelmChart(ctx, local, helmChart, plugin)
	if err != nil {
		return nil, err
	}

	// Do the Kubernetes version check beforehand to reflect incompatibilities in the Plugin status before attempting an installation or upgrade.
	if err := verifyKubeVersionIsCompatible(helmChart, cfg.Capabilities); err != nil {
		return nil, err
	}
	helmChart.Metadata.KubeVersion = ""
	return installAction.RunWithContext(ctx, helmChart, helmValues)
}

func loadHelmChart(chartPathOptions *action.ChartPathOptions, reference *greenhousev1alpha1.HelmChartReference, settings *cli.EnvSettings) (*chart.Chart, error) {
	name := filepath.Base(reference.Name)
	chartPath := settings.RepositoryCache + "/" + name + "-" + reference.Version + ".tgz"

	if _, err := os.Stat(chartPath); errors.Is(err, os.ErrNotExist) {
		chartName := configureChartPathOptions(chartPathOptions, reference)
		chartPath, err = chartPathOptions.LocateChart(chartName, settings)
		if err != nil {
			return nil, err
		}
	}

	return ChartLoader(chartPath)
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
		format = "[debug] " + format
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
		if err := strvals.ParseJSON(fmt.Sprintf("%s=%s", v.Name, v.ValueJSON()), helmValues); err != nil {
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
// The order is important as the values defined in the Helm chart can be overridden by the values defined in the Plugin.
func getValuesForHelmChart(ctx context.Context, c client.Client, helmChart *chart.Chart, plugin *greenhousev1alpha1.Plugin) (map[string]interface{}, error) {
	// Copy the values from the Helm chart ensuring a non-nil map.
	helmValues := mergeMaps(make(map[string]interface{}), helmChart.Values)
	// Get values defined in plugin.
	pluginValues, err := getValuesFromPlugin(ctx, c, plugin)
	if err != nil {
		return nil, err
	}
	helmPluginValues, err := convertFlatValuesToHelmValues(pluginValues)
	if err != nil {
		return nil, err
	}
	helmValues = mergeMaps(helmValues, helmPluginValues)
	return helmValues, nil
}

func getValuesFromPlugin(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) ([]greenhousev1alpha1.PluginOptionValue, error) {
	namedValues := make([]greenhousev1alpha1.PluginOptionValue, len(plugin.Spec.OptionValues))
	copy(namedValues, plugin.Spec.OptionValues)
	for idx, val := range namedValues {
		// Values already provided on plain text don't need to be extracted.
		if val.ValueFrom == nil {
			continue
		}
		// Retrieve value from secret.
		if val.ValueFrom.Secret != nil {
			valFromSecret, err := getValueFromSecret(ctx, c, plugin.GetNamespace(), val.ValueFrom.Secret.Name, val.ValueFrom.Secret.Key)
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
	// Allow the upgrade to the first release, even if it failed.
	if r.Version == 1 {
		return r.Info.Status, !r.Info.Status.IsPending()
	}
	// The release must neither be pending nor failed.
	return r.Info.Status, !r.Info.Status.IsPending() && r.Info.Status != release.StatusFailed
}

func replaceCustomResourceDefinitions(ctx context.Context, c client.Client, crdList []chart.CRD, isUpgrade bool) error {
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
			if apierrors.IsNotFound(err) {
				// On install or dryRun: let Helm handle the installation if the CRD doesn't exist yet.
				if !isUpgrade {
					continue
				}
				// On upgrade: re-create the CRD based on helm chart if the CRD was deleted.
				if err := c.Create(ctx, crd); err != nil {
					return err
				}
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

// CalculatePluginOptionChecksum calculates a hash of plugin option values.
// Secret-type option values are extracted first and all values are sorted to ensure that order is not important when comparing checksums.
func CalculatePluginOptionChecksum(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (string, error) {
	values, err := getValuesFromPlugin(ctx, c, plugin)
	if err != nil {
		return "", err
	}
	// Sort the option values by Name to ensure consistent ordering.
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})

	buf := make([]byte, 0)
	for _, v := range values {
		buf = append(buf, []byte(v.Name)...)
		buf = append(buf, v.Value.Raw...)
	}

	checksum := sha256.Sum256(buf)
	return hex.EncodeToString(checksum[:]), nil
}
