package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var rootCmd = &cobra.Command{
	Use:     "localenv",
	Version: "0.0.1",
	Short:   "Local Environment CLI to bootstrap Greenhouse local dev environment",
	Long:    "Use localenv CLI to setup KinD cluster, Greenhouse manifests, webhook, etc...",
	Example: "localenv --help",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Whoops! There was an error - ===== 😵 %s", err.Error())
	}
}
