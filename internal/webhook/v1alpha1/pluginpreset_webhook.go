// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the PluginPreset custom resource.

func SetupPluginPresetWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.PluginPreset{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultPluginPreset,
			ValidateCreateFunc: ValidateCreatePluginPreset,
			ValidateUpdateFunc: ValidateUpdatePluginPreset,
			ValidateDeleteFunc: ValidateDeletePluginPreset,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginpreset,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update,versions=v1alpha1,name=mpluginpreset.kb.io,admissionReviewVersions=v1

func DefaultPluginPreset(ctx context.Context, c client.Client, o runtime.Object) error {
	pluginPreset, ok := o.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil
	}

	// prevent deletion on plugin preset creation
	if pluginPreset.Annotations == nil {
		pluginPreset.Annotations = map[string]string{}
	}
	if pluginPreset.CreationTimestamp.IsZero() {
		pluginPreset.Annotations[greenhousev1alpha1.PreventDeletionAnnotation] = "true"
	}

	// default the PluginDefinitionKind if not set, leave the validation to validating webhook
	if pluginPreset.Spec.Plugin.PluginDefinitionKind == "" && pluginPreset.Spec.Plugin.PluginDefinition != "" {
		// check if PluginDefinition exists in PluginPreset's namespace
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{Namespace: pluginPreset.GetNamespace(), Name: pluginPreset.Spec.Plugin.PluginDefinition}, pluginDefinition)
		if err == nil {
			pluginPreset.Spec.Plugin.PluginDefinitionKind = greenhousev1alpha1.PluginDefinitionKind
		} else {
			// check if ClusterPluginDefinition exists
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err = c.Get(ctx, types.NamespacedName{Name: pluginPreset.Spec.Plugin.PluginDefinition}, clusterPluginDefinition)
			if err == nil {
				pluginPreset.Spec.Plugin.PluginDefinitionKind = greenhousev1alpha1.ClusterPluginDefinitionKind
			}
		}
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
	var allWarns admission.Warnings

	if warn := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset); warn != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", warn)
	}

	// ensure PluginDefinition reference is set
	if pluginPreset.Spec.Plugin.PluginDefinition == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinition must be set"))
	}
	if pluginPreset.Spec.Plugin.PluginDefinitionKind == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinitionKind"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinitionKind must be set"))
	}

	// ensure ClusterSelector is set
	if pluginPreset.Spec.ClusterSelector.Size() == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("clusterSelector"), pluginPreset.Spec.ClusterSelector, "ClusterSelector must be set"))
	}

	// ensure ClusterName is not set
	if pluginPreset.Spec.Plugin.ClusterName != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("clusterName"), pluginPreset.Spec.Plugin.ClusterName, "ClusterName must not be set"))
	}

	if err := validateReleaseName(pluginPreset.Spec.Plugin.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("releaseName"), pluginPreset.Spec.Plugin.ReleaseName, err.Error()))
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	var pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec

	// ensure (Cluster-)PluginDefinition exists and validate OptionValues defined by the Preset
	switch pluginPreset.Spec.Plugin.PluginDefinitionKind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: pluginPreset.GetNamespace(),
			Name:      pluginPreset.Spec.Plugin.PluginDefinition,
		}, pluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return allWarns, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition,
				fmt.Sprintf("PluginDefinition %s does not exist in namespace %s", pluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.GetNamespace()))
		case err != nil:
			return allWarns, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition,
				fmt.Sprintf("PluginDefinition %s could not be retrieved from namespace %s: %s", pluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.GetNamespace(), err.Error()))
		default:
			pluginDefinitionSpec = pluginDefinition.Spec
		}
	case greenhousev1alpha1.ClusterPluginDefinitionKind:
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: "",
			Name:      pluginPreset.Spec.Plugin.PluginDefinition,
		}, clusterPluginDefinition)
		switch {
		case apierrors.IsNotFound(err):
			return allWarns, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition,
				fmt.Sprintf("ClusterPluginDefinition %s does not exist", pluginPreset.Spec.Plugin.PluginDefinition))
		case err != nil:
			return allWarns, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition,
				fmt.Sprintf("ClusterPluginDefinition %s could not be retrieved: %s", pluginPreset.Spec.Plugin.PluginDefinition, err.Error()))
		default:
			pluginDefinitionSpec = clusterPluginDefinition.Spec
		}
	default:
		return allWarns, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionKind"), pluginPreset.Spec.Plugin.PluginDefinitionKind, "unsupported PluginDefinitionKind")
	}

	// validate OptionValues defined by the Preset
	if errList := validatePluginOptionValuesForPreset(pluginPreset, pluginPreset.Spec.Plugin.PluginDefinition, pluginDefinitionSpec); len(errList) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, errList)
	}

	return allWarns, nil
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
	var allWarns admission.Warnings

	if warn := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset); warn != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", warn)
	}

	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.Spec.Plugin.PluginDefinition, field.NewPath("spec", "plugin", "pluginDefinition")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinitionKind, pluginPreset.Spec.Plugin.PluginDefinitionKind, field.NewPath("spec", "plugin", "pluginDefinitionKind")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.ClusterName, pluginPreset.Spec.Plugin.ClusterName, field.NewPath("spec", "plugin", "clusterName")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}
	return allWarns, nil
}

func ValidateDeletePluginPreset(_ context.Context, _ client.Client, obj runtime.Object) (admission.Warnings, error) {
	pluginPreset, ok := obj.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList
	if _, ok := pluginPreset.Annotations[greenhousev1alpha1.PreventDeletionAnnotation]; ok {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("annotation").Child(greenhousev1alpha1.PreventDeletionAnnotation),
			pluginPreset.Annotations, fmt.Sprintf("PluginPreset with annotation '%s' set may not be deleted.", greenhousev1alpha1.PreventDeletionAnnotation)))
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}
	return nil, nil
}

// validatePluginOptionValuesForPreset validates plugin options and their values, but skips the check for required options.
// Required options are checked at the Plugin creation level, because the preset can override options and we cannot predict what clusters will be a part of the PluginPreset later on.
func validatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha1.PluginPreset, pluginDefinitionName string, pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := validatePluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinitionName, pluginDefinitionSpec, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = validatePluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinitionName, pluginDefinitionSpec, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}
