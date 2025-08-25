// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vladimirvivien/gexe"
	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/json"

	helminternal "github.com/cloudoperators/greenhouse/internal/helm"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	PluginDefinitionKind        = "PluginDefinition"
	ClusterPluginDefinitionKind = "ClusterPluginDefinition"
	PluginPresetKind            = "PluginPreset"
)

func init() {
	pluginCmd.AddCommand(newPluginTemplatePresetCmd())
}

type PluginTemplatePresetOptions struct {
	pluginPresetPath     string
	pluginDefinitionPath string
	clusterName          string

	pluginPreset     *greenhousev1alpha1.PluginPreset
	pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	values           []greenhousev1alpha1.PluginOptionValue
}

func newPluginTemplatePresetCmd() *cobra.Command {
	o := &PluginTemplatePresetOptions{}

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Template the Helm Chart for a Plugin created from a given PluginPreset",
		Long: `The command performs a helm template on the Helm Chart referenced by the PluginPreset.
		PluginPreset and (Cluster-)PluginDefinition needs to match. Values are defaulted from (Cluster-)PluginDefinition,
		PluginPreset and optionally from ClusterOptionOverrides. References to Secrets and .global.greenhouse
		values are defaulted to their value name. 
		e.g. .global.greenhouse.baseDomain: ".global.greenhouse.baseDomain"`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			helm := gexe.ProgAvail("helm")
			if strings.TrimSpace(helm) == "" {
				return errors.New("please install helm first, see https://helm.sh/docs/intro/install/")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.validate(); err != nil {
				return err
			}

			if err := o.complete(); err != nil {
				return err
			}

			return o.run()
		},
	}

	cmd.Flags().StringVar(&o.pluginPresetPath, "pluginpreset", "", "Path to PluginPreset YAML file (required)")
	cmd.Flags().StringVar(&o.pluginDefinitionPath, "plugindefinition", "", "Path to (Cluster-)PluginDefinition YAML file (required)")
	cmd.Flags().StringVar(&o.clusterName, "cluster", "", "Cluster name (required)")

	if err := cmd.MarkFlagRequired("pluginpreset"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "pluginpreset")
	}

	if err := cmd.MarkFlagRequired("plugindefinition"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "plugindefinition")
	}

	if err := cmd.MarkFlagRequired("cluster"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "cluster")
	}

	return cmd
}

func (o *PluginTemplatePresetOptions) validate() error {
	if _, err := os.Stat(o.pluginPresetPath); os.IsNotExist(err) {
		return fmt.Errorf("PluginPreset file does not exist: %s", o.pluginPresetPath)
	}

	if _, err := os.Stat(o.pluginDefinitionPath); os.IsNotExist(err) {
		return fmt.Errorf("ClusterPluginDefinition file does not exist: %s", o.pluginDefinitionPath)
	}

	return nil
}

func (o *PluginTemplatePresetOptions) complete() error {
	var err error
	o.pluginDefinition, err = loadFromFile[greenhousev1alpha1.ClusterPluginDefinition](o.pluginDefinitionPath)
	if err != nil {
		return err
	}

	o.pluginPreset, err = loadFromFile[greenhousev1alpha1.PluginPreset](o.pluginPresetPath)
	if err != nil {
		return err
	}

	if err := o.validateCompatibility(); err != nil {
		return err
	}

	if err := o.prepareValues(); err != nil {
		return err
	}

	return nil
}

func (o *PluginTemplatePresetOptions) run() error {
	helmValues, err := helminternal.ConvertFlatValuesToHelmValues(o.values)
	if err != nil {
		return err
	}

	valuesYAML, err := yaml.Marshal(helmValues)
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "values-*.yaml")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if _, err := f.Write(valuesYAML); err != nil {
		return err
	}

	return o.runHelmTemplate(f.Name())
}

func (o *PluginTemplatePresetOptions) validateCompatibility() error {
	if o.pluginPreset.Kind != "PluginPreset" {
		return fmt.Errorf("expected PluginPreset kind, got %q in file %s", o.pluginPreset.Kind, o.pluginPresetPath)
	}

	if o.pluginDefinition.Kind != ClusterPluginDefinitionKind && o.pluginDefinition.Kind != PluginDefinitionKind {
		return fmt.Errorf("expected either %s or %s kind, got %q in file %s", ClusterPluginDefinitionKind, PluginDefinitionKind, o.pluginDefinition.Kind, o.pluginDefinitionPath)
	}

	expectedPluginDef := o.pluginPreset.Spec.Plugin.PluginDefinition
	actualPluginDef := o.pluginDefinition.Name
	if expectedPluginDef != actualPluginDef {
		return fmt.Errorf("PluginPreset references (Cluster-)PluginDefinition '%s' but provided file defines '%s'", expectedPluginDef, actualPluginDef)
	}

	if o.pluginDefinition.Spec.HelmChart == nil {
		return fmt.Errorf("(Cluster-)PluginDefinition '%s' must have a HelmChart reference", actualPluginDef)
	}

	return nil
}

func (o *PluginTemplatePresetOptions) prepareValues() error {
	// Get greenhouse values.
	greenhouseValues, err := o.getGreenhouseValuesForTemplate()
	if err != nil {
		return err
	}

	// Start with ClusterPluginDefinition defaults and merge with default greenhouse values.
	values := helminternal.MergePluginAndPluginOptionValueSlice(
		o.pluginDefinition.Spec.Options,
		greenhouseValues,
	)

	// Merge PluginPreset values.
	values = helminternal.MergePluginOptionValues(values, o.pluginPreset.Spec.Plugin.OptionValues)

	// Merge cluster overrides.
	values = helminternal.MergePluginOptionValues(values, o.getClusterSpecificOverrides())

	// Process secrets to literals.
	values, err = o.processSecretsToLiterals(values)
	if err != nil {
		return err
	}

	o.values = values
	return nil
}

func (o *PluginTemplatePresetOptions) runHelmTemplate(valuesFile string) error {
	chartRef := o.pluginDefinition.Spec.HelmChart

	var args []string
	if strings.HasPrefix(chartRef.Repository, "oci://") {
		args = []string{
			"template",
			o.pluginPreset.Name,
			fmt.Sprintf("%s/%s", chartRef.Repository, chartRef.Name),
			"--namespace", o.pluginPreset.Spec.Plugin.ReleaseNamespace,
			"--values", valuesFile,
			"--version", chartRef.Version,
		}
	} else {
		args = []string{
			"template",
			o.pluginPreset.Name,
			chartRef.Name,
			"--repo", chartRef.Repository,
			"--version", chartRef.Version,
			"--namespace", o.pluginPreset.Spec.Plugin.ReleaseNamespace,
			"--values", valuesFile,
		}
	}

	cmd := exec.CommandContext(context.Background(), "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm template failed: %w\nCommand: helm %s\nOutput: %s", err, strings.Join(args, " "), string(output))
	}

	fmt.Print(string(output))
	return nil
}

func (o *PluginTemplatePresetOptions) getGreenhouseValuesForTemplate() ([]greenhousev1alpha1.PluginOptionValue, error) {
	literalPaths := []string{
		"global.greenhouse.clusterNames",
		"global.greenhouse.teamNames",
		"global.greenhouse.baseDomain",
		"global.greenhouse.ownedBy",
	}

	var values []greenhousev1alpha1.PluginOptionValue

	for _, path := range literalPaths {
		value, err := createPluginOptionValue(path, path)
		if err != nil {
			return nil, fmt.Errorf("failed to create greenhouse value for %s: %w", path, err)
		}
		values = append(values, *value)
	}

	if o.pluginPreset.Namespace != "" {
		orgValue, err := createPluginOptionValue("global.greenhouse.organizationName", o.pluginPreset.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create organization name value: %w", err)
		}
		values = append(values, *orgValue)
	}

	if o.clusterName != "" {
		clusterValue, err := createPluginOptionValue("global.greenhouse.clusterName", o.clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster name value: %w", err)
		}
		values = append(values, *clusterValue)
	}

	return values, nil
}

func createPluginOptionValue(name, value string) (*greenhousev1alpha1.PluginOptionValue, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return &greenhousev1alpha1.PluginOptionValue{Name: name, Value: &apiextensionsv1.JSON{Raw: raw}}, nil
}

func (o *PluginTemplatePresetOptions) getClusterSpecificOverrides() []greenhousev1alpha1.PluginOptionValue {
	for _, override := range o.pluginPreset.Spec.ClusterOptionOverrides {
		if override.ClusterName == o.clusterName {
			return override.Overrides
		}
	}
	return []greenhousev1alpha1.PluginOptionValue{}
}

func (o *PluginTemplatePresetOptions) processSecretsToLiterals(values []greenhousev1alpha1.PluginOptionValue) ([]greenhousev1alpha1.PluginOptionValue, error) {
	for i := range values {
		if values[i].ValueFrom != nil && values[i].ValueFrom.Secret != nil {
			literal := fmt.Sprintf("%s/%s", values[i].ValueFrom.Secret.Name, values[i].ValueFrom.Secret.Key)
			raw, err := json.Marshal(literal)
			if err != nil {
				return nil, err
			}
			values[i].Value = &apiextensionsv1.JSON{Raw: raw}
			values[i].ValueFrom = nil
		}
	}
	return values, nil
}
