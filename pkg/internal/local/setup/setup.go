package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"os"
)

type Config struct {
	Config []*ClusterConfig `json:"config"`
}

type ClusterConfig struct {
	Cluster        *Cluster             `json:"cluster"`
	CurrentContext *bool                `json:"currentContext"`
	KubeConfigPath *string              `json:"kubeConfigPath"`
	Dependencies   []*ClusterDependency `json:"dependencies"`
}

type ClusterDependency struct {
	Helm    *HelmConfig `json:"helm"`
	Webhook *Webhook    `json:"webhook"`
}

type HelmConfig struct {
	ReleaseName  string  `json:"releaseName"`
	ChartPath    string  `json:"chartPath"`
	ValuesPath   *string `json:"valuesPath"`
	CRDOnly      bool    `json:"crdOnly"`
	excludeKinds []string
	hc           IHelm
}

type ISetup interface {
	Setup(ctx context.Context) error
}

func (c *ClusterConfig) Setup(ctx context.Context) error {
	// setup cluster
	if c.CurrentContext != nil {
		c.Cluster = Configure(c.Cluster, *c.CurrentContext)
	}
	err := c.Cluster.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup cluster: %w", err)
	}
	err = c.prepareDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare dependencies: %w", err)
	}
	// setup manifests
	for _, dependency := range c.Dependencies {
		if dependency.Webhook != nil {
			dependency.Helm.excludeKinds = append(dependency.Helm.excludeKinds, "Deployment")
		}
		manifest := NewManifestsSetup(dependency.Helm.hc, dependency.Webhook, dependency.Helm.excludeKinds, dependency.Helm.CRDOnly)
		err = manifest.Setup(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}
	}
	// setup webhook
	return nil
}

func NewGreenHouseFromConfig(configFile string) (*Config, error) {
	_, err := os.Stat(configFile)
	if err != nil {
		return nil, fmt.Errorf("config file - %s not found: %w", configFile, err)
	}
	f, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file - %s: %w", configFile, err)
	}
	cfg := &Config{}
	err = json.Unmarshal(f, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file - %s: %w", configFile, err)
	}
	return cfg, nil
}

func (c *ClusterConfig) prepareDependencies(ctx context.Context) error {
	if c.Cluster == nil {
		utils.Log("warning: cluster configuration not provided, some functionalities will be skipped")
	}
	for _, dependency := range c.Dependencies {
		opts := make([]HelmClientOption, 0)
		opts = append(opts, WithChartPath(dependency.Helm.ChartPath))
		opts = append(opts, WithNamespace(*c.Cluster.Namespace))
		opts = append(opts, WithReleaseName(dependency.Helm.ReleaseName))
		if c.Cluster != nil {
			opts = append(opts, WithClusterName(c.Cluster.Name))
		}

		if c.CurrentContext != nil {
			opts = append(opts, WithCurrentContext(*c.CurrentContext))
		} else {
			if c.KubeConfigPath != nil {
				opts = append(opts, WithKubeConfigPath(*c.KubeConfigPath))
			}
		}

		if dependency.Helm.ValuesPath != nil {
			opts = append(opts, WithValuesPath(*dependency.Helm.ValuesPath))
		}

		helmClient, err := NewHelmClient(ctx, opts...)
		if err != nil {
			return err
		}
		dependency.Helm.hc = helmClient
	}
	return nil
}
