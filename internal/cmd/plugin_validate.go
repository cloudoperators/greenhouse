// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
)

var pluginDefinitionValidateCmdUsage = "validate [plugindefinition.yaml path] [plugin.yaml path]"

func init() {
	pluginCmd.AddCommand(newPluginValidateCmd())
}

type pluginValidateOptions struct {
	pathToPluginDefinition, pathToPlugin string
}

func newPluginValidateCmd() *cobra.Command {
	o := &pluginValidateOptions{}
	return &cobra.Command{
		Use:   pluginDefinitionValidateCmdUsage,
		Short: "Validate a PluginDefinition",
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
		return errors.New(pluginDefinitionValidateCmdUsage)
	}
	return nil
}

func (o *pluginValidateOptions) complete(args []string) error {
	var err error
	o.pathToPluginDefinition, err = filepath.Abs(args[0])
	if err != nil {
		return err
	}
	o.pathToPlugin, err = filepath.Abs(args[1])
	return err
}

func (o *pluginValidateOptions) run() error {
	fmt.Printf("validating pluginDefinition %s with plugin %s\n", o.pathToPluginDefinition, o.pathToPlugin)
	// Load both the PluginDefinition and Plugin from the provided files.
	pluginDefinition, err := loadFromFile[greenhousev1alpha1.ClusterPluginDefinition](o.pathToPluginDefinition)
	if err != nil {
		return err
	}
	plugin, err := loadFromFile[greenhousev1alpha1.Plugin](o.pathToPlugin)
	if err != nil {
		return err
	}

	// Validate the Plugin against the PluginDefinition.
	if err = validateOptions(pluginDefinition, plugin); err != nil {
		return err
	}

	// Start validation of Helm Chart
	if err = validateHelmChart(pluginDefinition, plugin); err != nil {
		return err
	}
	fmt.Printf("successfully validated pluginDefinition %s\n", pluginDefinition.GetName())
	return nil
}

// validateOptions validates that all required options are set and that the values are valid.
func validateOptions(pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition, plugin *greenhousev1alpha1.Plugin) error {
	// Validate that all required options are set.
	var errList []error
	for _, option := range pluginDefinition.Spec.Options {
		var isSet = false
		for _, optionValue := range plugin.Spec.OptionValues {
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
		return fmt.Errorf("pluginDefinition %v and plugin %v are not compatible: %v", pluginDefinition.GetName(), plugin.GetName(), errString)
	}
}

func validateHelmChart(pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition, plugin *greenhousev1alpha1.Plugin) error {
	if pluginDefinition.Spec.HelmChart == nil {
		return nil
	}

	restClientGetter := clientutil.NewRestClientGetterFromRestConfig(ctrl.GetConfigOrDie(), plugin.Namespace)

	local, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)

	if err != nil {
		return err
	}

	fmt.Printf("rendering helm chart %s\n", pluginDefinition.Spec.HelmChart.String())
	_, err = helm.TemplateHelmChartFromPlugin(context.Background(), local, restClientGetter, pluginDefinition.Spec, plugin)
	return err
}
