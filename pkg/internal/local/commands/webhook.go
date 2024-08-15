// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:               "webhook",
	Short:             "Setup webhooks for Greenhouse (Validating and Mutating webhooks)",
	Long:              "Setup Validating and Mutating webhooks for Greenhouse controller development convenience",
	Example:           `greenhousectl dev setup webhook -c my-kind-cluster-name -n my-namespace -p path/to/chart -f path/to/Dockerfile`,
	DisableAutoGenTag: true,
	RunE:              processWebhook,
}

func processWebhook(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := make([]setup.HelmClientOption, 0)
	opts = append(opts, setup.WithChartPath(chartPath))
	opts = append(opts, setup.WithNamespace(namespaceName))
	opts = append(opts, setup.WithReleaseName("greenhouse-manager"))
	opts = append(opts, setup.WithClusterName(clusterName))

	if valuesPath != "" {
		opts = append(opts, setup.WithValuesPath(valuesPath))
	}

	if currentContext {
		opts = append(opts, setup.WithCurrentContext(currentContext))
	} else {
		if kubeConfigPath != "" {
			opts = append(opts, setup.WithKubeConfigPath(kubeConfigPath))
		}
	}

	hookCfg := &setup.Webhook{
		DockerFile: dockerFile,
		Envs: []setup.WebhookEnv{
			{
				Name:  "WEBHOOK_ONLY",
				Value: "true",
			},
		},
	}

	helmClient, err := setup.NewHelmClient(ctx, opts...)
	if err != nil {
		return err
	}
	m := setup.NewManifestsSetup(helmClient, hookCfg, []string{"Deployment"}, false)
	return m.Setup(ctx)
}

func init() {
	setupCmd.AddCommand(webhookCmd)
	webhookCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. my-cluster (without the kind prefix)")
	webhookCmd.Flags().StringVarP(&kubeConfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file")
	webhookCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	webhookCmd.Flags().StringVarP(&chartPath, "chart-path", "p", "", "local chart path where manifests are located - e.g. <path>/<to>/charts/manager")
	webhookCmd.Flags().StringVarP(&valuesPath, "values-path", "v", "", "local absolute values file path - e.g. <path>/<to>/my-values.yaml")
	webhookCmd.Flags().StringVarP(&dockerFile, "dockerfile", "f", "", "local path to the Dockerfile of greenhouse manager")
	webhookCmd.Flags().BoolVarP(&currentContext, "current-context", "x", false, "Use your current kubectl context")

	webhookCmd.MarkFlagsMutuallyExclusive("current-context", "kubeconfig")
	cobra.CheckErr(webhookCmd.MarkFlagRequired("name"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("chart-path"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("dockerfile"))
}
