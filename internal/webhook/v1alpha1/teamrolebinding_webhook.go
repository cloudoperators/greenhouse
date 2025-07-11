// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the RoleBinding custom resource.

func SetupTeamRoleBindingWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.TeamRoleBinding{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultRoleBinding,
			ValidateCreateFunc: ValidateCreateRoleBinding,
			ValidateUpdateFunc: ValidateUpdateRoleBinding,
			ValidateDeleteFunc: ValidateDeleteRoleBinding,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-teamrolebinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamrolebindings,verbs=create;update,versions=v1alpha1,name=mrolebinding-v1alpha1.kb.io,admissionReviewVersions=v1

func DefaultRoleBinding(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-teamrolebinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamrolebindings,verbs=create;update;delete,versions=v1alpha1,name=vrolebinding-v1alpha1.kb.io,admissionReviewVersions=v1

func ValidateCreateRoleBinding(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	rb, ok := o.(*greenhousev1alpha1.TeamRoleBinding)
	if !ok {
		return nil, nil
	}

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

	err := validateClusterNameOrSelector(rb)
	if err != nil {
		return nil, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, rb)
	if labelValidationWarning != "" {
		return admission.Warnings{"TeamRoleBinding should have a support-group Team set as its owner", labelValidationWarning}, nil
	}
	return nil, nil
}

func ValidateUpdateRoleBinding(ctx context.Context, c client.Client, old, cur runtime.Object) (admission.Warnings, error) {
	oldRB, ok := old.(*greenhousev1alpha1.TeamRoleBinding)
	if !ok {
		return nil, nil
	}
	curRB, ok := cur.(*greenhousev1alpha1.TeamRoleBinding)
	if !ok {
		return nil, nil
	}
	switch {
	case validateClusterNameOrSelector(curRB) != nil:
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec"), "must contain either spec.clusterName or spec.clusterSelector"))
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

func ValidateDeleteRoleBinding(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validateClusterNameOrSelector checks if the TeamRoleBinding has a valid clusterName or clusterSelector but not both.
func validateClusterNameOrSelector(rb *greenhousev1alpha1.TeamRoleBinding) error {
	if rb.Spec.ClusterName != "" && (len(rb.Spec.ClusterSelector.MatchLabels) > 0 || len(rb.Spec.ClusterSelector.MatchExpressions) > 0) {
		return apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "clusterName"), rb.Spec.ClusterName, "cannot specify both spec.clusterName and spec.clusterSelector")})
	}

	if rb.Spec.ClusterName == "" && (len(rb.Spec.ClusterSelector.MatchLabels) == 0 && len(rb.Spec.ClusterSelector.MatchExpressions) == 0) {
		return apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "clusterName"), rb.Spec.ClusterName, "must specify either spec.clusterName or spec.clusterSelector")})
	}
	return nil
}

// isClusterScoped returns true if the TeamRoleBinding will create ClusterRoleBindings.
func isClusterScoped(trb *greenhousev1alpha1.TeamRoleBinding) bool {
	return len(trb.Spec.Namespaces) == 0
}
