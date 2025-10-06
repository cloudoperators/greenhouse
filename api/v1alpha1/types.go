// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
)

// PluginDefinitionReference defines the reference to the PluginDefinition or ClusterPluginDefinition.
type PluginDefinitionReference struct {
	// Name of the referenced PluginDefinition or ClusterPluginDefinition resource.
	// +Optional
	Name string `json:"name,omitempty"`
	// Kind of the referent. Supported values: PluginDefinition, ClusterPluginDefinition.
	// +kubebuilder:default=PluginDefinition
	// +kubebuilder:validation:Enum=PluginDefinition;ClusterPluginDefinition
	Kind string `json:"kind,omitempty"`
}

// HelmChartReference references a Helm Chart in a chart repository.
type HelmChartReference struct {
	// Name of the HelmChart chart.
	Name string `json:"name"`
	// Repository of the HelmChart chart.
	Repository string `json:"repository"`
	// Version of the HelmChart chart.
	Version string `json:"version"`
}

// String returns the printable HelmChartReference.
func (h *HelmChartReference) String() string {
	return fmt.Sprintf("%s/%s:%s", h.Repository, h.Name, h.Version)
}

// ValueFromSource is a valid source for a value.
type ValueFromSource struct {
	// Secret references the secret containing the value.
	Secret *SecretKeyReference `json:"secret,omitempty"`
}

// SecretKeyReference specifies the secret and key containing the value.
type SecretKeyReference struct {
	// Name of the secret in the same namespace.
	Name string `json:"name"`
	// Key in the secret to select the value from.
	Key string `json:"key"`
}

// UIApplicationReference references the UI pluginDefinition to use.
type UIApplicationReference struct {
	// URL specifies the url to a built javascript asset.
	// By default, assets are loaded from the Juno asset server using the provided name and version.
	URL string `json:"url,omitempty"`

	// Name of the UI application.
	Name string `json:"name"`

	// Version of the frontend application.
	Version string `json:"version"`
}
