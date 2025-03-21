// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

// Webhook for the Organization custom resource.

func SetupOrganizationWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.Organization{},
		webhookFuncs{
			defaultFunc:        DefaultOrganization,
			validateCreateFunc: ValidateCreateOrganization,
			validateUpdateFunc: ValidateUpdateOrganization,
			validateDeleteFunc: ValidateDeleteOrganization,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-organization,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=organizations,verbs=create;update,versions=v1alpha1,name=morganization.kb.io,admissionReviewVersions=v1

func DefaultOrganization(_ context.Context, _ client.Client, o runtime.Object) error {
	org, ok := o.(*greenhousev1alpha1.Organization)
	if !ok {
		return nil
	}
	// Default the displayName to a normalized version of metadata.name.
	if org.Spec.DisplayName == "" {
		normalizedName := strings.ReplaceAll(org.GetName(), "-", " ")
		normalizedName = strings.TrimSpace(normalizedName)
		org.Spec.DisplayName = normalizedName
	}
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-organization,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=organizations,verbs=create;update;delete,versions=v1alpha1,name=vorganization.kb.io,admissionReviewVersions=v1

func ValidateCreateOrganization(_ context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	organization, ok := obj.(*greenhousev1alpha1.Organization)
	if !ok {
		return nil, nil
	}

	allErrs := field.ErrorList{}
	if err := validateMappedOrgAdminIDPGroup(organization); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateAuthConfig(organization); err != nil {
		allErrs = append(allErrs, err)
	}

	return nil, allErrs.ToAggregate()
}

func ValidateUpdateOrganization(_ context.Context, _ client.Client, _, newObj runtime.Object) (admission.Warnings, error) {
	organization, ok := newObj.(*greenhousev1alpha1.Organization)
	if !ok {
		return nil, nil
	}

	allErrs := field.ErrorList{}
	if err := validateMappedOrgAdminIDPGroup(organization); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateAuthConfig(organization); err != nil {
		allErrs = append(allErrs, err)
	}

	return nil, allErrs.ToAggregate()
}

func ValidateDeleteOrganization(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateMappedOrgAdminIDPGroup(organization *greenhousev1alpha1.Organization) *field.Error {
	if organization.Spec.MappedOrgAdminIDPGroup == "" {
		return field.Required(field.NewPath("spec").Child("MappedOrgAdminIDPGroup"),
			"An Organization without spec.MappedOrgAdminIDPGroup is invalid")
	}

	return nil
}

func validateAuthConfig(organization *greenhousev1alpha1.Organization) *field.Error {
	if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
		return nil
	}

	if organization.Spec.Authentication.SecretRef != "" {
		return nil
	}

	if organization.Spec.Authentication.SCIMConfig.BaseURL == "" {
		return field.Required(field.NewPath("spec").Child("Authentication").Child("SCIMConfig").Child("BaseURL"),
			"An Organization without SCIMConfig.BaseURL is invalid")
	}

	if organization.Spec.Authentication.SCIMConfig.AuthType != scim.Basic && organization.Spec.Authentication.SCIMConfig.AuthType != scim.BearerToken {
		return field.Required(field.NewPath("spec").Child("Authentication").Child("SCIMConfig").Child("AuthType"),
			"An Organization without SCIMConfig.AuthType is invalid")
	}

	return nil
}
