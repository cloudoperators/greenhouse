// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	extensionsgreenhouse "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse"
	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
)

// Webhook for the Role custom resource.

func SetupRoleWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&extensionsgreenhousev1alpha1.Role{},
		webhookFuncs{
			defaultFunc:        DefaultRole,
			validateCreateFunc: ValidateCreateRole,
			validateUpdateFunc: ValidateUpdateRole,
			validateDeleteFunc: ValidateDeleteRole,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-extensions-greenhouse-sap-v1alpha1-role,mutating=true,failurePolicy=fail,sideEffects=None,groups=extensions.greenhouse.sap,resources=roles,verbs=create;update,versions=v1alpha1,name=mrole.kb.io,admissionReviewVersions=v1

func DefaultRole(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-extensions-greenhouse-sap-v1alpha1-role,mutating=false,failurePolicy=fail,sideEffects=None,groups=extensions.greenhouse.sap,resources=roles,verbs=create;update;delete,versions=v1alpha1,name=vrole.kb.io,admissionReviewVersions=v1

func ValidateCreateRole(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func ValidateUpdateRole(_ context.Context, _ client.Client, _, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func ValidateDeleteRole(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	r, ok := o.(*extensionsgreenhousev1alpha1.Role)
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
		}, r.GetName(), fmt.Errorf("role is still referenced by a rolebinding"))
	}
	return nil, nil
}

// isRoleReferenced returns true if there are any rolebindings referencing the given role.
func isRoleReferenced(ctx context.Context, c client.Client, r *extensionsgreenhousev1alpha1.Role) (bool, error) {
	l := &extensionsgreenhousev1alpha1.RoleBindingList{}
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(extensionsgreenhouse.RolebindingRoleRefField, r.GetName()),
		Namespace:     r.GetNamespace(),
	}

	if err := c.List(ctx, l, listOpts); err != nil {
		return false, err
	}
	return len(l.Items) > 0, nil
}
