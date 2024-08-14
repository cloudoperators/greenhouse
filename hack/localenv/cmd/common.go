package cmd

import "github.com/spf13/cobra"

var (
	clusterName    string
	namespaceName  string
	dockerFile     string
	releaseName    string
	chartPath      string
	valuesPath     string
	kubeConfigPath string
	currentContext bool
	crdOnly        bool
	excludeKinds   []string
)

func GetCommands() []*cobra.Command {
	return []*cobra.Command{
		clusterCmd,
		createClusterCmd,
		deleteClusterCmd,
		listClusterCmd,
		manifestsCmd,
		webhookCmd,
		setupCmd,
	}
}
