// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
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

	if err := validateMappedOrgAdminIDPGroup(organization); err != nil {
		return nil, err
	}

	return nil, nil
}

func ValidateUpdateOrganization(_ context.Context, _ client.Client, _, newObj runtime.Object) (admission.Warnings, error) {
	organization, ok := newObj.(*greenhousev1alpha1.Organization)
	if !ok {
		return nil, nil
	}

	if err := validateMappedOrgAdminIDPGroup(organization); err != nil {
		return nil, err
	}

	return nil, nil
}

func ValidateDeleteOrganization(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateMappedOrgAdminIDPGroup(organization *greenhousev1alpha1.Organization) error {
	if organization.Spec.MappedOrgAdminIDPGroup == "" {
		return apierrors.NewInvalid(organization.GroupVersionKind().GroupKind(), organization.GetName(), field.ErrorList{
			field.Required(field.NewPath("spec").Child("MappedOrgAdminIDPGroup"),
				"An Organization without spec.MappedOrgAdminIDPGroup is invalid"),
		})
	}

	return nil
}
