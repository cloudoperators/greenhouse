// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
)

type Config struct {
	Config []*clusterConfig `json:"config"`
}

type clusterConfig struct {
	Cluster      *setup.Cluster       `json:"cluster"`
	Dependencies []*ClusterDependency `json:"dependencies"`
}

type ClusterDependency struct {
	Manifest *setup.Manifest `json:"manifest"`
}

func setupExample() string {
	return `
# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup -f dev-env/localenv/sample.config.json

- This will create an admin and a remote cluster
- Install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster
- Depending on the devMode, it will install the webhook in-cluster or enable it for local development
`
}

var (
	setupConfigFile string
	setupCmd        = &cobra.Command{
		Use:               "setup",
		Short:             "setup dev environment",
		Long:              "setup dev environment with a configuration file",
		Example:           setupExample(),
		DisableAutoGenTag: true,
		RunE:              processSetup,
	}
)

func processSetup(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	_, err := os.Stat(setupConfigFile)
	if err != nil {
		return fmt.Errorf("config file - %s not found: %w", setupConfigFile, err)
	}
	f, err := os.ReadFile(setupConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read config file - %s: %w", setupConfigFile, err)
	}
	config := &Config{}
	err = json.Unmarshal(f, config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file - %s: %w", setupConfigFile, err)
	}

	for _, cfg := range config.Config {
		namespace := ""
		if cfg.Cluster == nil {
			return errors.New("cluster config is missing")
		}
		if cfg.Cluster.Namespace != nil {
			namespace = *cfg.Cluster.Namespace
		}
		env := setup.NewExecutionEnv().
			WithClusterSetup(cfg.Cluster.Name, namespace)
		for _, dep := range cfg.Dependencies {
			if dep.Manifest != nil && dep.Manifest.Webhook == nil {
				env = env.WithLimitedManifests(ctx, dep.Manifest)
			}
			if dep.Manifest != nil && dep.Manifest.Webhook != nil {
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
	setupCmd.Flags().StringVarP(&setupConfigFile, "config", "f", "", "configuration file path - e.g. -f hack/localenv/sample.config.json")
	cobra.CheckErr(setupCmd.MarkFlagRequired("config"))
}
