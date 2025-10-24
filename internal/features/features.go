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
	DexFeatureKey      = "dex"
	TemplateFeatureKey = "template"
)

// Getter is an interface for accessing feature flags.
type Getter interface {
	IsTemplateRenderingEnabled(ctx context.Context) bool
	GetDexStorageType(ctx context.Context) *string
}

type Features struct {
	raw      map[string]string
	dex      *dexFeatures      `yaml:"dex"`
	template *templateFeatures `yaml:"template"`
}

type dexFeatures struct {
	Storage string `yaml:"storage"`
}

type templateFeatures struct {
	Rendering bool `yaml:"rendering"`
}

func NewFeatures(ctx context.Context, k8sClient client.Reader, configMapName, namespace string) (Getter, error) {
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

func (f *Features) resolveTemplateFeatures() error {
	// Extract the `template` key from the ConfigMap
	templateRaw, exists := f.raw[TemplateFeatureKey]
	if !exists {
		return errors.New("template feature not found in ConfigMap")
	}

	// Unmarshal the `template` YAML string into the struct
	tmpl := &templateFeatures{}
	err := yaml.Unmarshal([]byte(templateRaw), tmpl)
	if err != nil {
		return err
	}

	f.template = tmpl
	return nil
}

func (f *Features) IsTemplateRenderingEnabled(ctx context.Context) bool {
	if f.template != nil {
		return f.template.Rendering
	}
	if err := f.resolveTemplateFeatures(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to resolve template features")
		// Default to disabled if feature flag is not configured
		return false
	}
	return f.template.Rendering
}
