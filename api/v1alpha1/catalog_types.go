// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"path"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

const (
	CatalogReadyReason             greenhousemetav1alpha1.ConditionReason = "CatalogReady"
	CatalogNotReadyReason          greenhousemetav1alpha1.ConditionReason = "CatalogNotReady"
	CatalogSourceReadyReason       greenhousemetav1alpha1.ConditionReason = "CatalogSourceReady"
	CatalogSourceNotReadyReason    greenhousemetav1alpha1.ConditionReason = "CatalogSourceNotReady"
	CatalogArtifactReadyReason     greenhousemetav1alpha1.ConditionReason = "CatalogArtifactReady"
	CatalogArtifactNotReadyReason  greenhousemetav1alpha1.ConditionReason = "CatalogArtifactNotReady"
	CatalogResourcesReadyReason    greenhousemetav1alpha1.ConditionReason = "CatalogResourcesReady"
	CatalogResourcesNotReadyReason greenhousemetav1alpha1.ConditionReason = "CatalogResourcesNotReady"
	CatalogSecretErrorReason       greenhousemetav1alpha1.ConditionReason = "CatalogSecretErr"
)

const (
	CatalogSourceReadyCondition       greenhousemetav1alpha1.ConditionType = "CatalogSourceReady"
	CatalogArtifactReadyCondition     greenhousemetav1alpha1.ConditionType = "CatalogArtifactReady"
	CatalogResourcesReadyCondition    greenhousemetav1alpha1.ConditionType = "CatalogResourcesReady"
	CatalogSourceSecretErrorCondition greenhousemetav1alpha1.ConditionType = "CatalogSourceSecretErr"
)

// CatalogSpec defines the desired state of Catalog.
type CatalogSpec struct {
	// Source is the medium from which the PluginDefinition needs to be fetched
	Source CatalogSource `json:"source"`
}

type CatalogSource struct {
	// Repository - the Git repository URL
	Repository string `json:"repository"`

	// Resources - list of path to PluginDefinition file
	Resources []string `json:"resources"`

	// Ref - the git reference (branch, tag, or SHA) to resolve PluginDefinitions
	// if not specified, defaults to main branch
	// +Optional
	Ref *GitRef `json:"ref,omitempty"`
	// SecretName is the name of v1.Secret containing credentials to access the Git repository
	// the secret must be in the same namespace as the Catalog resource
	/*
	  GitHub App Example:
	  -------------------
	  githubAppID: "<app-id>"
	  githubAppInstallationID: "<app-installation-id>"
	  githubAppPrivateKey: |
	    -----BEGIN RSA PRIVATE KEY-----
	    ...
	    -----END RSA PRIVATE KEY-----
	  githubAppBaseURL: "<github-enterprise-api-url>" #optional, required only for GitHub Enterprise Server users
	  ca.crt: | #optional, for GitHub Enterprise Server users
	    -----BEGIN CERTIFICATE-----
	    ...
	    -----END CERTIFICATE-----

	  GitHub Token Example:
	  -------------------
	  username: <BASE64>
	  password: <BASE64>
	  ca.crt: <BASE64> #optional, for GitHub Enterprise Server users
	*/
	// +Optional
	SecretName *string `json:"secretName,omitempty"`

	// Overrides are the PluginDefinition overrides to be applied
	// +Optional
	Overrides []CatalogOverrides `json:"overrides,omitempty"`
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
	// +Optional
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
	// +Optional
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

func (c *Catalog) Resources() []string {
	r := make([]string, 0, len(c.Spec.Source.Resources))
	for _, resourcePath := range c.Spec.Source.Resources {
		dir := path.Dir(resourcePath)
		base := path.Base(resourcePath)
		if dir == "." {
			r = append(r, base)
			continue
		}
		// Replace slashes in directory path with a separator to create a flat name.
		flatDir := strings.ReplaceAll(dir, "/", "-")
		r = append(r, flatDir+"-"+base)
	}
	return r
}

func (c *Catalog) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return c.Status.StatusConditions
}

func (c *Catalog) SetCondition(condition greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(condition)
}

func (c *Catalog) SetConditions(conditions ...greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(conditions...)
}

func (c *Catalog) FindCondition(conditionType greenhousemetav1alpha1.ConditionType) *greenhousemetav1alpha1.Condition {
	return c.Status.GetConditionByType(conditionType)
}

func (c *Catalog) SetConditionsUnknown() {
	unknowns := make([]greenhousemetav1alpha1.Condition, 0)
	if cond := c.FindCondition(greenhousemetav1alpha1.ReadyCondition); cond == nil {
		unknowns = append(unknowns, greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, CatalogNotReadyReason, "Catalog reconciliation in progress"))
	}
	if cond := c.FindCondition(CatalogSourceReadyCondition); cond == nil {
		unknowns = append(unknowns, greenhousemetav1alpha1.UnknownCondition(CatalogSourceReadyCondition, CatalogSourceNotReadyReason, "Catalog source reconciliation in progress"))
	}
	if cond := c.FindCondition(CatalogArtifactReadyCondition); cond == nil {
		unknowns = append(unknowns, greenhousemetav1alpha1.UnknownCondition(CatalogArtifactReadyCondition, CatalogArtifactNotReadyReason, "Catalog artifacts reconciliation in progress"))
	}
	if cond := c.FindCondition(CatalogResourcesReadyCondition); cond == nil {
		unknowns = append(unknowns, greenhousemetav1alpha1.UnknownCondition(CatalogResourcesReadyCondition, CatalogResourcesNotReadyReason, "Catalog resources reconciliation in progress"))
	}
	c.SetConditions(unknowns...)
}

func (c *Catalog) SetFalseCondition(conditionType greenhousemetav1alpha1.ConditionType, reason greenhousemetav1alpha1.ConditionReason, message string) {
	c.SetCondition(greenhousemetav1alpha1.FalseCondition(conditionType, reason, message))
}

func (c *Catalog) SetConditionsFalse(message string) {
	c.SetConditions(
		greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, CatalogNotReadyReason, message),
		greenhousemetav1alpha1.FalseCondition(CatalogSourceReadyCondition, CatalogSourceNotReadyReason, message),
		greenhousemetav1alpha1.FalseCondition(CatalogArtifactReadyCondition, CatalogArtifactNotReadyReason, message),
		greenhousemetav1alpha1.FalseCondition(CatalogResourcesReadyCondition, CatalogResourcesNotReadyReason, message),
	)
}

func (c *Catalog) SetTrueCondition(conditionType greenhousemetav1alpha1.ConditionType, reason greenhousemetav1alpha1.ConditionReason, message string) {
	c.SetCondition(greenhousemetav1alpha1.TrueCondition(conditionType, reason, message))
}

func (c *Catalog) RemoveCondition(conditionType greenhousemetav1alpha1.ConditionType) {
	condition := c.FindCondition(conditionType)
	if condition == nil {
		return
	}
	newConditions := slices.DeleteFunc(c.GetConditions().Conditions, func(cond greenhousemetav1alpha1.Condition) bool {
		return cond.Type == conditionType
	})
	c.Status.StatusConditions.Conditions = newConditions
}

func init() {
	SchemeBuilder.Register(&Catalog{}, &CatalogList{})
}
