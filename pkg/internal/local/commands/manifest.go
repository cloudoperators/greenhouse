// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
)

func manifestExample() string {
	return `
# Install manifests for Greenhouse into the target cluster (All manifests except Deployment - recommended)
greenhousectl dev setup manifest --name greenhouse-admin --namespace greenhouse --release greenhouse --chart-path charts/manager

# Install only CRDs for Greenhouse into the target cluster
greenhousectl dev setup manifest --name greenhouse-admin --namespace greenhouse --release greenhouse --chart-path charts/idproxy --crd-only

# Install manifests with excluded kinds for Greenhouse into the target cluster (Caution: Only exclude if you know what you are doing)
greenhousectl dev setup manifest --name greenhouse-admin --namespace greenhouse --release greenhouse --chart-path charts/manager --excludeKinds Deployment --excludeKinds Job

# Install manifests for Greenhouse into the target cluster with values file
greenhousectl dev setup manifest --name greenhouse-admin --namespace greenhouse --release greenhouse --chart-path charts/manager --values-path dev-env/localenv/sample.values.yaml
`
}

var manifestCmd = &cobra.Command{
	Use:               "manifest",
	Short:             "install manifests for Greenhouse",
	Long:              "install CRDs, Webhook definitions, RBACs, Certs, etc... for Greenhouse into the target cluster",
	Example:           manifestExample(),
	DisableAutoGenTag: true,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return validateFlagInputs(cmd.Flags())
	},
	RunE: processManifests,
}

func processManifests(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	manifest := &setup.Manifest{
		ReleaseName:  releaseName,
		ChartPath:    chartPath,
		ValuesPath:   valuesPath,
		CRDOnly:      crdOnly,
		ExcludeKinds: excludeKinds,
		Webhook:      nil,
	}
	err := setup.NewExecutionEnv().
		WithClusterSetup(clusterName, namespaceName, clusterVersion).
		WithLimitedManifests(ctx, manifest).
		Run()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	// required flags
	manifestCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. greenhouse-123 (without the kind prefix)")
	manifestCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "namespace to install the resources")
	manifestCmd.Flags().StringVar(&clusterVersion, "version", "", "create the cluster with a specific version - e.g. -v <v1.30.3>")
	manifestCmd.Flags().StringVarP(&chartPath, "chart-path", "p", "", "local absolute chart path where manifests are located - e.g. <path>/<to>/charts/manager")
	manifestCmd.Flags().StringVarP(&releaseName, "release", "r", "greenhouse", "Helm release name, Default value: greenhouse - e.g. your-release-name")
	// optional flags
	manifestCmd.Flags().StringVarP(&valuesPath, "values-path", "v", "", "local absolute values file path - e.g. <path>/<to>/my-values.yaml")
	manifestCmd.Flags().BoolVarP(&crdOnly, "crd-only", "d", false, "Install only CRDs")
	manifestCmd.Flags().StringArrayVarP(&excludeKinds, "excludeKinds", "e", []string{"Deployment"}, "Exclude kinds from the generated manifests: ex: -e Deployment -e Job")

	cobra.CheckErr(manifestCmd.MarkFlagRequired("name"))
	cobra.CheckErr(manifestCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(manifestCmd.MarkFlagRequired("chart-path"))
	cobra.CheckErr(manifestCmd.MarkFlagRequired("release"))

	setupCmd.AddCommand(manifestCmd)
}
