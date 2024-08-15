// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/setup"
	"github.com/spf13/cobra"
)

var localClusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "Create / Delete kinD clusters",
	DisableAutoGenTag: true,
}

// sub commands
var (
	createLocalClusterCmd = &cobra.Command{
		Use:               "create",
		Short:             "Create a kinD cluster",
		Long:              "Create a kinD cluster and setup the greenhouse namespace optionally",
		Example:           `greenhousectl dev cluster create --name <my-cluster> --namespace <my-namespace>`,
		DisableAutoGenTag: true,
		RunE:              processCreateLocalCluster,
	}
	deleteLocalClusterCmd = &cobra.Command{
		Use:               "delete",
		Short:             "Delete a kinD cluster",
		Long:              "Delete a specific kinD cluster",
		Example:           `greenhousectl dev cluster delete --name my-cluster`,
		DisableAutoGenTag: true,
		RunE:              processDeleteLocalCluster,
	}
)

func processDeleteLocalCluster(_ *cobra.Command, _ []string) error {
	s := setup.NewLocalCmdCluster(clusterName, "")
	return s.Delete()
}

func processCreateLocalCluster(_ *cobra.Command, _ []string) error {
	s := setup.NewLocalCmdCluster(clusterName, namespaceName)
	return s.Setup()
}
func init() {
	createLocalClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "create a kind cluster with a name - e.g. -c <my-cluster>")
	createLocalClusterCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "create a namespace in the cluster - e.g. -c <my-cluster> -n <my-namespace>")
	deleteLocalClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "delete the kind cluster - e.g. -c <my-cluster>")
	cobra.CheckErr(createLocalClusterCmd.MarkFlagRequired("name"))
	cobra.CheckErr(deleteLocalClusterCmd.MarkFlagRequired("name"))
	localClusterCmd.AddCommand(createLocalClusterCmd)
	localClusterCmd.AddCommand(deleteLocalClusterCmd)
}
