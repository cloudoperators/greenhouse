// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vladimirvivien/gexe"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/commands"
)

var devSetupCmd = &cobra.Command{
	Use:   "dev",
	Short: "Setup development environment",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// check if KinD is installed
		knd := gexe.ProgAvail("kind")
		if strings.TrimSpace(knd) == "" {
			return errors.New("please install KinD first, see https://kind.sigs.k8s.io/docs/user/quick-start/")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(devSetupCmd)
	devSetupCmd.AddCommand(commands.GetLocalSetupCommands()...)
}

func GenerateDevDocs() []*cobra.Command {
	return commands.GenerateDevCommandDocs()
}