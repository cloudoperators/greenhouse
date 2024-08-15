// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
