// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"github.com/pkg/errors"
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
	ReleaseName  string  `json:"release"`
	ChartPath    string  `json:"chartPath"`
	ValuesPath   *string `json:"valuesPath"`
	CRDOnly      bool    `json:"crdOnly"`
	excludeKinds []string
	hc           IHelm
}

type ISetup interface {
	Setup(ctx context.Context) error
}

// Setup - sets up the cluster and dependencies as defined in the config.json
func (c *ClusterConfig) Setup(ctx context.Context) error {
	// setup cluster
	if c.CurrentContext != nil {
		c.Cluster = Configure(c.Cluster, *c.CurrentContext)
	}
	err := c.Cluster.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup cluster: %w", err)
	}
	// prepare helm client
	err = c.prepareDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare dependencies: %w", err)
	}
	// setup manifests + webhook (if provided)
	for _, dependency := range c.Dependencies {
		if dependency.Webhook != nil {
			dependency.Helm.excludeKinds = append(dependency.Helm.excludeKinds, "Deployment", "Job", "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration")
		}
		manifest := NewManifestsSetup(dependency.Helm.hc, dependency.Webhook, dependency.Helm.excludeKinds, dependency.Helm.CRDOnly, true)
		// invokes manifest setup - install CRDs, modified manager deployment, cert job, webhook configurations...
		err = manifest.Setup(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}
	}
	return nil
}

// NewGreenHouseFromConfig - returns a config object from the given config file
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
		if c.Cluster == nil {
			return errors.New("cluster configuration not provided")
		}
		opts = append(opts, WithClusterName(c.Cluster.Name))
		if c.CurrentContext != nil {
			opts = append(opts, WithCurrentContext(*c.CurrentContext))
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
