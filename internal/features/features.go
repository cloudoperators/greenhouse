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

var (
	DefaultDeploymentToolValue = "helm"
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
	ExpressionEvaluationEnabled bool   `yaml:"expressionEvaluationEnabled"`
	DefaultDeploymentTool       string `yaml:"defaultDeploymentTool"`
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

func resolve[T any](f *Features, key string) (*T, error) {
	raw, exists := f.raw[key]
	if !exists {
		return nil, errors.New(key + " feature not found in ConfigMap")
	}

	result := new(T)
	if err := yaml.Unmarshal([]byte(raw), result); err != nil {
		return nil, err
	}

	return result, nil
}

func (f *Features) resolveDexFeatures() error {
	dex, err := resolve[dexFeatures](f, DexFeatureKey)
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
	plugin, err := resolve[pluginFeatures](f, PluginFeatureKey)
	if err != nil {
		return err
	}
	f.plugin = plugin
	return nil
}

// IsExpressionEvaluationEnabled returns whether plugin option expression evaluation is enabled.
// Returns false as default.
func (f *Features) IsExpressionEvaluationEnabled() bool {
	if f == nil {
		return false
	}

	if f.plugin != nil {
		return f.plugin.ExpressionEvaluationEnabled
	}
	if err := f.resolvePluginFeatures(); err != nil {
		ctrl.LoggerFrom(context.Background()).Error(err, "failed to resolve plugin features")
		return false
	}
	return f.plugin.ExpressionEvaluationEnabled
}

// GetDefaultDeploymentTool returns the default deployment tool for plugins.
// Returns nil if the value cannot be resolved.
func (f *Features) GetDefaultDeploymentTool() *string {
	if f == nil {
		return nil
	}

	if f.plugin != nil {
		if f.plugin.DefaultDeploymentTool == "" {
			return nil
		}
		return ptr.To(f.plugin.DefaultDeploymentTool)
	}

	if err := f.resolvePluginFeatures(); err != nil {
		ctrl.LoggerFrom(context.Background()).Error(err, "failed to resolve plugin features")
		return nil
	}
	if f.plugin.DefaultDeploymentTool == "" {
		return nil
	}
	return ptr.To(f.plugin.DefaultDeploymentTool)
}
