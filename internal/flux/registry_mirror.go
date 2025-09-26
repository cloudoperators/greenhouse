// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
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
	BaseDomain string `yaml:"basedomain"`
	SubPath    string `yaml:"subPath"`
}

// GetRegistryMirrorConfig retrieves and parses the registry mirror configuration from the Organization's ConfigMap.
// Registry mirrors redirect image pulls from original registries to mirror registries.
// For example, with config:
//
//	registryMirrors:
//	  ghcr.io:
//	    basedomain: "keppel.eu-de-1.cloud.sap"
//	    subPath: "ccloud-ghcr-io-mirror"
//
// An image like "ghcr.io/cloudoperators/greenhouse:main" becomes "keppel.eu-de-1.cloud.sap/ccloud-ghcr-io-mirror/cloudoperators/greenhouse:main"
func GetRegistryMirrorConfig(ctx context.Context, k8sClient client.Reader, plugin *greenhousev1alpha1.Plugin) (*RegistryMirrorConfig, error) {
	org := &greenhousev1alpha1.Organization{}
	orgName := plugin.Namespace
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: orgName}, org); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("organization %s not found", orgName)
		}
		return nil, fmt.Errorf("failed to get organization %s: %w", orgName, err)
	}

	if org.Spec.ConfigMapRef == "" {
		return nil, nil
	}

	configMap := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Spec.ConfigMapRef, Namespace: orgName}, configMap); err != nil {
		if kerrors.IsNotFound(err) {
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
			return fmt.Errorf("basedomain cannot be empty for registry %s", registry)
		}
		if mirror.SubPath == "" {
			return fmt.Errorf("subPath cannot be empty for registry %s", registry)
		}
	}

	return nil
}

// SetPluginConditionForRegistryMirrorError sets an error condition on the plugin for registry mirror configuration errors.
func SetPluginConditionForRegistryMirrorError(plugin *greenhousev1alpha1.Plugin, err error) {
	condition := greenhousemetav1alpha1.TrueCondition(
		greenhousev1alpha1.HelmReconcileFailedCondition,
		"RegistryMirrorConfigError",
		"Failed to read registry mirror configuration: "+err.Error())
	plugin.SetCondition(condition)
}
