// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"

	"github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

var (
	settings = cli.New()

	// IsHelmDebug is configured via a flag and enables extensive debug logging for Helm actions.
	IsHelmDebug bool
)

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
	_, err = uninstallAction.Run(plugin.GetReleaseName())
	return false, err
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
	return action.NewGet(cfg).Run(plugin.GetReleaseName())
}

// TemplateHelmChartFromPlugin returns the rendered manifest or an error.
func TemplateHelmChartFromPlugin(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, plugin *greenhousev1alpha1.Plugin) (*release.Release, error) {
	helmRelease, err := installRelease(ctx, local, restClientGetter, pluginDefinitionSpec, plugin, true)
	if err != nil {
		return nil, err
	}
	return helmRelease, nil
}

// TemplateHelmChartFromPluginOptionValues returns the rendered manifest or an error.
// This function
func TemplateHelmChartFromPluginOptionValues(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinitionSpec *greenhousev1alpha1.PluginDefinitionSpec, plugin *greenhousev1alpha1.Plugin, optionValues []greenhousev1alpha1.PluginOptionValue) (*release.Release, error) {
	installAction, _, err := newHelmInstallAction(restClientGetter, plugin.Spec.ReleaseName, plugin.Spec.ReleaseNamespace, pluginDefinitionSpec.Version, true)
	if err != nil {
		return nil, err
	}

	helmChart, err := loadHelmChart(&installAction.ChartPathOptions, pluginDefinitionSpec.HelmChart, settings)
	if err != nil {
		return nil, err
	}

	resolvedValues, err := resolvePluginOptionValueFrom(ctx, local, plugin.Namespace, optionValues)
	if err != nil {
		return nil, err
	}

	helmValues, err := mergeChartAndPluginOptionValues(helmChart.Values, resolvedValues)
	if err != nil {
		return nil, err
	}

	return installAction.RunWithContext(ctx, helmChart, helmValues)
}

type ChartLoaderFunc func(name string) (*chart.Chart, error)

var ChartLoader ChartLoaderFunc = loader.Load

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

func installRelease(ctx context.Context, local client.Client, restClientGetter genericclioptions.RESTClientGetter, pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, plugin *greenhousev1alpha1.Plugin, isDryRun bool) (*release.Release, error) {
	installAction, capabilities, err := newHelmInstallAction(restClientGetter, plugin.GetReleaseName(), plugin.Spec.ReleaseNamespace, pluginDefinitionSpec.Version, isDryRun)
	if err != nil {
		return nil, err
	}

	helmChart, err := loadHelmChart(&installAction.ChartPathOptions, pluginDefinitionSpec.HelmChart, settings)
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
	if err := verifyKubeVersionIsCompatible(helmChart, capabilities); err != nil {
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

func newHelmInstallAction(restClientGetter genericclioptions.RESTClientGetter, releaseName, releaseNamespace, pluginDefinitionVersion string, isDryRun bool) (*action.Install, *chartutil.Capabilities, error) {
	cfg, err := newHelmAction(restClientGetter, releaseNamespace)
	if err != nil {
		return nil, nil, err
	}
	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = releaseName
	installAction.Namespace = releaseNamespace
	installAction.Timeout = GetHelmTimeout() // set a timeout for the installation to not be stuck in pending state
	installAction.CreateNamespace = true
	installAction.DependencyUpdate = true
	installAction.DryRun = isDryRun
	installAction.ClientOnly = isDryRun
	installAction.Description = pluginDefinitionVersion

	return installAction, cfg.Capabilities, nil
}

func debug(format string, v ...any) {
	if IsHelmDebug {
		format = "[debug] " + format
		log.FromContext(context.Background()).Info(fmt.Sprintf(format, v...))
	}
}

/*
ConvertFlatValuesToHelmValues shall converts flat values for a Helm chart yaml-compatible structure.
Example:
The input

	global.image.registry=foobar

is transformed to

	global:
	  image:
	    registry: foobar
*/
func ConvertFlatValuesToHelmValues(values []greenhousev1alpha1.PluginOptionValue) (map[string]any, error) {
	if values == nil {
		return make(map[string]any, 0), nil
	}
	helmValues := make(map[string]any, 0)
	for _, v := range values {
		if err := strvals.ParseJSON(fmt.Sprintf("%s=%s", v.Name, v.ValueJSON()), helmValues); err != nil {
			return nil, err
		}
	}
	return helmValues, nil
}

// Taken from: https://github.com/helm/helm/blob/v3.10.3/pkg/cli/values/options.go#L99-L116
func MergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	maps.Copy(out, a)
	for k, v := range b {
		if v, ok := v.(map[string]any); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]any); ok {
					out[k] = MergeMaps(bv, v)
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
func getValuesForHelmChart(ctx context.Context, c client.Client, helmChart *chart.Chart, plugin *greenhousev1alpha1.Plugin) (map[string]any, error) {
	// Copy the values from the Helm chart ensuring a non-nil map.
	helmValues := MergeMaps(make(map[string]any), helmChart.Values)
	// Get values defined in plugin.
	optionValues, err := resolvePluginOptionValueFrom(ctx, c, plugin.Namespace, plugin.Spec.OptionValues)
	if err != nil {
		return nil, err
	}
	return mergeChartAndPluginOptionValues(helmValues, optionValues)
}

// mergeChartAndPluginOptionValues merges the values defined in the Helm chart with the values defined in the PluginOptionValues
func mergeChartAndPluginOptionValues(helmValues map[string]any, optionValues []greenhousev1alpha1.PluginOptionValue) (map[string]any, error) {
	helmPluginValues, err := ConvertFlatValuesToHelmValues(optionValues)
	if err != nil {
		return nil, err
	}
	helmValues = MergeMaps(helmValues, helmPluginValues)
	return helmValues, nil
}

func resolvePluginOptionValueFrom(ctx context.Context, c client.Client, namespace string, optionValues []greenhousev1alpha1.PluginOptionValue) ([]greenhousev1alpha1.PluginOptionValue, error) {
	namedValues := make([]greenhousev1alpha1.PluginOptionValue, len(optionValues))
	copy(namedValues, optionValues)
	for idx, val := range namedValues {
		// Values already provided on plain text don't need to be extracted.
		if val.ValueFrom == nil {
			continue
		}
		// Retrieve value from secret.
		if val.ValueFrom.Secret != nil {
			valFromSecret, err := getValueFromSecret(ctx, c, namespace, val.ValueFrom.Secret.Name, val.ValueFrom.Secret.Key)
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
	values, err := resolvePluginOptionValueFrom(ctx, c, plugin.Namespace, plugin.Spec.OptionValues)
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

		switch {
		case v.Value != nil:
			buf = append(buf, v.Value.Raw...)

		case v.Expression != nil:
			buf = append(buf, []byte(*v.Expression)...)

		case v.ValueFrom != nil && v.ValueFrom.Ref != nil:
			buf = append(buf, []byte(v.ValueFrom.Ref.Name)...)
			buf = append(buf, []byte(v.ValueFrom.Ref.Kind)...)
			buf = append(buf, []byte(v.ValueFrom.Ref.Expression)...)

		default:
			continue
		}
	}

	checksum := sha256.Sum256(buf)
	return hex.EncodeToString(checksum[:]), nil
}
