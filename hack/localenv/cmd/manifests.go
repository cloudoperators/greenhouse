package cmd

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/klient"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/manifests"
	"github.com/spf13/cobra"
)

var manifestsCmd = &cobra.Command{
	Use:               "manifests",
	Short:             "install manifests for Greenhouse",
	Long:              "install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster",
	Example:           `localenv manifests -x -n greenhouse -r greenhouse -p path/to/greenhouse/charts`,
	RunE:              processManifests,
	DisableAutoGenTag: true,
}

func processManifests(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := make([]klient.HelmClientOption, 0)
	opts = append(opts, klient.WithChartPath(chartPath))
	opts = append(opts, klient.WithNamespace(namespaceName))
	opts = append(opts, klient.WithReleaseName(releaseName))

	if currentContext {
		opts = append(opts, klient.WithCurrentContext(currentContext))
	} else {
		if kubeConfigPath != "" {
			opts = append(opts, klient.WithKubeConfigPath(kubeConfigPath))
		} else {
			opts = append(opts, klient.WithClusterName(clusterName))
		}
	}

	if valuesPath != "" {
		opts = append(opts, klient.WithValuesPath(valuesPath))
	}

	helmClient, err := klient.NewHelmClient(ctx, opts...)
	if err != nil {
		return err
	}

	m := manifests.NewCmdManifests(helmClient, excludeKinds, crdOnly)
	template, err := m.GenerateManifests(ctx)
	if err != nil {
		return err
	}
	return m.ApplyManifests(template)
}

func init() {
	rootCmd.AddCommand(manifestsCmd)
	manifestsCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	manifestsCmd.Flags().StringVarP(&chartPath, "chartPath", "p", "", "local absolute chart path where manifests are located - ex: <path>/charts/manager")
	manifestsCmd.Flags().StringVarP(&releaseName, "releaseName", "r", "greenhouse", "Helm release name, Default value: greenhouse, ex: your-release-name")
	manifestsCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - ex: greenhouse-123 (without the kind prefix)")
	manifestsCmd.Flags().StringVarP(&kubeConfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file")
	manifestsCmd.Flags().StringVarP(&valuesPath, "valuesPath", "v", "", "local absolute values file path - ex: <path>/values.yaml")
	manifestsCmd.Flags().BoolVarP(&currentContext, "current-context", "x", false, "Use your current kubectl context")
	manifestsCmd.Flags().BoolVarP(&crdOnly, "crd-only", "d", false, "Install only CRDs")
	manifestsCmd.Flags().StringArrayVarP(&excludeKinds, "excludeKinds", "e", []string{"Deployment"}, "Exclude kinds from the generated manifests")

	manifestsCmd.MarkFlagsMutuallyExclusive("name", "kubeconfig")
	manifestsCmd.MarkFlagsMutuallyExclusive("name", "current-context")
	manifestsCmd.MarkFlagsMutuallyExclusive("kubeconfig", "current-context")
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("chartPath"))
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("releaseName"))

}
