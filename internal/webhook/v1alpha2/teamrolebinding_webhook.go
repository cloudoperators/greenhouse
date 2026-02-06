// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the RoleBinding custom resource.

func SetupTeamRoleBindingWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha2.TeamRoleBinding{},
		webhook.WebhookFuncs[*greenhousev1alpha2.TeamRoleBinding]{
			DefaultFunc:        DefaultRoleBinding,
			ValidateCreateFunc: ValidateCreateRoleBinding,
			ValidateUpdateFunc: ValidateUpdateRoleBinding,
			ValidateDeleteFunc: ValidateDeleteRoleBinding,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha2-teamrolebinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamrolebindings,verbs=create;update,versions=v1alpha2,name=mrolebinding-v1alpha2.kb.io,admissionReviewVersions=v1

func DefaultRoleBinding(_ context.Context, _ client.Client, _ *greenhousev1alpha2.TeamRoleBinding) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha2-teamrolebinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamrolebindings,verbs=create;update;delete,versions=v1alpha2,name=vrolebinding-v1alpha2.kb.io,admissionReviewVersions=v1

func ValidateCreateRoleBinding(ctx context.Context, c client.Client, rb *greenhousev1alpha2.TeamRoleBinding) (admission.Warnings, error) {
	// check if the referenced role exists
	var r greenhousev1alpha1.TeamRole
	if err := c.Get(ctx, client.ObjectKey{Namespace: rb.Namespace, Name: rb.Spec.TeamRoleRef}, &r); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "roleRef"), rb.Spec.TeamRoleRef, "role does not exist")})
		}
		return nil, apierrors.NewInternalError(err)
	}

	// check if the referenced team exists
	var t greenhousev1alpha1.Team
	if err := c.Get(ctx, client.ObjectKey{Namespace: rb.Namespace, Name: rb.Spec.TeamRef}, &t); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "teamRef"), rb.Spec.TeamRef, "team does not exist")})
		}
		return nil, apierrors.NewInternalError(err)
	}

	if err := validateClusterSelector(rb); err != nil {
		return nil, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, rb)
	if labelValidationWarning != "" {
		return admission.Warnings{"TeamRoleBinding should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}

func ValidateUpdateRoleBinding(ctx context.Context, c client.Client, oldRB, curRB *greenhousev1alpha2.TeamRoleBinding) (admission.Warnings, error) {
	switch {
	case validateClusterSelector(curRB) != nil:
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec"), "must contain either spec.clusterSelector.name or spec.clusterSelector.labelSelector"))
	case oldRB.Spec.TeamRoleRef != curRB.Spec.TeamRoleRef:
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec", "teamRoleRef"), "cannot change TeamRoleRef of an existing TeamRoleBinding"))
	case oldRB.Spec.TeamRef != curRB.Spec.TeamRef:
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec", "teamRef"), "cannot change TeamRef of an existing TeamRoleBinding"))
	case isClusterScoped(oldRB) && !isClusterScoped(curRB):
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec", "namespaces"), "cannot change existing TeamRoleBinding from cluster-scoped to namespace-scoped by adding namespaces"))
	case !isClusterScoped(oldRB) && isClusterScoped(curRB):
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec", "namespaces"), "cannot remove all namespaces in existing TeamRoleBinding"))
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, curRB)
	if labelValidationWarning != "" {
		return admission.Warnings{"TeamRoleBinding should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}

func ValidateDeleteRoleBinding(_ context.Context, _ client.Client, _ *greenhousev1alpha2.TeamRoleBinding) (admission.Warnings, error) {
	return nil, nil
}

// validateClusterSelector checks if the TeamRoleBinding has a valid clusterSelector.
func validateClusterSelector(rb *greenhousev1alpha2.TeamRoleBinding) error {
	if rb.Spec.ClusterSelector.Name != "" && (len(rb.Spec.ClusterSelector.LabelSelector.MatchLabels) > 0 || len(rb.Spec.ClusterSelector.LabelSelector.MatchExpressions) > 0) {
		return apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "clusterSelector", "name"), rb.Spec.ClusterSelector.Name, "cannot specify both spec.clusterSelector.Name and spec.clusterSelector.labelSelector")})
	}

	if rb.Spec.ClusterSelector.Name == "" && (len(rb.Spec.ClusterSelector.LabelSelector.MatchLabels) == 0 && len(rb.Spec.ClusterSelector.LabelSelector.MatchExpressions) == 0) {
		return apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "clusterSelector", "name"), rb.Spec.ClusterSelector.Name, "must specify either spec.clusterSelector.name or spec.clusterSelector.labelSelector")})
	}
	return nil
}

// isClusterScoped returns true if the TeamRoleBinding will create ClusterRoleBindings.
func isClusterScoped(trb *greenhousev1alpha2.TeamRoleBinding) bool {
	return len(trb.Spec.Namespaces) == 0
}
