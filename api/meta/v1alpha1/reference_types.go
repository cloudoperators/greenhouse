// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// PluginDefinitionReference defines the reference to the PluginDefinition or ClusterPluginDefinition.
type PluginDefinitionReference struct {
	// Name is the name of the PluginDefinition or ClusterPluginDefinition resource.
	// +required
	Name string `json:"name"`

	// Namespace is the namespace of the PluginDefinition resource.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Kind of the referent.
	// +kubebuilder:validation:Enum=PluginDefinition;ClusterPluginDefinition
	Kind string `json:"kind"`
}

// PluginOptionValue is the value for a PluginOption.
type PluginOptionValue struct {
	// Name of the values.
	Name string `json:"name"`
	// Value is the actual value in plain text.
	Value *apiextensionsv1.JSON `json:"value,omitempty"`
	// ValueFrom references a potentially confidential value in another source.
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
}

// ValueJSON returns the value as JSON.
func (v *PluginOptionValue) ValueJSON() string {
	if v.Value == nil {
		return ""
	}
	return string(v.Value.Raw)
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
