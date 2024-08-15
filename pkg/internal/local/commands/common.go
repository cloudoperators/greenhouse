package commands

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

func GetLocalSetupCommands() []*cobra.Command {
	return []*cobra.Command{
		localClusterCmd,
		manifestsCmd,
		webhookCmd,
		setupCmd,
	}
}
