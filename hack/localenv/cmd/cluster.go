package cmd

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/cluster"
	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "Create, List and Delete kinD clusters",
	DisableAutoGenTag: true,
}

// sub commands
var (
	createClusterCmd = &cobra.Command{
		Use:               "create",
		Short:             "Create a kinD cluster",
		Long:              "Create a kinD cluster and setup the greenhouse namespace optionally",
		Example:           `localenv cluster create --name <my-cluster> --namespace <my-namespace>`,
		DisableAutoGenTag: true,
		RunE:              processCreateCluster,
	}
	listClusterCmd = &cobra.Command{
		Use:               "list",
		Short:             "List kinD clusters",
		Long:              "List all kinD clusters",
		Example:           `localenv cluster list`,
		DisableAutoGenTag: true,
		RunE:              processListClusters,
	}
	deleteClusterCmd = &cobra.Command{
		Use:               "delete",
		Short:             "Delete a kinD cluster",
		Long:              "Delete a specific kinD cluster",
		Example:           `localenv cluster delete --name my-cluster`,
		DisableAutoGenTag: true,
		RunE:              processDeleteCluster,
	}
)

func processDeleteCluster(_ *cobra.Command, _ []string) error {
	setup := cluster.NewCmdCluster(clusterName, "")
	return setup.Delete()
}

func processListClusters(_ *cobra.Command, _ []string) error {
	setup := cluster.NewCmdCluster("", "")
	return setup.List()
}

func processCreateCluster(_ *cobra.Command, _ []string) error {
	setup := cluster.NewCmdCluster(clusterName, namespaceName)
	return setup.Setup()
}
func init() {
	createClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "create a kind cluster with a name, e.g. localenv cluster create --name <my-cluster>")
	createClusterCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "create a namespace in the cluster, e.g. localenv cluster create --name <my-cluster> --namespace <my-namespace>")
	deleteClusterCmd.Flags().StringVarP(&clusterName, "name", "c", "", "delete the kind cluster, e.g. localenv cluster delete --name <my-cluster>")
	cobra.CheckErr(createClusterCmd.MarkFlagRequired("name"))
	cobra.CheckErr(deleteClusterCmd.MarkFlagRequired("name"))
	clusterCmd.AddCommand(createClusterCmd)
	clusterCmd.AddCommand(deleteClusterCmd)
	clusterCmd.AddCommand(listClusterCmd)
	rootCmd.AddCommand(clusterCmd)
}
