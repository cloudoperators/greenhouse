package cmd

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/commands"
	"github.com/spf13/cobra"
)

var devSetupCmd = &cobra.Command{
	Use:   "dev",
	Short: "Setup development environment",
}

func init() {
	rootCmd.AddCommand(devSetupCmd)
	devSetupCmd.AddCommand(commands.GetLocalSetupCommands()...)
}
