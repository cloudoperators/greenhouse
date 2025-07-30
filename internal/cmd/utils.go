// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func loadPluginDefinition(path string) (*greenhousev1alpha1.ClusterPluginDefinition, error) {
	var pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	err := loadAndUnmarshalObject(path, &pluginDefinition)
	return pluginDefinition, err
}

func loadPluginPreset(path string) (*greenhousev1alpha1.PluginPreset, error) {
	var pluginPreset *greenhousev1alpha1.PluginPreset
	err := loadAndUnmarshalObject(path, &pluginPreset)
	return pluginPreset, err
}

func loadPlugin(path string) (*greenhousev1alpha1.Plugin, error) {
	var plugin *greenhousev1alpha1.Plugin
	err := loadAndUnmarshalObject(path, &plugin)
	return plugin, err
}

func loadAndUnmarshalObject(path string, o any) error {
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
