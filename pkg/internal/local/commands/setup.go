// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
)

func setupExample() string {
	return `
# Setup Greenhouse dev environment with a configuration file
greenhousectl dev setup -f hack/localenv/sample.config.json

- This will create an admin and a remote cluster
- Install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster
- Depending on the devMode, it will install the webhook in-cluster or enable it for local development
`
}

var (
	setupConfigFile string
	setupCmd        = &cobra.Command{
		Use:               "setup",
		Short:             "setup Greenhouse",
		Long:              "setup Greenhouse dev environment with a configuration file",
		Example:           setupExample(),
		DisableAutoGenTag: true,
		RunE:              processSetup,
	}
)

func processSetup(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	config, err := setup.NewGreenHouseFromConfig(setupConfigFile)
	if err != nil {
		return err
	}
	for _, cfg := range config.Config {
		err := cfg.Setup(ctx)
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
