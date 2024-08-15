// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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
		createLocalClusterCmd,
		deleteLocalClusterCmd,
		manifestsCmd,
		webhookCmd,
		setupCmd,
	}
}
