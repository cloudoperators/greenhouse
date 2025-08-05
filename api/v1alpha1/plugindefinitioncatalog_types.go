// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

const (
	GitRepositoryReady greenhousemetav1alpha1.ConditionType = "GitRepositoryReady"
	KustomizationReady greenhousemetav1alpha1.ConditionType = "KustomizationReady"
	CatalogSuspended   greenhousemetav1alpha1.ConditionType = "CatalogSuspended"
)

const (
	CatalogRepositoryFailReason         greenhousemetav1alpha1.ConditionReason = "CatalogRepositoryFailed"
	CatalogKustomizationBuildFailReason greenhousemetav1alpha1.ConditionReason = "CatalogKustomizationBuildFailed"
	CatalogKustomizationFailReason      greenhousemetav1alpha1.ConditionReason = "CatalogKustomizationFailed"
	CatalogSuspendedReason              greenhousemetav1alpha1.ConditionReason = "CatalogSuspended"
	CatalogReadyReason                  greenhousemetav1alpha1.ConditionReason = "CatalogReady"
	CatalogNotReadyReason               greenhousemetav1alpha1.ConditionReason = "CatalogNotReady"
)

// PluginDefinitionCatalogSpec defines the desired state of PluginDefinitionCatalog.
type PluginDefinitionCatalogSpec struct {
	// Source is the medium from which the PluginDefinition needs to be fetched
	Source CatalogSource `json:"source"`
	// Overrides are the PluginDefinition overrides to be applied
	// +Optional
	Overrides []CatalogOverrides `json:"overrides,omitempty"`
	// Suspend indicates whether the reconciliation of the Catalog is suspended
	// +Optional
	Suspend bool `json:"suspend,omitempty"`
	// Interval is the interval at which the Catalog should be reconciled to check for updates
	// +Optional
	// +kubebuilder:default:="15m"
	Interval *metav1.Duration `json:"interval,omitempty"`
	// Timeout is the timeout for the reconciliation of the Catalog
	// +Optional
	// +kubebuilder:default:="5m"
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

type CatalogSource struct {
	// Git is the Git repository source for the PluginDefinition Catalog
	Git *GitSource `json:"git,omitempty"`
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

	// TODO: implement Options in Overrides as further patching mechanism or helm values
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
	RepositoryArtifact                      *sourcev1.Artifact             `json:"repositoryArtifact,omitempty"`
	KustomizeInventory                      *kustomizev1.ResourceInventory `json:"kustomizeInventory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`

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

func (p *PluginDefinitionCatalog) IsSuspended() bool {
	return p.Spec.Suspend
}

func (p *PluginDefinitionCatalog) Interval() metav1.Duration {
	return *p.Spec.Interval
}

func (p *PluginDefinitionCatalog) Timeout() *metav1.Duration {
	return p.Spec.Timeout
}

func (p *PluginDefinitionCatalog) GetCatalogSource() *GitSource {
	return p.Spec.Source.Git
}

func (p *PluginDefinitionCatalog) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return p.Status.StatusConditions
}

func (p *PluginDefinitionCatalog) SetCondition(condition greenhousemetav1alpha1.Condition) {
	p.Status.SetConditions(condition)
}

// Suspend Conditions

func (p *PluginDefinitionCatalog) SetSuspendedCondition() {
	p.Status.SetConditions(greenhousemetav1alpha1.TrueCondition(CatalogSuspended, "", "Catalog is suspended"))
}

func (p *PluginDefinitionCatalog) UnsetSuspendedCondition() {
	p.Status.SetConditions(greenhousemetav1alpha1.FalseCondition(CatalogSuspended, "", "Catalog is active"))

}

// GitRepository conditions

func (p *PluginDefinitionCatalog) SetGitRepositoryReadyUnknown(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.UnknownCondition(GitRepositoryReady, reason, message))
}

func (p *PluginDefinitionCatalog) SetGitRepositoryReadyFalse(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.FalseCondition(GitRepositoryReady, reason, message))
}

func (p *PluginDefinitionCatalog) SetGitRepositoryReadyTrue(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.TrueCondition(GitRepositoryReady, reason, message))
}

func (p *PluginDefinitionCatalog) IsGitRepositoryReady() bool {
	return p.Status.GetConditionByType(GitRepositoryReady).Status == metav1.ConditionTrue
}

// Kustomization conditions

func (p *PluginDefinitionCatalog) SetKustomizationReadyUnknown(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.UnknownCondition(KustomizationReady, reason, message))
}

func (p *PluginDefinitionCatalog) SetKustomizationReadyFalse(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.FalseCondition(KustomizationReady, reason, message))
}

func (p *PluginDefinitionCatalog) SetKustomizationReadyTrue(reason greenhousemetav1alpha1.ConditionReason, message string) {
	p.Status.SetConditions(greenhousemetav1alpha1.TrueCondition(KustomizationReady, reason, message))
}

func (p *PluginDefinitionCatalog) IsKustomizationReady() bool {
	return p.Status.GetConditionByType(KustomizationReady).Status == metav1.ConditionTrue
}

func init() {
	SchemeBuilder.Register(&PluginDefinitionCatalog{}, &PluginDefinitionCatalogList{})
}
