package cmd

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/klient"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/manifests"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:               "webhook",
	Short:             "Setup webhooks for Greenhouse (Validating and Mutating webhooks)",
	Long:              "Setup Validating and Mutating webhooks for Greenhouse controller development convenience",
	Example:           `localenv setup webhook -c my-kind-cluster-name -n my-namespace -p path/to/chart -f path/to/Dockerfile`,
	DisableAutoGenTag: true,
	RunE:              processWebhook,
}

func processWebhook(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := make([]klient.HelmClientOption, 0)
	opts = append(opts, klient.WithChartPath(chartPath))
	opts = append(opts, klient.WithNamespace(namespaceName))
	opts = append(opts, klient.WithReleaseName("greenhouse-manager"))
	opts = append(opts, klient.WithClusterName(clusterName))

	if currentContext {
		opts = append(opts, klient.WithCurrentContext(currentContext))
	} else {
		if kubeConfigPath != "" {
			opts = append(opts, klient.WithKubeConfigPath(kubeConfigPath))
		}
	}

	hookCfg := &manifests.Webhook{
		DockerFile: dockerFile,
		Envs: []manifests.WebhookEnv{
			{
				Name:  "WEBHOOK_ONLY",
				Value: "true",
			},
		},
	}

	helmClient, err := klient.NewHelmClient(ctx, opts...)
	if err != nil {
		return err
	}
	m := manifests.NewManifestsSetup(helmClient, hookCfg, []string{"Deployment"}, false)
	return m.Setup(ctx)
}

func init() {
	setupCmd.AddCommand(webhookCmd)
	webhookCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. my-cluster (without the kind prefix)")
	webhookCmd.Flags().StringVarP(&kubeConfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file")
	webhookCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	webhookCmd.Flags().StringVarP(&chartPath, "chartPath", "p", "", "local chart path where manifests are located - e.g. <path>/<to>/charts/manager")
	webhookCmd.Flags().StringVarP(&dockerFile, "dockerfile", "f", "", "local path to the Dockerfile of greenhouse manager")
	webhookCmd.Flags().BoolVarP(&currentContext, "current-context", "x", false, "Use your current kubectl context")

	webhookCmd.MarkFlagsMutuallyExclusive("current-context", "kubeconfig")
	cobra.CheckErr(webhookCmd.MarkFlagRequired("name"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("chartPath"))
	cobra.CheckErr(webhookCmd.MarkFlagRequired("dockerfile"))
}
