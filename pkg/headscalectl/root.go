// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package headscalectl

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

const programName = "headscalectl"

var (
	headscaleGRPCURL string
	headscaleAPIKey  string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&headscaleGRPCURL, "headscale-cli-address", "a", clientutil.GetEnvOrDefault("HEADSCALE_CLI_ADDRESS", ""), "Headscale API address. Can be set via HEADSCALE_CLI_ADDRESS env variable. Only GRPC is supported.")
	rootCmd.PersistentFlags().StringVarP(&headscaleAPIKey, "headscale-api-key", "k", clientutil.GetEnvOrDefault("HEADSCALE_CLI_API_KEY", ""), "Headscale API key. Can be set via HEADSCALE_CLI_API_KEY env variable")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output format. Empty for human-readable, 'json','json-line', 'yaml' or 'secret' | secret only works for apikey create")
	rootCmd.DisableSuggestions = true

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}

	ctrl.SetLogger(zap.New(
		zap.UseFlagOptions(&opts)),
	)
	rootCmd.DisableSuggestions = true
}

// rootCmd for headscalectl.
var rootCmd = &cobra.Command{
	Use:     programName,
	Short:   "Headscale command line tool for REST API",
	Version: version.GetVersionTemplate(programName),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.FromContext(ctrl.SetupSignalHandler()).Error(err, "Error executing command")
	}
}
