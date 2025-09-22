// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/internal/local/setup"
)

type Config struct {
	Config []*clusterConfig `yaml:"config" json:"config"`
}

type clusterConfig struct {
	Cluster      *setup.Cluster       `yaml:"cluster" json:"cluster"`
	Dependencies []*ClusterDependency `yaml:"dependencies" json:"dependencies"`
}

type ClusterDependency struct {
	Manifest *setup.Manifest `yaml:"manifest" json:"manifest"`
}

func devSetupExample() string {
	return `
# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup -f dev-env/dev.config.yaml

- This will create an admin and a remote cluster
- Install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the admin cluster
- Depending on the devMode, it will install the webhook in-cluster or enable it for local development

Overriding certain values in dev.config.yaml:

- Override devMode for webhook development with d=true or devMode=true
- Override helm chart installation with c=true or crdOnly=true

e.g. greenhousectl dev setup -f dev-env/dev.config.yaml d=true
`
}

var (
	setupConfigFile string
	setupCmd        = &cobra.Command{
		Use:               "setup",
		Short:             "setup dev environment with a configuration file",
		Example:           devSetupExample(),
		DisableAutoGenTag: true,
		RunE:              processDevSetup,
	}
)

func processDevSetup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	devMode := getBoolArg([]string{"d", "devMode"}, args)
	crdOnly := getBoolArg([]string{"c", "crdOnly"}, args)

	_, err := os.Stat(setupConfigFile)
	if err != nil {
		return fmt.Errorf("config file - %s not found: %w", setupConfigFile, err)
	}
	f, err := os.ReadFile(setupConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read config file - %s: %w", setupConfigFile, err)
	}
	config := &Config{}
	err = yaml.Unmarshal(f, config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file - %s: %w", setupConfigFile, err)
	}

	for _, cfg := range config.Config {
		if cfg.Cluster == nil {
			return errors.New("cluster config is missing")
		}
		cfg.setClusterVersion()
		// in case of plugin development we check if the plugin directory env exists
		// if it does, we generate KinD config to enable hostPath mounts
		hostPathConfig, err := createHostPathConfig(cfg.Cluster.ConfigPath)
		if err != nil {
			return err
		}
		// re-write the config file with the hostPath config
		if hostPathConfig != "" {
			cfg.Cluster.ConfigPath = hostPathConfig
		}
		env := setup.NewExecutionEnv(devMode).
			WithClusterSetup(cfg.Cluster)
		for _, dep := range cfg.Dependencies {
			if crdOnly {
				dep.Manifest.CRDOnly = crdOnly
			}
			if dep.Manifest != nil && (dep.Manifest.CRDOnly || dep.Manifest.Webhook == nil) {
				env = env.WithLimitedManifests(ctx, dep.Manifest)
				continue
			}
			if dep.Manifest != nil && dep.Manifest.Webhook != nil {
				if hostPathConfig != "" {
					env = env.WithLocalPluginDev(dep.Manifest)
				}
				dep.Manifest.ExcludeKinds = append(
					dep.Manifest.ExcludeKinds,
					"Deployment",
					"Job",
					"MutatingWebhookConfiguration",
					"ValidatingWebhookConfiguration",
				)
				env = env.WithGreenhouseDevelopment(ctx, dep.Manifest)
			}
		}
		err = env.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// setClusterVersion - set the cluster version via K8S_VERSION env var or default to v1.31.0
func (c *clusterConfig) setClusterVersion() {
	if strings.TrimSpace(c.Cluster.Version) == "" {
		k8sVersion, ok := os.LookupEnv(k8sVersionEnvKey)
		if !ok {
			k8sVersion = defaultK8sVersion
		}
		c.Cluster.Version = k8sVersion
	}
}

func init() {
	setupCmd.Flags().StringVarP(&setupConfigFile, "config", "f", "", "configuration file path - e.g. -f dev-env/dev.config.yaml")
	cobra.CheckErr(setupCmd.MarkFlagRequired("config"))
}
