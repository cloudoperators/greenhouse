// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

var pluginValidateCmdUsage = "validate [plugin.yaml path] [pluginConfig.yaml path]"

func init() {
	pluginCmd.AddCommand(newPluginValidateCmd())
}

type pluginValidateOptions struct {
	pathToPlugin, pathToPluginConfig string
}

func newPluginValidateCmd() *cobra.Command {
	o := &pluginValidateOptions{}
	return &cobra.Command{
		Use:   pluginValidateCmdUsage,
		Short: "Validate a Plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.validate(args); err != nil {
				return err
			}
			if err := o.complete(args); err != nil {
				return err
			}
			return o.run()
		},
	}
}

func (o *pluginValidateOptions) validate(args []string) error {
	if len(args) != 2 {
		return errors.New(pluginValidateCmdUsage)
	}
	return nil
}

func (o *pluginValidateOptions) complete(args []string) error {
	var err error
	o.pathToPlugin, err = filepath.Abs(args[0])
	if err != nil {
		return err
	}
	o.pathToPluginConfig, err = filepath.Abs(args[1])
	return err
}

func (o *pluginValidateOptions) run() error {
	fmt.Printf("validating plugin %s with pluginconfig %s\n", o.pathToPlugin, o.pathToPluginConfig)
	// Load both the Plugin and PluginConfig from the provided files.
	plugin, err := loadPlugin(o.pathToPlugin)
	if err != nil {
		return err
	}
	pluginConfig, err := loadPluginConfig(o.pathToPluginConfig)
	if err != nil {
		return err
	}

	// Validate the PluginConfig against the Plugin.
	if err = validateOptions(plugin, pluginConfig); err != nil {
		return err
	}

	// Start validation of Helm Chart
	if err = validateHelmChart(plugin, pluginConfig); err != nil {
		return err
	}
	fmt.Printf("successfully validated plugin %s\n", plugin.GetName())
	return nil
}

// validateOptions validates that all required options are set and that the values are valid.
func validateOptions(plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) error {
	// Validate that all required options are set.
	errList := []error{}
	for _, option := range plugin.Spec.Options {
		var isSet = false
		for _, optionValue := range pluginConfig.Spec.OptionValues {
			if optionValue.Name == option.Name {
				isSet = true
				if err := option.IsValidValue(optionValue.Value); err != nil {
					errList = append(errList, err)
				}
			}
		}
		if option.Required && !isSet {
			errList = append(errList, fmt.Errorf("required option %s not set", option.Name))
		}
	}
	switch {
	case len(errList) == 0:
		return nil
	default:
		errString := ""
		for _, err := range errList {
			errString += err.Error() + "\n"
		}
		return fmt.Errorf("plugin %v and pluginConfig %v are not compatible: %v", plugin.GetName(), pluginConfig.GetName(), errString)
	}
}

func validateHelmChart(plugin *greenhousev1alpha1.Plugin, pluginConfig *greenhousev1alpha1.PluginConfig) error {
	if plugin.Spec.HelmChart == nil {
		return nil
	}

	restClientGetter := clientutil.NewRestClientGetterFromRestConfig(ctrl.GetConfigOrDie(), pluginConfig.Namespace)

	local, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)

	if err != nil {
		return err
	}

	fmt.Printf("rendering helm chart %s\n", plugin.Spec.HelmChart.String())
	_, err = helm.TemplateHelmChartFromPlugin(context.Background(), local, restClientGetter, plugin, pluginConfig)
	return err
}

func loadPlugin(path string) (*greenhousev1alpha1.Plugin, error) {
	var plugin *greenhousev1alpha1.Plugin
	err := loadAndUnmarshalObject(path, &plugin)
	return plugin, err
}

func loadPluginConfig(path string) (*greenhousev1alpha1.PluginConfig, error) {
	var pluginConfig *greenhousev1alpha1.PluginConfig
	err := loadAndUnmarshalObject(path, &pluginConfig)
	return pluginConfig, err
}

func loadAndUnmarshalObject(path string, o interface{}) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(f, &o)
	return err
}
