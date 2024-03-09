// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
