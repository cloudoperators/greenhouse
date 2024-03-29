// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package tailscalestarter

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"tailscale.com/client/tailscale"

	"github.com/cloudoperators/greenhouse/pkg/version"
)

const programName = "tailscale_starter"

var (
	localClient tailscale.LocalClient
)

func init() {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	ctrl.SetLogger(zap.New(
		zap.UseFlagOptions(&opts)),
	)
}

// rootCmd for headscalectl.
func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     programName,
		Short:   "tailscale_starter is a tool to start tailscale and expose a health endpoint",
		Version: version.GetVersionTemplate(programName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("unexpected non-flag arguments to 'tailscale status'")
			}

			go func() {
				// start tailscale
				cmd := exec.Command("/tailscale/run.sh")
				cmd.Stdout = os.Stdout // or any other io.Writer
				cmd.Stderr = os.Stdout // or any other io.Writer
				if err := cmd.Run(); err != nil {
					log.FromContext(ctrl.SetupSignalHandler()).Error(err, "Error starting tailscaled")
					os.Exit(1)
				}
			}()

			httpServer := &http.Server{
				Addr:              ":8090",
				Handler:           newHealthMux(),
				IdleTimeout:       30 * time.Second,
				ReadHeaderTimeout: 20 * time.Second,
				ReadTimeout:       20 * time.Second,
			}

			if err := httpServer.ListenAndServe(); err != nil {
				log.FromContext(ctrl.SetupSignalHandler()).Error(err, "Error starting HTTP server")
				os.Exit(1)
			}

			return nil
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setPreAuthKey()
		},
	}
	cmd.DisableSuggestions = true
	cmd.Flags().StringVar(&localClient.Socket, "socket", "/var/run/tailscale/tailscaled.sock", "Path to the tailscale socket")
	return cmd
}

func Execute() {
	if err := rootCmd().Execute(); err != nil {
		log.FromContext(ctrl.SetupSignalHandler()).Error(err, "Error executing command")
		os.Exit(1)
	}
}
