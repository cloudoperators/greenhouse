// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// WithMappedAdminIDPGroup sets the MappedIDPGroup on an Organization
func WithMappedAdminIDPGroup(group string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		org.Spec.MappedOrgAdminIDPGroup = group
	}
}

func WithOrgAnnotations(annotations map[string]string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		org.SetAnnotations(annotations)
	}
}

// WithAdditionalRedirects - sets the additional redirect URIs on an Organization. (To be used with WithOIDCConfig)
func WithAdditionalRedirects(additionalRedirects ...string) func(organization *greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		if org.Spec.Authentication == nil {
			org.Spec.Authentication = &greenhousev1alpha1.Authentication{}
		}
		if org.Spec.Authentication.OIDCConfig == nil {
			org.Spec.Authentication.OIDCConfig = &greenhousev1alpha1.OIDCConfig{}
		}
		org.Spec.Authentication.OIDCConfig.OAuth2ClientRedirectURIs = additionalRedirects
	}
}

// WithOIDCConfig sets the OIDCConfig on an Organization
func WithOIDCConfig(issuer, secretName, clientIDKey, clientSecretKey string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		if org.Spec.Authentication == nil {
			org.Spec.Authentication = &greenhousev1alpha1.Authentication{}
		}
		org.Spec.Authentication.OIDCConfig = &greenhousev1alpha1.OIDCConfig{
			Issuer: issuer,
			ClientIDReference: greenhousev1alpha1.SecretKeyReference{
				Name: secretName,
				Key:  clientIDKey,
			},
			ClientSecretReference: greenhousev1alpha1.SecretKeyReference{
				Name: secretName,
				Key:  clientSecretKey,
			},
			RedirectURI: issuer + "/callback",
		}
	}
}

// NewOrganization returns a greenhousev1alpha1.Organization object. Opts can be used to set the desired state of the Organization.
func NewOrganization(name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	org := &greenhousev1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Organization",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: greenhousev1alpha1.OrganizationSpec{
			MappedOrgAdminIDPGroup: "default-admin-id-group",
		},
	}
	for _, o := range opts {
		o(org)
	}
	return org
}
