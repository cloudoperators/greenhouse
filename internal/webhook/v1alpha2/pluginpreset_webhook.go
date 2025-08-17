// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

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
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the PluginPreset custom resource.

func SetupPluginPresetWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha2.PluginPreset{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultPluginPreset,
			ValidateCreateFunc: ValidateCreatePluginPreset,
			ValidateUpdateFunc: ValidateUpdatePluginPreset,
			ValidateDeleteFunc: ValidateDeletePluginPreset,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha2-pluginpreset,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update,versions=v1alpha2,name=mpluginpreset-v1alpha2.kb.io,admissionReviewVersions=v1

func DefaultPluginPreset(ctx context.Context, c client.Client, o runtime.Object) error {
	pluginPreset, ok := o.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil
	}

	// prevent deletion on plugin preset creation
	if pluginPreset.Annotations == nil {
		pluginPreset.Annotations = map[string]string{}
	}
	if pluginPreset.CreationTimestamp.IsZero() {
		pluginPreset.Annotations[greenhousev1alpha2.PreventDeletionAnnotation] = "true"
	}

	// Migrate the deprecated PluginDefinition reference
	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Name == "" && pluginPreset.Spec.Plugin.PluginDefinition != "" {
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

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha2-pluginpreset,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update;delete,versions=v1alpha2,name=vpluginpreset-v1alpha2.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginPreset(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pp, ok := o.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil, nil
	}
	var allErrs field.ErrorList
	var allWarns admission.Warnings

	// Ensure PluginDefinitionRef is set correctly
	if fieldErr := validatePluginDefinitionReference(pp); fieldErr != nil {
		allErrs = append(allErrs, fieldErr)
	}

	// Ensure ClusterSelector is set
	if err := validateClusterSelector(pp.Spec.ClusterSelector, pp.GroupVersionKind().GroupKind()); err != nil {
		return nil, err
	}

	// Ensure ClusterName is not set
	if pp.Spec.Plugin.ClusterName != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("clusterName"), pp.Spec.Plugin.ClusterName, "spec.plugin.clusterName must not be set"))
	}

	if err := webhook.ValidateReleaseName(pp.Spec.Plugin.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("releaseName"), pp.Spec.Plugin.ReleaseName, err.Error()))
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, pp)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", labelValidationWarning)
	}

	// ensure PluginDefinition exists and validate OptionValues
	switch pp.Spec.Plugin.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: pp.Spec.Plugin.PluginDefinitionRef.Namespace,
			Name:      pp.Spec.Plugin.PluginDefinitionRef.Name,
		}, pluginDefinition)
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pp.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s does not exist in namespace %s", pp.Spec.Plugin.PluginDefinitionRef.Name, pp.Spec.Plugin.PluginDefinitionRef.Namespace)))
		} else if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pp.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s could not be retrieved from namespace %s: %s", pp.Spec.Plugin.PluginDefinitionRef.Name, pp.Spec.Plugin.PluginDefinitionRef.Namespace, err.Error())))
		}
		// validate OptionValues defined by the Preset
		if errList := ValidatePluginOptionValuesForPreset(pp, pluginDefinition.Name, pluginDefinition.Spec.Options); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	case "ClusterPluginDefinition":
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: "",
			Name:      pp.Spec.Plugin.PluginDefinitionRef.Name,
		}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pp.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s does not exist", pp.Spec.Plugin.PluginDefinitionRef.Name)))
		} else if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), pp.Spec.Plugin.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s could not be retrieved: %s", pp.Spec.Plugin.PluginDefinitionRef.Name, err.Error())))
		}
		// validate OptionValues defined by the Preset
		if errList := ValidatePluginOptionValuesForPreset(pp, clusterPluginDefinition.Name, clusterPluginDefinition.Spec.Options); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "plugin", "pluginDefinitionRef", "kind"), pp.Spec.Plugin.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind"))
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pp.GroupVersionKind().GroupKind(), pp.Name, allErrs)
	}

	return allWarns, nil
}

func validatePluginDefinitionReference(pp *greenhousev1alpha2.PluginPreset) *field.Error {
	// Require at least one
	if pp.Spec.Plugin.PluginDefinitionRef.Name == "" && pp.Spec.Plugin.PluginDefinition == "" {
		return field.Required(field.NewPath("spec", "plugin", "pluginDefinitionRef", "name"), "either pluginDefinitionRef or pluginDefinition must be set")
	}

	// If both set, they must match
	if pp.Spec.Plugin.PluginDefinitionRef.Name != "" && pp.Spec.Plugin.PluginDefinition != "" &&
		pp.Spec.Plugin.PluginDefinitionRef.Name != pp.Spec.Plugin.PluginDefinition {
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

func ValidateUpdatePluginPreset(ctx context.Context, c client.Client, oldObj, curObj runtime.Object) (admission.Warnings, error) {
	oldPluginPreset, ok := oldObj.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil, nil
	}
	pluginPreset, ok := curObj.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList
	var allWarns admission.Warnings

	if err := webhook.ValidateImmutableField(oldPluginPreset.Spec.Plugin.PluginDefinition, pluginPreset.Spec.Plugin.PluginDefinition, field.NewPath("spec", "plugin", "pluginDefinition")); err != nil {
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
	pluginPreset, ok := obj.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil, nil
	}

	var allErrs field.ErrorList
	if _, ok := pluginPreset.Annotations[greenhousev1alpha2.PreventDeletionAnnotation]; ok {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("annotation").Child(greenhousev1alpha2.PreventDeletionAnnotation),
			pluginPreset.Annotations, fmt.Sprintf("PluginPreset with annotation '%s' set may not be deleted.", greenhousev1alpha2.PreventDeletionAnnotation)))
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return nil, nil
}
