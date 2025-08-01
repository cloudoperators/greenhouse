// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	kv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

	"github.com/cloudoperators/greenhouse/internal/local/utils"

	"github.com/spf13/cobra"
)

const (
	defaultK8sVersion = "v1.31.0"
	k8sVersionEnvKey  = "K8S_VERSION"
)

func GetLocalSetupCommands() []*cobra.Command {
	return []*cobra.Command{
		setupCmd,
	}
}

func GenerateDevCommandDocs() []*cobra.Command {
	return []*cobra.Command{
		setupCmd,
		dashboardCmd,
	}
}

func getBoolArg(matchArgs, args []string) (boolArg bool) {
	for _, a := range args {
		// Split each argument into key and value
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 && slices.Contains(matchArgs, parts[0]) {
			// Use strconv to parse the value as a boolean
			parsedBool, err := strconv.ParseBool(parts[1])
			if err == nil {
				boolArg = parsedBool
				return
			}
			// If parsing fails, consider it invalid and return as not found
			break
		}
	}
	// If not found or invalid value, return false
	return
}

func createHostPathConfig(configFile string) (string, error) {
	pluginDir, ok := os.LookupEnv(utils.PluginDirectoryPath)
	if !ok {
		return "", nil
	}
	if strings.TrimSpace(pluginDir) == "" {
		return "", nil
	}
	var kindConfig kv1alpha4.Cluster
	pluginMount := extraMountForKinD(pluginDir)
	kindConfig = defaultHostPathConfig(pluginMount)
	if strings.TrimSpace(configFile) != "" {
		configBytes, err := readConfigFile(configFile)
		if err != nil {
			return "", err
		}
		err = yaml.Unmarshal(configBytes, &kindConfig)
		if err != nil {
			return "", err
		}
		if len(kindConfig.Nodes) > 0 {
			cpNodeIdx := slices.IndexFunc(kindConfig.Nodes, func(node kv1alpha4.Node) bool {
				return node.Role == kv1alpha4.ControlPlaneRole
			})
			if cpNodeIdx != -1 {
				if len(kindConfig.Nodes[cpNodeIdx].ExtraMounts) == 0 {
					kindConfig.Nodes[cpNodeIdx].ExtraMounts = make([]kv1alpha4.Mount, 0)
				}
				kindConfig.Nodes[cpNodeIdx].ExtraMounts = append(kindConfig.Nodes[cpNodeIdx].ExtraMounts, pluginMount)
			} else {
				kindConfig.Nodes = make([]kv1alpha4.Node, 0)
				kindConfig.Nodes = append(kindConfig.Nodes, newKinDNodeWithHostPath(pluginDir))
			}
		} else {
			kindConfig.Nodes = make([]kv1alpha4.Node, 0)
			kindConfig.Nodes = append(kindConfig.Nodes, newKinDNodeWithHostPath(pluginDir))
		}
	}
	kindConfigBytes, err := yaml.Marshal(kindConfig)
	if err != nil {
		return "", err
	}
	return utils.RandomWriteToTmpFolder("plugin-config.yaml", string(kindConfigBytes))
}

func newKinDNodeWithHostPath(pluginDir string) kv1alpha4.Node {
	return kv1alpha4.Node{
		Role:        kv1alpha4.ControlPlaneRole,
		ExtraMounts: []kv1alpha4.Mount{extraMountForKinD(pluginDir)},
	}
}

func extraMountForKinD(pluginDir string) kv1alpha4.Mount {
	return kv1alpha4.Mount{
		HostPath:      pluginDir,
		ContainerPath: utils.PluginHostPath,
	}
}

func defaultHostPathConfig(pluginMount kv1alpha4.Mount) kv1alpha4.Cluster {
	return kv1alpha4.Cluster{
		TypeMeta: kv1alpha4.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "kind.x-k8s.io/v1alpha4",
		},
		Nodes: []kv1alpha4.Node{
			{
				Role:        kv1alpha4.ControlPlaneRole,
				ExtraMounts: []kv1alpha4.Mount{pluginMount},
			},
		},
	}
}

func readConfigFile(configFile string) ([]byte, error) {
	if utils.CheckIfFileExists(configFile) {
		return os.ReadFile(configFile)
	}
	return nil, fmt.Errorf("config file - %s not found", configFile)
}
