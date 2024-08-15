package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
)

var manifestsCmd = &cobra.Command{
	Use:               "manifests",
	Short:             "install manifests for Greenhouse",
	Long:              "install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster",
	Example:           `greenhousectl dev manifests --current-context --namespace greenhouse --release greenhouse --chart-path path/to/greenhouse/charts`,
	DisableAutoGenTag: true,
	RunE:              processManifests,
}

func processManifests(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := make([]setup.HelmClientOption, 0)
	opts = append(opts, setup.WithChartPath(chartPath))
	opts = append(opts, setup.WithNamespace(namespaceName))
	opts = append(opts, setup.WithReleaseName(releaseName))

	if currentContext {
		opts = append(opts, setup.WithCurrentContext(currentContext))
	} else {
		if kubeConfigPath != "" {
			opts = append(opts, setup.WithKubeConfigPath(kubeConfigPath))
		} else {
			opts = append(opts, setup.WithClusterName(clusterName))
		}
	}

	if valuesPath != "" {
		opts = append(opts, setup.WithValuesPath(valuesPath))
	}

	helmClient, err := setup.NewHelmClient(ctx, opts...)
	if err != nil {
		return err
	}

	m := setup.NewCmdManifests(helmClient, excludeKinds, crdOnly)
	template, err := m.GenerateManifests(ctx)
	if err != nil {
		return err
	}
	return m.ApplyManifests(template)
}

func init() {
	manifestsCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	manifestsCmd.Flags().StringVarP(&chartPath, "chart-path", "p", "", "local absolute chart path where manifests are located - e.g. <path>/<to>/charts/manager")
	manifestsCmd.Flags().StringVarP(&releaseName, "releaseName", "r", "greenhouse", "Helm release name, Default value: greenhouse - e.g. your-release-name")
	manifestsCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. greenhouse-123 (without the kind prefix)")
	manifestsCmd.Flags().StringVarP(&kubeConfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file")
	manifestsCmd.Flags().StringVarP(&valuesPath, "values-path", "v", "", "local absolute values file path - e.g. <path>/<to>/my-values.yaml")
	manifestsCmd.Flags().BoolVarP(&currentContext, "current-context", "x", false, "Use your current kubectl context")
	manifestsCmd.Flags().BoolVarP(&crdOnly, "crd-only", "d", false, "Install only CRDs")
	manifestsCmd.Flags().StringArrayVarP(&excludeKinds, "excludeKinds", "e", []string{"Deployment"}, "Exclude kinds from the generated manifests")

	manifestsCmd.MarkFlagsMutuallyExclusive("name", "kubeconfig")
	manifestsCmd.MarkFlagsMutuallyExclusive("name", "current-context")
	manifestsCmd.MarkFlagsMutuallyExclusive("kubeconfig", "current-context")
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("chart-path"))
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("releaseName"))

}
