// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vladimirvivien/gexe"
)

func init() {
	rootCmd.AddCommand(templateCmd)
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Template related commands",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		helm := gexe.ProgAvail("helm")
		if strings.TrimSpace(helm) == "" {
			return errors.New("please install helm first, see https://helm.sh/docs/intro/install/")
		}
		return nil
	},
}
