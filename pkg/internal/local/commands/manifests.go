// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
	"strings"
)

func manifestExample() string {
	return `
# Install manifests for Greenhouse into the target cluster (All manifests except Deployment - recommended)
greenhousectl dev manifests --name greenhouse-admin --namespace greenhouse --chart-path charts/manger

# Install only CRDs for Greenhouse into the target cluster
greenhousectl dev manifests --current-context --namespace greenhouse --chart-path charts/idproxy --crd-only

# Install manifests with excluded kinds for Greenhouse into the target cluster (Caution: Only exclude if you know what you are doing)
greenhousectl dev manifests --current-context --namespace greenhouse --chart-path charts/manager --excludeKinds Deployment --excludeKinds Job

# Install manifests for Greenhouse into the target cluster with values file
greenhousectl dev manifests --name greenhouse-admin --namespace greenhouse --chart-path charts/manager --values-path hack/localenv/sample.values.yaml

# Note: Only one of
	--name, --current-context can be used at a time
`
}

var manifestsCmd = &cobra.Command{
	Use:               "manifests",
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
	opts := make([]setup.HelmClientOption, 0)
	opts = append(opts, setup.WithChartPath(chartPath))
	opts = append(opts, setup.WithNamespace(namespaceName))
	opts = append(opts, setup.WithReleaseName(releaseName))

	if currentContext {
		opts = append(opts, setup.WithCurrentContext(currentContext))
	} else {
		if strings.TrimSpace(clusterName) != "" {
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
	manifestsCmd.Flags().StringVarP(&releaseName, "release", "r", "greenhouse", "Helm release name, Default value: greenhouse - e.g. your-release-name")
	manifestsCmd.Flags().StringVarP(&clusterName, "name", "c", "", "Name of the kind cluster - e.g. greenhouse-123 (without the kind prefix)")
	manifestsCmd.Flags().StringVarP(&valuesPath, "values-path", "v", "", "local absolute values file path - e.g. <path>/<to>/my-values.yaml")
	manifestsCmd.Flags().BoolVarP(&currentContext, "current-context", "x", false, "Use your current kubectl context")
	manifestsCmd.Flags().BoolVarP(&crdOnly, "crd-only", "d", false, "Install only CRDs")
	manifestsCmd.Flags().StringArrayVarP(&excludeKinds, "excludeKinds", "e", []string{"Deployment"}, "Exclude kinds from the generated manifests: ex: -e Deployment -e Job")

	manifestsCmd.MarkFlagsMutuallyExclusive("name", "current-context")
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("namespace"))
	cobra.CheckErr(manifestsCmd.MarkFlagRequired("chart-path"))

}
