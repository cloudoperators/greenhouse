// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jeremywohl/flatten/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var pluginGenerateCmdUsage = "generate [Helm chart path] [output path]"

func init() {
	pluginCmd.AddCommand(newPluginGenerateCmd())
}

type pluginGenerateOptions struct {
	helmChartPath, outPath string
}

func newPluginGenerateCmd() *cobra.Command {
	o := &pluginGenerateOptions{}
	return &cobra.Command{
		Use:   pluginGenerateCmdUsage,
		Short: "Create a Greenhouse PluginDefinition based on an existing Helm Chart",
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

func (o *pluginGenerateOptions) validate(args []string) error {
	if len(args) != 2 {
		return errors.New(pluginGenerateCmdUsage)
	}
	return nil
}

func (o *pluginGenerateOptions) complete(args []string) error {
	var err error
	o.helmChartPath, err = filepath.Abs(args[0])
	if err != nil {
		return err
	}
	o.outPath, err = filepath.Abs(args[1])
	return err
}

func (o *pluginGenerateOptions) run() error {
	helmChart, err := loader.Load(o.helmChartPath)
	if err != nil {
		return err
	}
	if helmChart.Metadata == nil || helmChart.Metadata.Version == "" {
		fmt.Println("the Helm chart must have a version")
		os.Exit(1)
	}
	// Ensure directory.
	pluginDirectory := filepath.Join(o.outPath, helmChart.Name(), helmChart.Metadata.Version)
	fmt.Printf("creating directory for extension: %s\n", pluginDirectory)
	if err := os.MkdirAll(pluginDirectory, 0755); err != nil {
		return err
	}
	// Write output.
	pluginDefinition, err := helmChartToPlugin(helmChart)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(pluginDefinition)
	if err != nil {
		return err
	}
	yamlBytes, err := jsonToYaml(jsonBytes)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pluginDirectory, "plugindefinition.yaml"), yamlBytes, 0755); err != nil {
		return err
	}
	return nil
}

func helmChartToPlugin(helmChart *chart.Chart) (*greenhousev1alpha1.ClusterPluginDefinition, error) {
	pluginVersion := "1.0.0"
	if helmChart.Metadata != nil && helmChart.Metadata.Version != "" {
		pluginVersion = helmChart.Metadata.Version
	}
	pluginValues, err := chartValuesToNamedValues(helmChart.Values)
	if err != nil {
		return nil, err
	}
	return &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", helmChart.Name(), helmChart.Metadata.Version),
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Version:     pluginVersion,
			Description: helmChart.Name(),
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:       helmChart.Name(),
				Repository: "TODO: Repository for this Helm chart.",
				Version:    helmChart.Metadata.Version,
			},
			UIApplication: &greenhousev1alpha1.UIApplicationReference{
				URL:     "TODO: Javascript asset server URL.",
				Name:    helmChart.Name(),
				Version: "latest",
			},
			Options: pluginValues,
		},
	}, nil
}

func chartValuesToNamedValues(chartValues map[string]any) ([]greenhousev1alpha1.PluginOption, error) {
	if chartValues == nil {
		return nil, nil
	}
	flatChartValues, err := flatten.Flatten(chartValues, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}

	namedValues := make([]greenhousev1alpha1.PluginOption, 0)
	for k, v := range flatChartValues {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		namedValues = append(namedValues, greenhousev1alpha1.PluginOption{
			Name:        k,
			Description: k,
			Default:     &apiextensionsv1.JSON{Raw: raw},
		})
	}
	sort.Slice(namedValues, func(i, j int) bool {
		return namedValues[i].Name < namedValues[j].Name
	})
	return namedValues, nil
}

func jsonToYaml(jsonBytes []byte) ([]byte, error) {
	var o any
	if err := yaml.Unmarshal(jsonBytes, &o); err != nil {
		return nil, err
	}
	return yaml.Marshal(o)
}
