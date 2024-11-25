// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"gopkg.in/yaml.v3"
	"os"
	kv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	clusterName    string
	namespaceName  string
	clusterVersion string
	kindConfigPath string
	dockerFile     string
	releaseName    string
	chartPath      string
	valuesPath     string
	crdOnly        bool
	excludeKinds   []string
)

func GetLocalSetupCommands() []*cobra.Command {
	return []*cobra.Command{
		clusterCmd,
		setupCmd,
	}
}

func GenerateDevCommandDocs() []*cobra.Command {
	return []*cobra.Command{
		createClusterCmd,
		deleteClusterCmd,
		manifestCmd,
		webhookCmd,
		setupCmd,
	}
}

func validateFlagInputs(flags *pflag.FlagSet) error {
	invalidFlags := make([]string, 0)
	flags.VisitAll(func(flag *pflag.Flag) {
		switch flag.Value.Type() {
		case "string":
			_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
			if required && strings.TrimSpace(flag.Value.String()) == "" {
				invalidFlags = append(invalidFlags, flag.Name)
				return
			}
			if !required && flag.Changed && strings.TrimSpace(flag.Value.String()) == "" {
				invalidFlags = append(invalidFlags, flag.Name)
				return
			}
		case "stringArray":
			if flag.Changed {
				arr, err := flags.GetStringArray(flag.Name)
				if err != nil {
					invalidFlags = append(invalidFlags, flag.Name)
					return
				}
				for _, a := range arr {
					if strings.TrimSpace(a) == "" || strings.Contains(a, "-") {
						invalidFlags = append(invalidFlags, flag.Name)
						return
					}
				}
			}
		default:
			return
		}
	})
	if len(invalidFlags) > 0 {
		return fmt.Errorf("flag validation failed for: %s", strings.Join(invalidFlags, ", "))
	}
	return nil
}

func createHostPathConfig() (string, error) {
	pluginDir, ok := os.LookupEnv(utils.PluginDirectoryPath)
	if !ok {
		return "", nil
	}
	if strings.TrimSpace(pluginDir) == "" {
		return "", nil
	} else {
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
}
