// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	// DisplayName is an optional name for the organization to be displayed in the Greenhouse UI.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Authentication configures the organizations authentication mechanism.
	Authentication *Authentication `json:"authentication,omitempty"`

	// Description provides additional details of the organization.
	Description string `json:"description,omitempty"`

	// MappedOrgAdminIDPGroup is the IDP group ID identifying org admins
	MappedOrgAdminIDPGroup string `json:"mappedOrgAdminIdPGroup,omitempty"`
}

type Authentication struct {
	// OIDConfig configures the OIDC provider.
	OIDCConfig *OIDCConfig `json:"oidc,omitempty"`
	// SCIMConfig configures the SCIM client.
	SCIMConfig *SCIMConfig `json:"scim,omitempty"`
}

type OIDCConfig struct {
	// Issuer is the URL of the identity service.
	Issuer string `json:"issuer"`
	// RedirectURI is the redirect URI.
	// If none is specified, the Greenhouse ID proxy will be used.
	RedirectURI string `json:"redirectURI,omitempty"`
	// ClientIDReference references the Kubernetes secret containing the client id.
	ClientIDReference SecretKeyReference `json:"clientIDReference"`
	// ClientSecretReference references the Kubernetes secret containing the client secret.
	ClientSecretReference SecretKeyReference `json:"clientSecretReference"`
}

type SCIMConfig struct {
	// URL to the SCIM server.
	BaseURL string `json:"baseURL"`
	// User to be used for basic authentication.
	BasicAuthUser ValueFromSource `json:"basicAuthUser"`
	// Password to be used for basic authentication.
	BasicAuthPw ValueFromSource `json:"basicAuthPw"`
}

// OrganizationStatus defines the observed state of an Organization
type OrganizationStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=org
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
//+kubebuilder:printcolumn:name="IdP admin group",type="string",JSONPath=".spec.mappedOrgAdminIdPGroup"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Organization is the Schema for the organizations API
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
