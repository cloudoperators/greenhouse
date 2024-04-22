// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// Webhook for the PluginBundle custom resource.

func SetupPluginBundleWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.PluginBundle{},
		webhookFuncs{
			defaultFunc:        DefaultPluginBundle,
			validateCreateFunc: ValidateCreatePluginBundle,
			validateUpdateFunc: ValidateUpdatePluginBundle,
			validateDeleteFunc: ValidateDeletePluginBundle,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginbundle,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginbundles,verbs=create;update,versions=v1alpha1,name=mpluginbundle.kb.io,admissionReviewVersions=v1

func DefaultPluginBundle(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginbundle,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginbundles,verbs=create;update,versions=v1alpha1,name=vpluginbundle.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginBundle(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginBundle, ok := o.(*greenhousev1alpha1.PluginBundle)
	if !ok {
		return nil, nil
	}
	var allErrs field.ErrorList

	// ensure PluginDefinition and ClusterSelector are set
	if pluginBundle.Spec.PluginDefinition == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("pluginDefinition"), pluginBundle.Spec.PluginDefinition, "PluginDefinition must be set"))
	}

	if pluginBundle.Spec.ClusterSelector.Size() == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("clusterSelector"), pluginBundle.Spec.ClusterSelector, "ClusterSelector must be set"))
	}

	// ensure PluginDefinition exists
	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginBundle.Spec.PluginDefinition}, pluginDefinition)
	switch {
	case err != nil && apierrors.IsNotFound(err):
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("pluginDefinition"), pluginBundle.Spec.PluginDefinition, fmt.Sprintf("PluginDefinition %s does not exist", pluginBundle.Spec.PluginDefinition)))
	case err != nil:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("pluginDefinition"), pluginBundle.Spec.PluginDefinition, "PluginDefinition could not be retrieved: "+err.Error()))
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginBundle.GroupVersionKind().GroupKind(), pluginBundle.Name, allErrs)
	}

	return nil, nil
}

func ValidateUpdatePluginBundle(ctx context.Context, c client.Client, oldObj, curObj runtime.Object) (admission.Warnings, error) {
	oldPluginBundle, ok := oldObj.(*greenhousev1alpha1.PluginBundle)
	if !ok {
		return nil, nil
	}
	pluginBundle, ok := curObj.(*greenhousev1alpha1.PluginBundle)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList

	if err := validateImmutableField(oldPluginBundle.Spec.PluginDefinition, pluginBundle.Spec.PluginDefinition, field.NewPath("spec", "pluginDefinition")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableField(oldPluginBundle.Spec.ClusterSelector.String(), pluginBundle.Spec.ClusterSelector.String(), field.NewPath("spec", "clusterSelector")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginBundle.GroupVersionKind().GroupKind(), pluginBundle.Name, allErrs)
	}

	return nil, nil
}

func ValidateDeletePluginBundle(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
