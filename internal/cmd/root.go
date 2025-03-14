// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/internal/version"
)

const programName = "greenhousectl"

// rootCmd for greenhousectl.
var rootCmd = &cobra.Command{
	Use:     programName,
	Short:   "The toolset for greenhouse.",
	Version: version.GetVersionTemplate(programName),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
