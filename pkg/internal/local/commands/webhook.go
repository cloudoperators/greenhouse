// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
)

var devMode bool

func webhookExample() string {
	return `
# Setup webhook for Greenhouse controller development convenience (Webhooks run in cluster)
greenhousectl dev setup webhook --name kind-cluster-name --namespace greenhouse --release greenhouse --chart-path charts/manager --dockerfile ./

# Setup webhook for Greenhouse webhook development convenience (Webhooks run local)
greenhousectl dev setup webhook --name kind-cluster-name --namespace greenhouse --release greenhouse --chart-path charts/manager --dockerfile ./ --dev-mode

# Additionally provide values file (defaults may not work since charts change over time)
greenhousectl dev setup webhook --name kind-cluster-name --namespace greenhouse --release greenhouse --chart-path charts/manager --dockerfile ./ --values-path hack/localenv/sample.values.yaml

`
}

var webhookCmd = &cobra.Command{
	Use:               "webhook",
	Short:             "Setup webhooks for Greenhouse (Validating and Mutating webhooks)",
	Long:              "Setup Validating and Mutating webhooks for Greenhouse controller development convenience",
	Example:           webhookExample(),
	DisableAutoGenTag: true,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return validateFlagInputs(cmd.Flags())
	},
	RunE: processWebhook,
}

func processWebhook(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := make([]setup.HelmClientOption, 0)
	opts = append(opts, setup.WithChartPath(chartPath))
	opts = append(opts, setup.WithNamespace(namespaceName))
	opts = append(opts, setup.WithReleaseName(releaseName))
	opts = append(opts, setup.WithClusterName(clusterName))

	if valuesPath != "" {
		opts = append(opts, setup.WithValuesPath(valuesPath))
	}

	hookCfg := &setup.Webhook{
		DockerFile: dockerFile,
		Envs: []setup.WebhookEnv{
			{
				Name:  "WEBHOOK_ONLY",
				Value: "true",
			},
		},
		DevMode: devMode,
	}

	helmClient, err := setup.NewHelmClient(ctx, opts...)
	if err != nil {
		return err
	}
	m := setup.NewManifestsSetup(helmClient, hookCfg, []string{"Deployment", "Job", "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration"}, false, true)
	return m.Setup(ctx)
}

func init() {
	setupCmd.AddCommand(webhookCmd)
	webhookCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. my-cluster (without the kind prefix)")
	webhookCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	webhookCmd.Flags().StringVarP(&chartPath, "chart-path", "p", "", "local chart path where manifests are located - e.g. <path>/<to>/charts/manager")
	webhookCmd.Flags().StringVarP(&valuesPath, "values-path", "v", "", "local absolute values file path - e.g. <path>/<to>/my-values.yaml")
	webhookCmd.Flags().StringVarP(&dockerFile, "dockerfile", "f", "", "local path to the Dockerfile of greenhouse manager")
	webhookCmd.Flags().StringVarP(&releaseName, "release", "r", "greenhouse", "Helm release name, Default value: greenhouse - e.g. your-release-name")
	webhookCmd.Flags().BoolVarP(&devMode, "dev-mode", "m", false, "Enable dev mode for webhook setup - Note: Admission Webhooks will be modified for local development")

	cobra.CheckErr(webhookCmd.MarkFlagRequired("name"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("chart-path"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("dockerfile"))
}
