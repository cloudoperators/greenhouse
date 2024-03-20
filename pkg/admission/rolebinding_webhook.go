// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
)

// Webhook for the RoleBinding custom resource.

func SetupRoleBindingWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&extensionsgreenhousev1alpha1.RoleBinding{},
		webhookFuncs{
			defaultFunc:        DefaultRoleBinding,
			validateCreateFunc: ValidateCreateRoleBinding,
			validateUpdateFunc: ValidateUpdateRoleBinding,
			validateDeleteFunc: ValidateDeleteRoleBinding,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-extensions-greenhouse-sap-v1alpha1-rolebinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=extensions.greenhouse.sap,resources=rolebindings,verbs=create;update,versions=v1alpha1,name=mrolebinding.kb.io,admissionReviewVersions=v1

func DefaultRoleBinding(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-extensions-greenhouse-sap-v1alpha1-rolebinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=extensions.greenhouse.sap,resources=rolebindings,verbs=create;update,versions=v1alpha1,name=vrolebinding.kb.io,admissionReviewVersions=v1

func ValidateCreateRoleBinding(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	rb, ok := o.(*extensionsgreenhousev1alpha1.RoleBinding)
	if !ok {
		return nil, nil
	}

	var r extensionsgreenhousev1alpha1.Role
	if err := c.Get(ctx, client.ObjectKey{Namespace: rb.Namespace, Name: rb.Spec.RoleRef}, &r); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewInvalid(rb.GroupVersionKind().GroupKind(), rb.Name, field.ErrorList{field.Invalid(field.NewPath("spec", "roleRef"), rb.Spec.RoleRef, "role does not exist")})
		}
		return nil, apierrors.NewInternalError(err)
	}
	return nil, nil
}

func ValidateUpdateRoleBinding(ctx context.Context, c client.Client, old, cur runtime.Object) (admission.Warnings, error) {
	oldRB, ok := old.(*extensionsgreenhousev1alpha1.RoleBinding)
	if !ok {
		return nil, nil
	}
	curRB, ok := cur.(*extensionsgreenhousev1alpha1.RoleBinding)
	if !ok {
		return nil, nil
	}
	switch {
	case hasClusterChanged(oldRB, curRB):
		return nil, apierrors.NewForbidden(
			schema.GroupResource{
				Group:    oldRB.GroupVersionKind().Group,
				Resource: oldRB.Kind,
			}, oldRB.Name, field.Forbidden(field.NewPath("spec", "clusterName"), "cannot be changed"))
	case hasNamespacesChanged(oldRB, curRB):
		return nil, apierrors.NewForbidden(schema.GroupResource{Group: oldRB.GroupVersionKind().Group, Resource: oldRB.Kind}, oldRB.Name, field.Forbidden(field.NewPath("spec", "namespaces"), "cannot be changed"))
	default:
		return nil, nil
	}
}

func ValidateDeleteRoleBinding(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// hasClusterChanged returns true if the clusterSelector in the old and current RoleBinding are different.
func hasClusterChanged(old, cur *extensionsgreenhousev1alpha1.RoleBinding) bool {
	return old.Spec.ClusterName != cur.Spec.ClusterName
}

// hasNamespacesChanged returns true if the namespaces in the old and current RoleBinding are different.
func hasNamespacesChanged(old, cur *extensionsgreenhousev1alpha1.RoleBinding) bool {
	return !reflect.DeepEqual(old.Spec.Namespaces, cur.Spec.Namespaces)
}
