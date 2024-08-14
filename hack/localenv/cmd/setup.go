package cmd

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/setup"
	"github.com/spf13/cobra"
)

var (
	setupConfigFile string
	setupCmd        = &cobra.Command{
		Use:     "setup",
		Short:   "setup Greenhouse",
		Long:    "setup Greenhouse dev environment with a configuration file",
		Example: `localenv setup -f path/to/config/file`,
		RunE:    processSetup,
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
	setupCmd.Flags().StringVarP(&setupConfigFile, "config", "f", "", "configuration file path")
	cobra.CheckErr(setupCmd.MarkFlagRequired("config"))
	rootCmd.AddCommand(setupCmd)
}
