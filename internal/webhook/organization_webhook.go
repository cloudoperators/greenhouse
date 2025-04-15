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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/scim"
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

	if err := validateSCIMConfig(organization); err != nil {
		allErrs = append(allErrs, err...)
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

	if errs := validateSCIMConfig(organization); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
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

func validateSCIMConfig(org *greenhousev1alpha1.Organization) field.ErrorList {
	scimCfg := org.Spec.GetSCIMConfig()
	if scimCfg == nil {
		return nil
	}

	var errs field.ErrorList
	scimPath := field.NewPath("spec", "authentication", "scim")

	if scimCfg.BaseURL == "" {
		errs = append(errs, field.Required(scimPath.Child("baseURL"), "baseURL is required"))
	}

	switch scimCfg.AuthType {
	case scim.Basic:
		if scimCfg.BasicAuthUser == nil || scimCfg.BasicAuthUser.Secret == nil {
			errs = append(errs, field.Required(scimPath.Child("basicAuthUser"), "basicAuthUser and its secret are required"))
		}
		if scimCfg.BasicAuthPw == nil || scimCfg.BasicAuthPw.Secret == nil {
			errs = append(errs, field.Required(scimPath.Child("basicAuthPw"), "basicAuthPw and its secret are required"))
		}
	case scim.BearerToken:
		if scimCfg.BearerToken == nil || scimCfg.BearerToken.Secret == nil {
			errs = append(errs, field.Required(scimPath.Child("bearerToken"), "bearerToken and its secret are required"))
		}
	default:
		errs = append(errs, field.Invalid(scimPath.Child("authType"), scimCfg.AuthType, "authType must be either 'basic' or 'token'"))
	}

	return errs
}
