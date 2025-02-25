// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
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
greenhousectl dev setup -f dev-env/localenv/dev.config.yaml

- This will create an admin and a remote cluster
- Install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the admin cluster
- Depending on the devMode, it will install the webhook in-cluster or enable it for local development

Overriding certain values in dev.config.yaml:

- Override devMode for webhook development with d=true or devMode=true
- Override helm chart installation with c=true or crdOnly=true
- Override environment variables for manager deployment with e="ENV_NAME=VALUE" or env="ENV_NAME=VALUE" (can be repeated)

e.g. greenhousectl dev setup -f dev-env/localenv/dev.config.yaml d=true e="WEBHOOK_ONLY=false" e="CONTROLLERS_ONLY=true"
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

	overrideWebhookDev, webhookDev := getBoolArg([]string{"d", "devMode"}, args)
	overrideCRDOnly, onlyCRD := getBoolArg([]string{"c", "crdOnly"}, args)
	overrideEnvs := getArgArray([]string{"e", "env"}, args)

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
		// in case of plugin development we check if the plugin directory env exists
		// if it does, we generate KinD config to enable hostPath mounts
		hostPathConfig, err := createHostPathConfig(cfg.Cluster.ConfigPath)
		if err != nil {
			return err
		}
		env := setup.NewExecutionEnv().
			WithClusterSetup(cfg.Cluster)
		for _, dep := range cfg.Dependencies {
			if overrideCRDOnly {
				dep.Manifest.CRDOnly = onlyCRD
			}
			if dep.Manifest != nil && (dep.Manifest.CRDOnly || dep.Manifest.Webhook == nil) {
				env = env.WithLimitedManifests(ctx, dep.Manifest)
				continue
			}
			if dep.Manifest != nil && dep.Manifest.Webhook != nil {
				if hostPathConfig != "" {
					env = env.WithLocalPluginDev(dep.Manifest)
				}
				if overrideWebhookDev {
					dep.Manifest.Webhook.DevMode = webhookDev
				}

				if len(overrideEnvs) > 0 {
					dep.Manifest.Webhook.AddOrOverrideEnv(overrideEnvs)
				}
				dep.Manifest.ExcludeKinds = append(
					dep.Manifest.ExcludeKinds,
					"Deployment",
					"Job",
					"MutatingWebhookConfiguration",
					"ValidatingWebhookConfiguration",
				)
				env = env.WithWebhookDevelopment(ctx, dep.Manifest)
			}
		}
		err = env.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func init() {
	setupCmd.Flags().StringVarP(&setupConfigFile, "config", "f", "", "configuration file path - e.g. -f dev-env/localenv/dev.config.yaml")
	cobra.CheckErr(setupCmd.MarkFlagRequired("config"))
}
