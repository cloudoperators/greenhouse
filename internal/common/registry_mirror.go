// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const (
	registryMirrorConfigKey = "containerRegistryConfig"
)

// RegistryMirrorConfig represents the registry mirror configuration structure.
type RegistryMirrorConfig struct {
	PrimaryMirror   string                    `yaml:"primaryMirror"`
	RegistryMirrors map[string]RegistryMirror `yaml:"registryMirrors"`
}

// RegistryMirror represents a single registry mirror configuration.
type RegistryMirror struct {
	BaseDomain string `yaml:"baseDomain"`
	SubPath    string `yaml:"subPath"`
}

// GetRegistryMirrorConfig retrieves and parses the registry mirror configuration from the Organization's ConfigMap.
// Registry mirrors redirect image pulls from original registries to mirror registries.
func GetRegistryMirrorConfig(ctx context.Context, k8sClient client.Reader, orgName string) (*RegistryMirrorConfig, error) {
	org := &greenhousev1alpha1.Organization{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: orgName}, org); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("organization %s not found", orgName)
		}
		return nil, fmt.Errorf("failed to get organization %s: %w", orgName, err)
	}

	if org.Spec.ConfigMapRef == "" {
		return nil, nil
	}

	configMap := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Spec.ConfigMapRef, Namespace: orgName}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("organization ConfigMap %s not found in namespace %s", org.Spec.ConfigMapRef, orgName)
		}
		return nil, fmt.Errorf("failed to get organization ConfigMap %s: %w", org.Spec.ConfigMapRef, err)
	}

	registryMirrorData, exists := configMap.Data[registryMirrorConfigKey]
	if !exists {
		return nil, nil
	}

	var mirrorConfig RegistryMirrorConfig
	if err := yaml.Unmarshal([]byte(registryMirrorData), &mirrorConfig); err != nil {
		return nil, fmt.Errorf("failed to parse registry mirror configuration: %w", err)
	}

	if err := validateRegistryMirrorConfig(&mirrorConfig); err != nil {
		return nil, fmt.Errorf("invalid registry mirror configuration: %w", err)
	}

	return &mirrorConfig, nil
}

// validateRegistryMirrorConfig validates the registry mirror configuration.
func validateRegistryMirrorConfig(config *RegistryMirrorConfig) error {
	if len(config.RegistryMirrors) == 0 {
		return errors.New("registryMirrors cannot be empty")
	}

	for registry, mirror := range config.RegistryMirrors {
		if registry == "" {
			return errors.New("registry name cannot be empty")
		}
		if mirror.BaseDomain == "" {
			return fmt.Errorf("baseDomain cannot be empty for registry %s", registry)
		}
		if mirror.SubPath == "" {
			return fmt.Errorf("subPath cannot be empty for registry %s", registry)
		}
	}

	return nil
}
