// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
)

var clusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "Create / Delete kinD clusters",
	DisableAutoGenTag: true,
}

// sub commands
var (
	createClusterCmd = &cobra.Command{
		Use:               "create",
		Short:             "Create a kinD cluster",
		Long:              "Create a kinD cluster and setup the greenhouse namespace optionally",
		Example:           `greenhousectl dev cluster create --name <my-cluster-name> --namespace <my-namespace>`,
		DisableAutoGenTag: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlagInputs(cmd.Flags())
		},
		RunE: processCreateLocalCluster,
	}
	deleteClusterCmd = &cobra.Command{
		Use:               "delete",
		Short:             "Delete a kinD cluster",
		Long:              "Delete a specific kinD cluster",
		Example:           `greenhousectl dev cluster delete --name <my-cluster-name>`,
		DisableAutoGenTag: true,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return validateFlagInputs(cmd.Flags())
		},
		RunE: processDeleteLocalCluster,
	}
)

func processDeleteLocalCluster(_ *cobra.Command, _ []string) error {
	err := setup.NewExecutionEnv().WithClusterDelete(clusterName).Run()
	if err != nil {
		return err
	}
	return nil
}

func processCreateLocalCluster(_ *cobra.Command, _ []string) error {
	err := setup.NewExecutionEnv().WithClusterSetup(clusterName, namespaceName).Run()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	createClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "create a kind cluster with a name - e.g. -c <my-cluster>")
	createClusterCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "create a namespace in the cluster - e.g. -c <my-cluster> -n <my-namespace>")
	deleteClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "delete the kind cluster - e.g. -c <my-cluster>")
	cobra.CheckErr(createClusterCmd.MarkFlagRequired("name"))
	cobra.CheckErr(deleteClusterCmd.MarkFlagRequired("name"))
	clusterCmd.AddCommand(createClusterCmd)
	clusterCmd.AddCommand(deleteClusterCmd)
}
