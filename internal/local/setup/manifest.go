// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudoperators/greenhouse/internal/local/helm"
	"github.com/cloudoperators/greenhouse/internal/local/utils"
)

type Manifest struct {
	ReleaseName          string   `yaml:"release" json:"release"`
	ChartPath            string   `yaml:"chartPath" json:"chartPath"`
	ValuesPath           string   `yaml:"valuesPath" json:"valuesPath"`
	CRDOnly              bool     `yaml:"crdOnly" json:"crdOnly"`
	ExcludeKinds         []string `yaml:"excludeKinds" json:"excludeKinds"`
	Webhook              *Webhook `yaml:"webhook" json:"webhook"`
	hc                   helm.IHelm
	enableLocalPluginDev bool
}

func limitedManifestSetup(ctx context.Context, m *Manifest) Step {
	return func(env *ExecutionEnv) error {
		var clusterName, namespace string
		if env.cluster != nil {
			clusterName = env.cluster.Name
		}
		if env.cluster.Namespace != nil {
			namespace = *env.cluster.Namespace
		}
		err := m.prepareHelmClient(ctx, m, clusterName, namespace, env.cluster.kubeConfigPath)
		if err != nil {
			return err
		}
		resources, err := m.generateManifests(ctx)
		if err != nil {
			return err
		}
		return m.applyManifests(resources, namespace, env.cluster.kubeConfigPath)
	}
}

func dashboardSetup(ctx context.Context, m *Manifest) Step {
	return func(env *ExecutionEnv) error {
		var clusterName, namespace string
		if env.cluster != nil {
			clusterName = env.cluster.Name
		}
		if env.cluster.Namespace != nil {
			namespace = *env.cluster.Namespace
		}
		err := m.prepareHelmClient(ctx, m, clusterName, namespace, env.cluster.kubeConfigPath)
		if err != nil {
			return err
		}
		dashboardManifests, err := m.setupDashboard(ctx, clusterName, namespace)
		if err != nil {
			return err
		}
		err = m.applyManifests(dashboardManifests, namespace, env.cluster.kubeConfigPath)
		if err != nil {
			return err
		}
		env.info = append(env.info, m.getDashboardSetupInfo())
		return m.waitUntilDeploymentReady(ctx, clusterName, m.ReleaseName+dashboardDeploymentSuffix, namespace)
	}
}

// webhookManifestSetup - generates and applies manifest to the Cluster
// if webhook configuration is provided, modified webhook manifest are generated and applied
// if dev mode is enabled, webhook certs are extracted from the Cluster and saved to the local filesystem
func webhookManifestSetup(ctx context.Context, m *Manifest) Step {
	return func(env *ExecutionEnv) error {
		var clusterName, namespace string
		if env.cluster != nil {
			clusterName = env.cluster.Name
		}
		if env.cluster.Namespace != nil {
			namespace = *env.cluster.Namespace
		}
		err := m.prepareHelmClient(ctx, m, clusterName, namespace, env.cluster.kubeConfigPath)
		if err != nil {
			return err
		}
		resources, err := m.generateAllManifests(ctx)
		if err != nil {
			return err
		}
		excluded := m.resourceExclusion(resources)
		filtered := m.filterCustomResources(excluded)
		if m.Webhook == nil {
			utils.Log("no webhook configuration provided, skipping webhook kustomization")
			noWbManifests := excludeResources(filtered, []string{"MutatingWebhookConfiguration", "ValidatingWebhookConfiguration"})
			return m.applyManifests(noWbManifests, namespace, env.cluster.kubeConfigPath)
		}
		webHookManifests, err := m.setupWebhookManifest(resources, clusterName)
		if err != nil {
			return err
		}
		filtered = append(filtered, webHookManifests...)
		err = m.applyManifests(filtered, namespace, env.cluster.kubeConfigPath)
		if err != nil {
			return err
		}
		if m.Webhook.DevMode {
			err = m.extractWebhookCerts(ctx, clusterName, namespace)
			if err != nil {
				return err
			}
		}
		return m.waitUntilDeploymentReady(ctx, clusterName, m.ReleaseName+ManagerDeploymentNameSuffix, namespace)
	}
}

func (m *Manifest) prepareHelmClient(ctx context.Context, manifest *Manifest, clusterName, namespace, kubeConfigPath string) error {
	opts := []helm.ClientOption{
		helm.WithChartPath(manifest.ChartPath),
		helm.WithClusterName(clusterName),
		helm.WithNamespace(namespace),
		helm.WithReleaseName(manifest.ReleaseName),
		helm.WithValuesPath(manifest.ValuesPath),
		helm.WithKubeConfigPath(kubeConfigPath),
	}
	hc, err := helm.NewClient(ctx, opts...)
	if err != nil {
		return err
	}
	m.hc = hc
	return nil
}

// GenerateManifests - uses helm templating to explode the chart and returns the raw manifest
func (m *Manifest) generateManifests(ctx context.Context) ([]map[string]any, error) {
	resources, err := m.generateAllManifests(ctx)
	if err != nil {
		return nil, err
	}
	excluded := m.resourceExclusion(resources)
	return m.filterCustomResources(excluded), nil
}

func (m *Manifest) generateAllManifests(ctx context.Context) ([]map[string]any, error) {
	utils.Logf("generating manifest for chart %s...", m.ChartPath)
	templates, err := m.hc.Template(ctx)
	if err != nil {
		return nil, err
	}
	docs := strings.Split(templates, "---")
	resources := make([]map[string]any, 0)
	for _, doc := range docs {
		resource, err := utils.RawK8sInterface([]byte(doc))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}
		if resource != nil {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

// ApplyManifests - applies the given resources to the Cluster using kubectl
func (m *Manifest) applyManifests(resources []map[string]any, namespace, kubeConfigPath string) error {
	manifestsToApply := filterResourceApplicability(resources, "apply")
	manifestsToCreate := filterResourceApplicability(resources, "create")
	if len(manifestsToCreate) > 0 {
		createManifests, err := utils.Stringify(manifestsToCreate)
		if err != nil {
			return err
		}
		err = m.apply(createManifests, namespace, kubeConfigPath, "create")
		if err != nil {
			return err
		}
	}
	if len(manifestsToApply) > 0 {
		applyManifests, err := utils.Stringify(manifestsToApply)
		if err != nil {
			return err
		}
		return m.apply(applyManifests, namespace, kubeConfigPath, "apply")
	}
	return nil
}

func (m *Manifest) apply(manifests, namespace, kubeConfigPath, applyType string) error {
	utils.Log("applying manifest...")
	sh := utils.Shell{}
	tmpResourcePath, err := utils.RandomWriteToTmpFolder("kind-resources", manifests)
	if err != nil {
		return err
	}
	defer utils.CleanUp(tmpResourcePath)
	sh.Cmd = fmt.Sprintf("kubectl %s --kubeconfig=%s -f %s -n %s", applyType, kubeConfigPath, tmpResourcePath, namespace)
	return sh.Exec()
}

func (m *Manifest) resourceExclusion(resources []map[string]any) []map[string]any {
	if len(m.ExcludeKinds) == 0 {
		return resources
	}
	return excludeResources(resources, m.ExcludeKinds)
}

func excludeResources(resources []map[string]any, exclusions []string) []map[string]any {
	excludeResources := make([]map[string]any, 0)
	excludeResources = append(excludeResources, resources...)
	for i := 0; i < len(excludeResources); {
		if k, ok := excludeResources[i]["kind"].(string); ok && utils.SliceContains(exclusions, k) {
			excludeResources[i] = excludeResources[len(excludeResources)-1]
			excludeResources = excludeResources[:len(excludeResources)-1]
		} else {
			i++
		}
	}
	return excludeResources
}

func (m *Manifest) filterCustomResources(resources []map[string]any) []map[string]any {
	if m.CRDOnly {
		return filterResourcesBy(resources, "CustomResourceDefinition")
	}
	return resources
}

func filterResourcesBy(resources []map[string]any, filterBy string) []map[string]any {
	filteredResource := make([]map[string]any, 0)
	filteredResource = append(filteredResource, resources...)
	for i := 0; i < len(filteredResource); {
		if k, ok := filteredResource[i]["kind"].(string); ok && k != filterBy {
			filteredResource[i] = filteredResource[len(filteredResource)-1]
			filteredResource = filteredResource[:len(filteredResource)-1]
		} else {
			i++
		}
	}
	return filteredResource
}

func filterResourceApplicability(resources []map[string]any, applyType string) []map[string]any {
	filteredResource := make([]map[string]any, 0)
	key := "name"
	if applyType == "create" {
		key = "generateName"
	}
	for _, resource := range resources {
		if r, ok := resource["metadata"].(map[string]any)[key].(string); ok && strings.TrimSpace(r) != "" {
			filteredResource = append(filteredResource, resource)
		}
	}
	return filteredResource
}
