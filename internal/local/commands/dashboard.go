// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/cloudoperators/greenhouse/internal/local/setup"
)

func dashboardSetupExample() string {
	return `
# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup dashboard -f dev-env/ui.config.yaml

- Installs the Greenhouse dashboard and CORS proxy into the admin cluster
`
}

var dashboardCmd = &cobra.Command{
	Use:               "dashboard",
	Short:             "setup dashboard for local development with a configuration file",
	Example:           dashboardSetupExample(),
	DisableAutoGenTag: true,
	RunE:              processDashboardSetup,
}

func processDashboardSetup(cmd *cobra.Command, _ []string) error {
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
	err = yaml.Unmarshal(f, config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file - %s: %w", setupConfigFile, err)
	}

	for _, cfg := range config.Config {
		if cfg.Cluster == nil {
			return errors.New("cluster config is missing")
		}
		env := setup.NewExecutionEnv(false).
			WithClusterSetup(cfg.Cluster)
		for _, dep := range cfg.Dependencies {
			if dep.Manifest.ChartPath == "charts/dashboard" {
				env = env.WithDashboardSetup(ctx, dep.Manifest)
				continue
			}
			if dep.Manifest != nil {
				env = env.WithLimitedManifests(ctx, dep.Manifest)
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
	dashboardCmd.Flags().StringVarP(&setupConfigFile, "config", "f", "", "configuration file path - e.g. -f dev-env/ui.config.yaml")
	cobra.CheckErr(dashboardCmd.MarkFlagRequired("config"))
	setupCmd.AddCommand(dashboardCmd)
}
