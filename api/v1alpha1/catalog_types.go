// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// CatalogSpec defines the desired state of Catalog.
type CatalogSpec struct {
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
	// Repository is the repository to override in the PluginDefinition .spec.helmChart.repository
	// +Optional
	Repository string `json:"repository,omitempty"`

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

// CatalogStatus defines the observed state of Catalog.
type CatalogStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the Catalog
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
// +kubebuilder:resource:shortName=cat

// Catalog is the Schema for the catalogs API.
type Catalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CatalogSpec   `json:"spec,omitempty"`
	Status CatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CatalogList contains a list of Catalog.
type CatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Catalog `json:"items"`
}

func (c *Catalog) ResourcePath() string {
	return c.Spec.Source.Path
}

func (c *Catalog) GetCatalogSource() GitSource {
	return c.Spec.Source.Git
}

func (c *Catalog) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return c.Status.StatusConditions
}

func (c *Catalog) SetCondition(condition greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(condition)
}

func init() {
	SchemeBuilder.Register(&Catalog{}, &CatalogList{})
}
