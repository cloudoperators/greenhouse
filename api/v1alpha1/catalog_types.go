// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// PluginDefinitionCatalogSpec defines the desired state of PluginDefinitionCatalog.
type PluginDefinitionCatalogSpec struct {
	// Source is the medium from which the PluginDefinition needs to be fetched
	Source CatalogSource `json:"source"`
	// Overrides are the PluginDefinition overrides to be applied
	// +Optional
	Overrides []CatalogOverrides `json:"overrides,omitempty"`
}

type CatalogSource struct {
	// Git is the Git repository source for the PluginDefinition Catalog
	Git GitSource `json:"git"`
	// Path is the path within the repository where the ClusterPluginDefinition / PluginDefinition Catalog is located
	// an empty path indicates the root of the repository
	// +Optional
	Path string `json:"path,omitempty"`
}

type CatalogOverrides struct {
	// Name is the name of the PluginDefinition to patch with an alias
	Name string `json:"name"`
	// Alias is the alias to apply to the PluginDefinition Name via Kustomize patches
	// For SourceType Helm, this field is passed to postRender Kustomize patch
	Alias string `json:"alias"`

	// TODO: implement Options in Overrides for further values patching in PluginDefinition || ClusterPluginDefinition
}

type GitSource struct {
	// Repository is the URL of the GitHub repository containing the ClusterPluginDefinition / PluginDefinition Catalog
	URL string `json:"url"`

	// Ref is the Git reference (branch, tag, or SHA) to resolve the ClusterPluginDefinition / PluginDefinition Catalog
	Ref *GitRef `json:"ref,omitempty"`
}

type GitRef struct {
	Branch *string `json:"branch,omitempty"`
	Tag    *string `json:"tag,omitempty"`
	SHA    *string `json:"sha,omitempty"`
}

// PluginDefinitionCatalogStatus defines the observed state of PluginDefinitionCatalog.
type PluginDefinitionCatalogStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the PluginDefinitionCatalog
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
// +kubebuilder:resource:shortName=pdc

// PluginDefinitionCatalog is the Schema for the plugindefinitioncatalogs API.
type PluginDefinitionCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginDefinitionCatalogSpec   `json:"spec,omitempty"`
	Status PluginDefinitionCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PluginDefinitionCatalogList contains a list of PluginDefinitionCatalog.
type PluginDefinitionCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginDefinitionCatalog `json:"items"`
}

func (p *PluginDefinitionCatalog) ResourcePath() string {
	return p.Spec.Source.Path
}

func (p *PluginDefinitionCatalog) GetCatalogSource() GitSource {
	return p.Spec.Source.Git
}

func (p *PluginDefinitionCatalog) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return p.Status.StatusConditions
}

func (p *PluginDefinitionCatalog) SetCondition(condition greenhousemetav1alpha1.Condition) {
	p.Status.SetConditions(condition)
}

func init() {
	SchemeBuilder.Register(&PluginDefinitionCatalog{}, &PluginDefinitionCatalogList{})
}
