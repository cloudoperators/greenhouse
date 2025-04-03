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

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/greenhouse/v1alpha1"
)

// Webhook for the PluginPreset custom resource.

const preventDeletionAnnotation = "greenhouse.sap/prevent-deletion"

func SetupPluginPresetWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.PluginPreset{},
		webhookFuncs{
			defaultFunc:        DefaultPluginPreset,
			validateCreateFunc: ValidateCreatePluginPreset,
			validateUpdateFunc: ValidateUpdatePluginPreset,
			validateDeleteFunc: ValidateDeletePluginPreset,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginpreset,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update,versions=v1alpha1,name=mpluginpreset.kb.io,admissionReviewVersions=v1

func DefaultPluginPreset(_ context.Context, _ client.Client, o runtime.Object) error {
	pluginPreset, ok := o.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil
	}

	// prevent deletion on plugin preset creation
	if pluginPreset.Annotations == nil {
		pluginPreset.Annotations = map[string]string{}
	}
	if pluginPreset.CreationTimestamp.IsZero() {
		pluginPreset.Annotations[preventDeletionAnnotation] = "true"
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginpreset,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update;delete,versions=v1alpha1,name=vpluginpreset.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginPreset(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginPreset, ok := o.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}
	var allErrs field.ErrorList

	// ensure PluginDefinition and ClusterSelector are set
	if pluginPreset.Spec.Plugin.PluginDefinition == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinition must be set"))
	}

	if pluginPreset.Spec.ClusterSelector.Size() == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("clusterSelector"), pluginPreset.Spec.ClusterSelector, "ClusterSelector must be set"))
	}

	// ensure ClusterName is not set
	if pluginPreset.Spec.Plugin.ClusterName != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("clusterName"), pluginPreset.Spec.Plugin.ClusterName, "ClusterName must not be set"))
	}

	// ensure PluginDefinition exists
	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginPreset.Spec.Plugin.PluginDefinition}, pluginDefinition)
	switch {
	case err != nil && apierrors.IsNotFound(err):
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, fmt.Sprintf("PluginDefinition %s does not exist", pluginPreset.Spec.Plugin.PluginDefinition)))
	case err != nil:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinition could not be retrieved: "+err.Error()))
	}

	// validate OptionValues defined by the Preset
	if errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return nil, nil
}

func ValidateUpdatePluginPreset(ctx context.Context, c client.Client, oldObj, curObj runtime.Object) (admission.Warnings, error) {
	oldPluginPreset, ok := oldObj.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}
	pluginPreset, ok := curObj.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList

	if err := validateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.Spec.Plugin.PluginDefinition, field.NewPath("spec", "plugin", "pluginDefinition")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableField(oldPluginPreset.Spec.Plugin.ClusterName, pluginPreset.Spec.Plugin.ClusterName, field.NewPath("spec", "plugin", "clusterName")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return nil, nil
}

func ValidateDeletePluginPreset(_ context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	pluginPreset, ok := obj.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList
	if _, ok := pluginPreset.Annotations[preventDeletionAnnotation]; ok {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("annotation").Child(preventDeletionAnnotation),
			pluginPreset.Annotations, fmt.Sprintf("PluginPreset with annotation '%s' set may not be deleted.", preventDeletionAnnotation)))
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return nil, nil
}

// validatePluginOptionValuesForPreset validates plugin options and their values, but skips the check for required options.
// Required options are checked at the Plugin creation level, because the preset can override options and we cannot predict what clusters will be a part of the PluginPreset later on.
func validatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha1.PluginPreset, pluginDefinition *greenhousev1alpha1.PluginDefinition) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := validatePluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinition, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = validatePluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinition, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}
