// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"os"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	kv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

	"github.com/cloudoperators/greenhouse/internal/internal/local/utils"

	"github.com/spf13/cobra"
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

func getBoolArg(matchArgs, args []string) (override, overrideVal bool) {
	for _, a := range args {
		// Split each argument into key and value
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 && slices.Contains(matchArgs, parts[0]) {
			// Use strconv to parse the value as a boolean
			parsedBool, err := strconv.ParseBool(parts[1])
			if err == nil {
				override = true
				overrideVal = parsedBool
				return
			}
			// If parsing fails, consider it invalid and return as not found
			break
		}
	}
	// If not found or invalid value, return false
	return
}

func getArgArray(matchArgs, args []string) []string {
	var result []string

	for _, a := range args {
		// Split each argument into key and value
		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 2 && slices.Contains(matchArgs, parts[0]) {
			// Append the argument to the result if the key matches
			result = append(result, parts[1])
		}
	}

	return result
}

func createHostPathConfig() (string, error) {
	pluginDir, ok := os.LookupEnv(utils.PluginDirectoryPath)
	if !ok {
		return "", nil
	}
	if strings.TrimSpace(pluginDir) == "" {
		return "", nil
	}
	kindConfig := kv1alpha4.Cluster{
		TypeMeta: kv1alpha4.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "kind.x-k8s.io/v1alpha4",
		},
		Nodes: []kv1alpha4.Node{
			{
				Role: kv1alpha4.ControlPlaneRole,
				ExtraMounts: []kv1alpha4.Mount{
					{
						HostPath:      pluginDir,
						ContainerPath: utils.PluginHostPath,
					},
				},
			},
		},
	}
	kindConfigBytes, err := yaml.Marshal(kindConfig)
	if err != nil {
		return "", err
	}
	return utils.RandomWriteToTmpFolder("plugin-config.yaml", string(kindConfigBytes))
}
