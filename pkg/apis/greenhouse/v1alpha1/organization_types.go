// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SCIMAPIAvailableCondition reflects if there is a connection to SCIM API.
	SCIMAPIAvailableCondition ConditionType = "SCIMAPIAvailable"
	// SecretNotFoundReason is set when the secret with credentials to SCIM is not found.
	SecretNotFoundReason ConditionReason = "SecretNotFound"
	// SCIMRequestFailedReason is set when a request to SCIM failed.
	SCIMRequestFailedReason ConditionReason = "SCIMRequestFailed"
	// SCIMConfigNotProvidedReason is set when scim config is not present in spec as it is optional
	SCIMConfigNotProvidedReason ConditionReason = "SCIMConfigNotProvided"

	// NamespaceCreated is set when the namespace for organization is created.
	NamespaceCreated ConditionType = "NamespaceCreated"
	// OrganizationRBACConfigured is set when the RBAC for organization is configured
	OrganizationRBACConfigured ConditionType = "OrganizationRBACConfigured"
	// OrganizationDefaultTeamRolesConfigured is set when default team roles are configured
	OrganizationDefaultTeamRolesConfigured ConditionType = "OrganizationDefaultTeamRolesConfigured"
	// ServiceProxyProvisioned is set when the service proxy is provisioned
	ServiceProxyProvisioned ConditionType = "ServiceProxyProvisioned"
	// OrganizationOICDConfigured is set when the OICD is configured
	OrganizationOICDConfigured ConditionType = "OrganizationOICDConfigured"
	// DefaultConnectorRedirectsConfigured is set when the default connector redirects are appended with redirects URIs from new organizations
	DefaultConnectorRedirectsConfigured ConditionType = "DefaultConnectorRedirectsConfigured"
	// DexReconcileFailed is set when dex reconcile step has failed
	DexReconcileFailed ConditionReason = "DexReconcileFailed"
	// OAuthOICDFailed is set when OAuth reconciler has failed
	OAuthOICDFailed ConditionReason = "OAuthOICDFailed"
	// DefaultConnectorRedirectsFailed is set when the default connector redirects are not updated with new organization redirect URIs
	DefaultConnectorRedirectsFailed ConditionReason = "DefaultConnectorRedirectsFailed"
	// OrganizationAdminTeamConfigured is set when the admin team is configured for organization
	OrganizationAdminTeamConfigured ConditionType = "OrganizationAdminTeamConfigured"
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
type OrganizationStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the Organization.
	StatusConditions `json:"statusConditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=org
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
//+kubebuilder:printcolumn:name="IdP admin group",type="string",JSONPath=".spec.mappedOrgAdminIdPGroup"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`

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

func (o *Organization) GetConditions() StatusConditions {
	return o.Status.StatusConditions
}

func (o *Organization) SetCondition(condition Condition) {
	o.Status.StatusConditions.SetConditions(condition)
}
