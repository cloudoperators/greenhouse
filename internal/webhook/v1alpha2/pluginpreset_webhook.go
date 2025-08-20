// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

func DefaultPluginPreset(_ context.Context, _ client.Client, o runtime.Object) error {
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

	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha2-pluginpreset,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update;delete,versions=v1alpha2,name=vpluginpreset-v1alpha2.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginPreset(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginPreset, ok := o.(*greenhousev1alpha2.PluginPreset)
	if !ok {
		return nil, nil
	}
	var allErrs field.ErrorList
	var allWarns admission.Warnings

	// ensure PluginDefinition and ClusterSelector are set
	if pluginPreset.Spec.Plugin.PluginDefinition == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinition must be set"))
	}

	if err := validateClusterSelector(pluginPreset.Spec.ClusterSelector, pluginPreset.GroupVersionKind().GroupKind()); err != nil {
		return nil, err
	}

	// ensure ClusterName is not set
	if pluginPreset.Spec.Plugin.ClusterName != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("clusterName"), pluginPreset.Spec.Plugin.ClusterName, "spec.plugin.clusterName must not be set"))
	}

	if err := webhook.ValidateReleaseName(pluginPreset.Spec.Plugin.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("releaseName"), pluginPreset.Spec.Plugin.ReleaseName, err.Error()))
	}

	// ensure PluginDefinition exists
	pluginDefinition := new(greenhousev1alpha1.ClusterPluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginPreset.Spec.Plugin.PluginDefinition}, pluginDefinition)
	switch {
	case err != nil && apierrors.IsNotFound(err):
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, fmt.Sprintf("PluginDefinition %s does not exist", pluginPreset.Spec.Plugin.PluginDefinition)))
	case err != nil:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("pluginDefinition"), pluginPreset.Spec.Plugin.PluginDefinition, "PluginDefinition could not be retrieved: "+err.Error()))
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", labelValidationWarning)
	}

	// validate OptionValues defined by the Preset
	if errList := ValidatePluginOptionValuesForPreset(pluginPreset, pluginDefinition); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return allWarns, nil
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
