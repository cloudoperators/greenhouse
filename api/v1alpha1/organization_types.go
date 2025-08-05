// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/scim"
)

const (
	// SCIMAPIAvailableCondition reflects if there is a connection to SCIM API.
	SCIMAPIAvailableCondition greenhousemetav1alpha1.ConditionType = "SCIMAPIAvailable"
	// SecretNotFoundReason is set when the secret with credentials to SCIM is not found.
	SecretNotFoundReason greenhousemetav1alpha1.ConditionReason = "SecretNotFound"
	// SCIMRequestFailedReason is set when a request to SCIM failed.
	SCIMRequestFailedReason greenhousemetav1alpha1.ConditionReason = "SCIMRequestFailed"
	// SCIMConfigErrorReason is set when scim config is not present in spec as it is optional
	SCIMConfigErrorReason greenhousemetav1alpha1.ConditionReason = "SCIMConfigErrorReason"

	// NamespaceCreated is set when the namespace for organization is created.
	NamespaceCreated greenhousemetav1alpha1.ConditionType = "NamespaceCreated"
	// OrganizationRBACConfigured is set when the RBAC for organization is configured
	OrganizationRBACConfigured greenhousemetav1alpha1.ConditionType = "OrganizationRBACConfigured"
	// OrganizationDefaultTeamRolesConfigured is set when default team roles are configured
	OrganizationDefaultTeamRolesConfigured greenhousemetav1alpha1.ConditionType = "OrganizationDefaultTeamRolesConfigured"
	// ServiceProxyProvisioned is set when the service proxy is provisioned
	ServiceProxyProvisioned greenhousemetav1alpha1.ConditionType   = "ServiceProxyProvisioned"
	ServiceProxyNotFound    greenhousemetav1alpha1.ConditionReason = "ServiceProxyNotFound"
	ServiceProxyFailed      greenhousemetav1alpha1.ConditionReason = "ServiceProxyFailed"
	// OrganizationOICDConfigured is set when the OICD is configured
	OrganizationOICDConfigured greenhousemetav1alpha1.ConditionType = "OrganizationOICDConfigured"
	// DexReconcileFailed is set when dex reconcile step has failed
	DexReconcileFailed greenhousemetav1alpha1.ConditionReason = "DexReconcileFailed"
	// OAuthOICDFailed is set when OAuth reconciler has failed
	OAuthOICDFailed greenhousemetav1alpha1.ConditionReason = "OAuthOICDFailed"
	// DefaultConnectorRedirectsFailed is set when the default connector redirects are not updated with new organization redirect URIs
	DefaultConnectorRedirectsFailed greenhousemetav1alpha1.ConditionReason = "DefaultConnectorRedirectsFailed"
	// OrganizationAdminTeamConfigured is set when the admin team is configured for organization
	OrganizationAdminTeamConfigured greenhousemetav1alpha1.ConditionType = "OrganizationAdminTeamConfigured"
	// OAuthProxySecretFailed is set when the secret for OAuth proxy is not successfully created
	OAuthProxySecretFailed greenhousemetav1alpha1.ConditionReason = "OAuthProxySecretFailed"
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

	// ConfigMapRef allows to reference organizational config map.
	ConfigMapRef string `json:"configMapRef,omitempty"`
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
	// RedirectURI is the redirect URI to be used for the OIDC flow against the upstream IdP.
	// If none is specified, the Greenhouse ID proxy will be used.
	RedirectURI string `json:"redirectURI,omitempty"`
	// ClientIDReference references the Kubernetes secret containing the client id.
	ClientIDReference SecretKeyReference `json:"clientIDReference"`
	// ClientSecretReference references the Kubernetes secret containing the client secret.
	ClientSecretReference SecretKeyReference `json:"clientSecretReference"`
	// OAuth2ClientRedirectURIs are a registered set of redirect URIs. When redirecting from the idproxy to
	// the client application, the URI requested to redirect to must be contained in this list.
	OAuth2ClientRedirectURIs []string `json:"oauth2ClientRedirectURIs,omitempty"`
}

type SCIMConfig struct {
	// URL to the SCIM server.
	BaseURL string `json:"baseURL"`
	// AuthType defined possible authentication type
	// +kubebuilder:validation:Enum=basic;token
	// +kubebuilder:default="basic"
	AuthType scim.AuthType `json:"authType,omitempty"`
	// User to be used for basic authentication.
	BasicAuthUser *ValueFromSource `json:"basicAuthUser,omitempty"`
	// Password to be used for basic authentication.
	BasicAuthPw *ValueFromSource `json:"basicAuthPw,omitempty"`
	// BearerToken to be used for bearer token authorization
	BearerToken *ValueFromSource `json:"bearerToken,omitempty"`
	// BearerPrefix to be used to defined bearer token prefix
	BearerPrefix string `json:"bearerPrefix,omitempty"`
	// BearerHeader to be used to defined bearer token header
	BearerHeader string `json:"bearerHeader,omitempty"`
}

// OrganizationStatus defines the observed state of an Organization
type OrganizationStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the Organization.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
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

func (o *Organization) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return o.Status.StatusConditions
}

func (o *Organization) SetCondition(condition greenhousemetav1alpha1.Condition) {
	o.Status.SetConditions(condition)
}

func (o *OrganizationSpec) GetSCIMConfig() *SCIMConfig {
	if o.Authentication != nil {
		return o.Authentication.SCIMConfig
	}
	return nil
}
