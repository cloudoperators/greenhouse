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

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
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

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginpreset,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update,versions=v1alpha1,name=mpluginpreset-v1alpha1.kb.io,admissionReviewVersions=v1

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

	// Migrate the deprecated PluginDefinition reference
	//nolint:staticcheck
	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Name == "" && pluginPreset.Spec.Plugin.PluginDefinition != "" {
		//nolint:staticcheck
		pluginPreset.Spec.Plugin.PluginDefinitionRef.Name = pluginPreset.Spec.Plugin.PluginDefinition
	}

	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind == "" {
		if pluginPreset.Spec.Plugin.PluginDefinitionRef.Name == "" {
			return nil
		}
		if pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace == "" {
			// Check if PluginDefinition exists in PluginPreset's namespace
			pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
			err := c.Get(ctx, types.NamespacedName{Namespace: pluginPreset.Namespace, Name: pluginPreset.Spec.Plugin.PluginDefinitionRef.Name}, pluginDefinition)
			if err == nil {
				pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace = pluginPreset.Namespace
				pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind = "PluginDefinition"
				return nil
			}
			// Check if ClusterPluginDefinition exists
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err = c.Get(ctx, types.NamespacedName{Name: pluginPreset.Spec.Plugin.PluginDefinitionRef.Name}, clusterPluginDefinition)
			if err == nil {
				pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind = "ClusterPluginDefinition"
				return nil
			}
		}
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginpreset,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update;delete,versions=v1alpha1,name=vpluginpreset-v1alpha1.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginPreset(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginPreset, ok := o.(*greenhousev1alpha1.PluginPreset)
	if !ok {
		return nil, nil
	}
	var allErrs field.ErrorList
	var allWarns admission.Warnings

	// Ensure PluginDefinitionRef is set correctly
	if fieldErr := validatePluginDefinitionReference(pluginPreset); fieldErr != nil {
		allErrs = append(allErrs, fieldErr)
	}

	// ensure only one of ClusterName and Cluster selector is set
	if pluginPreset.Spec.ClusterName != "" && pluginPreset.Spec.ClusterSelector.Size() > 0 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "clusterName"),
			pluginPreset.Spec.ClusterName,
			"cannot specify both spec.clusterName and spec.clusterSelector",
		))
	}
	if pluginPreset.Spec.ClusterName == "" && pluginPreset.Spec.ClusterSelector.Size() == 0 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "clusterSelector"),
			pluginPreset.Spec.ClusterSelector,
			"must specify either spec.clusterName or spec.clusterSelector",
		))
	}

	// ensure ClusterName is not set in PluginSpec
	if pluginPreset.Spec.Plugin.ClusterName != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("clusterName"), pluginPreset.Spec.Plugin.ClusterName, "spec.plugin.clusterName must not be set"))
	}

	if err := webhook.ValidateReleaseName(pluginPreset.Spec.Plugin.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("releaseName"), pluginPreset.Spec.Plugin.ReleaseName, err.Error()))
	}

	// Ensure PluginDefinition exists and validate OptionValues
	switch pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace,
			Name:      pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
		}, pluginDefinition)
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s does not exist in namespace %s", pluginPreset.Spec.Plugin.PluginDefinitionRef.Name, pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace)))
		} else if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s could not be retrieved from namespace %s: %s", pluginPreset.Spec.Plugin.PluginDefinitionRef.Name, pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace, err.Error())))
		}
		// validate OptionValues defined by the Preset
		if errList := validatePluginOptionValuesForPreset(pluginPreset, pluginDefinition.Name, pluginDefinition.Spec); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	case "ClusterPluginDefinition":
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: "",
			Name:      pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
		}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s does not exist", pluginPreset.Spec.Plugin.PluginDefinitionRef.Name)))
		} else if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pluginPreset.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s could not be retrieved: %s", pluginPreset.Spec.Plugin.PluginDefinitionRef.Name, err.Error())))
		}
		// validate OptionValues defined by the Preset
		if errList := validatePluginOptionValuesForPreset(pluginPreset, clusterPluginDefinition.Name, clusterPluginDefinition.Spec); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "kind"), pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind"))
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", labelValidationWarning)
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
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

	//nolint:staticcheck
	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.Spec.Plugin.PluginDefinition, field.NewPath("spec", "plugin", "pluginDefinition")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinitionRef.Name, pluginPreset.Spec.Plugin.PluginDefinitionRef.Name, field.NewPath("spec", "plugin", "pluginDefinitionRef", "name")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinitionRef.Kind, pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind, field.NewPath("spec", "plugin", "pluginDefinitionRef", "kind")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace, pluginPreset.Spec.Plugin.PluginDefinitionRef.Namespace, field.NewPath("spec", "plugin", "pluginDefinitionRef", "namespace")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.ClusterName, pluginPreset.Spec.Plugin.ClusterName, field.NewPath("spec", "plugin", "clusterName")); err != nil {
		allErrs = append(allErrs, err)
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", labelValidationWarning)
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
func validatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha1.PluginPreset, pluginDefinitionName string, pluginDefinitionSpec greenhousemetav1alpha1.PluginDefinitionTemplateSpec) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := webhook.ValidatePluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinitionName, pluginDefinitionSpec, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = webhook.ValidatePluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinitionName, pluginDefinitionSpec, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}

func validatePluginDefinitionReference(pp *greenhousev1alpha1.PluginPreset) *field.Error {
	// Require at least one
	//nolint:staticcheck
	if pp.Spec.Plugin.PluginDefinitionRef.Name == "" && pp.Spec.Plugin.PluginDefinition == "" {
		return field.Required(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), "either pluginDefinitionRef or pluginDefinition must be set")
	}

	// If both set, they must match
	//nolint:staticcheck
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" && pp.Spec.Plugin.PluginDefinition != "" &&
		//nolint:staticcheck
		pp.Spec.Plugin.PluginDefinitionRef.Name != pp.Spec.Plugin.PluginDefinition {
		//nolint:staticcheck
		return field.Invalid(field.NewPath("spec", "plugin", "pluginDefinition"), pp.Spec.Plugin.PluginDefinition, "pluginDefinitionRef.name does not match deprecated pluginDefinition")
	}

	// Validate Kind and Namespace
	switch pp.Spec.Plugin.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		if pp.Spec.Plugin.PluginDefinitionRef.Namespace == "" {
			return field.Required(field.NewPath("spec", "plugin", "pluginDefinitionRef", "namespace"), "pluginDefinitionRef.namespace must be set when kind is PluginDefinition")
		}
	case "ClusterPluginDefinition":
		if pp.Spec.Plugin.PluginDefinitionRef.Namespace != "" {
			return field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "namespace"), pp.Spec.Plugin.PluginDefinitionRef.Namespace, "pluginDefinitionRef.namespace must be empty when kind is ClusterPluginDefinition")
		}
	default:
		return field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "kind"), pp.Spec.Plugin.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind")
	}

	return nil
}
