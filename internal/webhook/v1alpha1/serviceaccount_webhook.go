// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the ServiceAccount core resource.

func SetupServiceAccountWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&corev1.ServiceAccount{},
		webhook.WebhookFuncs[*corev1.ServiceAccount]{
			ValidateUpdateFunc: ValidateUpdateServiceAccount,
		},
	)
}

//+kubebuilder:webhook:path=/validate--v1-serviceaccount,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=serviceaccounts,verbs=update,versions=v1,name=vserviceaccount.kb.io,admissionReviewVersions=v1

func ValidateUpdateServiceAccount(_ context.Context, _ client.Client, oldSA, newSA *corev1.ServiceAccount) (admission.Warnings, error) {
	return nil, validateOwnedByLabelImmutable(oldSA, newSA)
}

// validateOwnedByLabelImmutable ensures that the greenhouse.sap/owned-by label cannot be
// changed or removed once set on a ServiceAccount.
func validateOwnedByLabelImmutable(oldSA, newSA *corev1.ServiceAccount) error {
	oldOwner, hadLabel := oldSA.Labels[greenhouseapis.LabelKeyOwnedBy]
	if !hadLabel {
		return nil
	}

	newOwner := newSA.Labels[greenhouseapis.LabelKeyOwnedBy]
	if oldOwner == newOwner {
		return nil
	}

	// Prevents the greenhouse.sap/owned-by label from being changed or removed
	// on support-group ServiceAccounts, to prevent cross-Team privilege escalation.
	return apierrors.NewForbidden(
		corev1.Resource("serviceaccounts"),
		newSA.Name,
		field.Forbidden(
			field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			"label is immutable and cannot be changed or removed once set",
		),
	)
}
