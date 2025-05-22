// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

const errAggregationRuleAndRulesExclusive = ".spec.rules and .spec.aggregationRule are mutually exclusive"

// Webhook for the Role custom resource.

func SetupTeamRoleWebhookWithManager(mgr ctrl.Manager) error {
	// index RoleBindings by the TeamRoleRef field for faster lookups
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha2.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha2.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRoleRef}
	}); clientutil.IgnoreIndexerConflict(err) != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &greenhousev1alpha1.TeamRoleBinding{}, greenhouseapis.RolebindingTeamRoleRefField, func(rawObj client.Object) []string {
		// Extract the TeamRole name from the TeamRoleBinding Spec, if one is provided
		teamRoleBinding, ok := rawObj.(*greenhousev1alpha1.TeamRoleBinding)
		if teamRoleBinding.Spec.TeamRoleRef == "" || !ok {
			return nil
		}
		return []string{teamRoleBinding.Spec.TeamRoleRef}
	}); clientutil.IgnoreIndexerConflict(err) != nil {
		return err
	}

	return setupWebhook(mgr,
		&greenhousev1alpha1.TeamRole{},
		webhookFuncs{
			defaultFunc:        DefaultRole,
			validateCreateFunc: ValidateCreateRole,
			validateUpdateFunc: ValidateUpdateRole,
			validateDeleteFunc: ValidateDeleteRole,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-teamrole,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamroles,verbs=create;update,versions=v1alpha1,name=mteamrole-v1alpha1.kb.io,admissionReviewVersions=v1

func DefaultRole(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-teamrole,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=teamroles,verbs=create;update;delete,versions=v1alpha1,name=vteamrole-v1alpha1.kb.io,admissionReviewVersions=v1

func ValidateCreateRole(_ context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	role, ok := o.(*greenhousev1alpha1.TeamRole)
	if !ok {
		return nil, nil
	}

	if err := isRulesAndAggregationRuleExclusive(role); err != nil {
		return nil, err
	}

	return nil, nil
}

func ValidateUpdateRole(_ context.Context, c client.Client, _, o runtime.Object) (admission.Warnings, error) {
	role, ok := o.(*greenhousev1alpha1.TeamRole)
	if !ok {
		return nil, nil
	}

	if err := isRulesAndAggregationRuleExclusive(role); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateDeleteRole(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	r, ok := o.(*greenhousev1alpha1.TeamRole)
	if !ok {
		return nil, nil
	}

	isReferenced, err := isRoleReferenced(ctx, c, r)
	if err != nil {
		return nil, err
	}
	if isReferenced {
		return nil, apierrors.NewForbidden(schema.GroupResource{
			Group:    r.GroupVersionKind().Group,
			Resource: r.GroupVersionKind().Kind,
		}, r.GetName(), errors.New("role is still referenced by a rolebinding"))
	}
	return nil, nil
}

// isRoleReferenced returns true if there are any rolebindings referencing the given role.
func isRoleReferenced(ctx context.Context, c client.Client, r *greenhousev1alpha1.TeamRole) (bool, error) {
	l := &greenhousev1alpha2.TeamRoleBindingList{}
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(greenhouseapis.RolebindingTeamRoleRefField, r.GetName()),
		Namespace:     r.GetNamespace(),
	}

	if err := c.List(ctx, l, listOpts); err != nil {
		return false, err
	}
	return len(l.Items) > 0, nil
}

// isRulesAndAggregationRuleExclusive checks if Rules and AggregationRule are not both specified.
// Rules will be overwritten on the remote cluster if the AggregationRule is set as well.
// Returning the error in case both are defined will prevent unexpected behavior by the User.
func isRulesAndAggregationRuleExclusive(role *greenhousev1alpha1.TeamRole) error {
	if len(role.Spec.Rules) != 0 && role.Spec.AggregationRule != nil {
		return apierrors.NewBadRequest(errAggregationRuleAndRulesExclusive)
	}
	return nil
}
