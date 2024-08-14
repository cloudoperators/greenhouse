package cmd

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/cluster"
	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:              "cluster",
	Short:            "Create, List and Delete kinD clusters",
	TraverseChildren: true,
}

// sub commands
var (
	createClusterCmd = &cobra.Command{
		Use:     "create",
		Short:   "Create a kinD cluster",
		Long:    "Create a kinD cluster and setup the greenhouse namespace optionally",
		Example: `localenv cluster create --name <my-cluster> --namespace <my-namespace>`,
		RunE:    processCreateCluster,
	}
)

func processCreateCluster(_ *cobra.Command, _ []string) error {
	setup := cluster.NewCmdCluster(clusterName, namespaceName)
	return setup.Setup()
}
func init() {
	clusterCmd.PersistentFlags().StringVarP(&clusterName, "name", "c", "", "create a kind cluster with a name")
	cobra.CheckErr(clusterCmd.MarkPersistentFlagRequired("name"))
	createClusterCmd.Flags().StringVarP(&namespaceName, "namespace", "n", "", "create a namespace in the cluster")
	clusterCmd.AddCommand(createClusterCmd)
	rootCmd.AddCommand(clusterCmd)
}
