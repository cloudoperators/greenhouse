// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package features

import (
	"context"
	"errors"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DexFeatureKey    = "dex"
	PluginFeatureKey = "plugin"
)

type Features struct {
	raw    map[string]string
	dex    *dexFeatures    `yaml:"dex"`
	plugin *pluginFeatures `yaml:"plugin"`
}

type dexFeatures struct {
	Storage string `yaml:"storage"`
}

type pluginFeatures struct {
	OptionValueTemplating bool `yaml:"optionValueTemplating"`
}

func NewFeatures(ctx context.Context, k8sClient client.Reader, configMapName, namespace string) (*Features, error) {
	featureMap := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, featureMap); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &Features{
		raw: featureMap.Data,
	}, nil
}

func (f *Features) resolveDexFeatures() error {
	// Extract the `dex` key from the ConfigMap
	dexRaw, exists := f.raw[DexFeatureKey]
	if !exists {
		return errors.New("dex feature not found in ConfigMap")
	}

	// Unmarshal the `dex` YAML string into the struct
	dex := &dexFeatures{}
	err := yaml.Unmarshal([]byte(dexRaw), dex)
	if err != nil {
		return err
	}

	f.dex = dex
	return nil
}

func (f *Features) GetDexStorageType(ctx context.Context) *string {
	if f.dex != nil {
		return ptr.To(f.dex.Storage)
	}
	if err := f.resolveDexFeatures(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to resolve dex features")
		return nil
	}
	if f.dex.Storage == "" {
		return nil
	}
	return ptr.To(f.dex.Storage)
}

func (f *Features) resolvePluginFeatures() error {
	// Extract the `plugin` key from the ConfigMap
	pluginRaw, exists := f.raw[PluginFeatureKey]
	if !exists {
		return errors.New("plugin feature not found in ConfigMap")
	}

	// Unmarshal the `plugin` YAML string into the struct
	plugin := &pluginFeatures{}
	err := yaml.Unmarshal([]byte(pluginRaw), plugin)
	if err != nil {
		return err
	}

	f.plugin = plugin
	return nil
}

func (f *Features) IsTemplateRenderingEnabled(ctx context.Context) bool {
	if f.plugin != nil {
		return f.plugin.OptionValueTemplating
	}
	if err := f.resolvePluginFeatures(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to resolve plugin features")
		// Default to disabled if feature flag is not configured
		return false
	}
	return f.plugin.OptionValueTemplating
}
