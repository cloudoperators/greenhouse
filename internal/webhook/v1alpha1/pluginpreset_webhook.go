// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the PluginPreset custom resource.

func SetupPluginPresetWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.PluginPreset{},
		webhook.WebhookFuncs[*greenhousev1alpha1.PluginPreset]{
			DefaultFunc:        DefaultPluginPreset,
			ValidateCreateFunc: ValidateCreatePluginPreset,
			ValidateUpdateFunc: ValidateUpdatePluginPreset,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginpreset,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update,versions=v1alpha1,name=mpluginpreset.kb.io,admissionReviewVersions=v1

func DefaultPluginPreset(ctx context.Context, c client.Client, pluginPreset *greenhousev1alpha1.PluginPreset) error {
	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind == "" {
		pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind = greenhousev1alpha1.PluginDefinitionKind
	}

	// Set a label identifying the referenced PluginDefinition for easier listing and watch-based reconciliation.
	if pluginPreset.Labels == nil {
		pluginPreset.Labels = make(map[string]string)
	}
	switch pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind {
	case greenhousev1alpha1.PluginDefinitionKind:
		pluginPreset.Labels[greenhouseapis.LabelKeyPluginDefinition] = pluginPreset.Spec.Plugin.PluginDefinitionRef.Name
		delete(pluginPreset.Labels, greenhouseapis.LabelKeyClusterPluginDefinition)
	case greenhousev1alpha1.ClusterPluginDefinitionKind:
		pluginPreset.Labels[greenhouseapis.LabelKeyClusterPluginDefinition] = pluginPreset.Spec.Plugin.PluginDefinitionRef.Name
		delete(pluginPreset.Labels, greenhouseapis.LabelKeyPluginDefinition)
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginpreset,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginpresets,verbs=create;update;delete,versions=v1alpha1,name=vpluginpreset.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginPreset(ctx context.Context, c client.Client, pluginPreset *greenhousev1alpha1.PluginPreset) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var allWarns admission.Warnings

	if warn := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset); warn != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", warn)
	}

	// ensure PluginDefinition reference is set
	pluginDefinitionRefNamePath := field.NewPath("spec", "plugin", "pluginDefinitionRef", "name")
	pluginDefinitionRefKindPath := field.NewPath("spec", "plugin", "pluginDefinitionRef", "kind")
	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Name == "" {
		allErrs = append(allErrs, field.Invalid(pluginDefinitionRefNamePath, pluginPreset.Spec.Plugin.PluginDefinitionRef.Name, "PluginDefinition name must be set"))
	}
	if pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind == "" {
		allErrs = append(allErrs, field.Invalid(pluginDefinitionRefKindPath, pluginPreset.Spec.Plugin.PluginDefinitionRef.Kind, "PluginDefinition kind must be set"))
	}

	// ensure ClusterSelector is set
	if pluginPreset.Spec.ClusterSelector.Size() == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("clusterSelector"), pluginPreset.Spec.ClusterSelector, "ClusterSelector must be set"))
	}

	if err := validateReleaseName(pluginPreset.Spec.Plugin.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("plugin").Child("releaseName"), pluginPreset.Spec.Plugin.ReleaseName, err.Error()))
	}

	// validate WaitFor items are unique and that PluginRef's fields are mutually exclusive
	if errList := validateWaitForPluginRefs(pluginPreset.Spec.WaitFor, false); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}

	return allWarns, nil
}

func ValidateUpdatePluginPreset(ctx context.Context, c client.Client, oldPluginPreset, pluginPreset *greenhousev1alpha1.PluginPreset) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var allWarns admission.Warnings

	if warn := webhook.ValidateLabelOwnedBy(ctx, c, pluginPreset); warn != "" {
		allWarns = append(allWarns, "PluginPreset should have a support-group Team set as its owner", warn)
	}

	// validate WaitFor items are unique and that PluginRef's fields are mutually exclusive
	if errList := validateWaitForPluginRefs(pluginPreset.Spec.WaitFor, false); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) > 0 {
		return allWarns, apierrors.NewInvalid(pluginPreset.GroupVersionKind().GroupKind(), pluginPreset.Name, allErrs)
	}
	return allWarns, nil
}

// validatePluginOptionValuesForPreset validates plugin options and their values, but skips the check for required options.
// Required options are checked at the Plugin creation level, because the preset can override options and we cannot predict what clusters will be a part of the PluginPreset later on.
func validatePluginOptionValuesForPreset(pluginPreset *greenhousev1alpha1.PluginPreset, pluginDefinitionName string, pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec) field.ErrorList {
	var allErrs field.ErrorList

	optionValuesPath := field.NewPath("spec").Child("plugin").Child("optionValues")
	errors := validatePresetPluginOptionValues(pluginPreset.Spec.Plugin.OptionValues, pluginDefinitionName, pluginDefinitionSpec, false, optionValuesPath)
	allErrs = append(allErrs, errors...)

	for idx, overridesForSingleCluster := range pluginPreset.Spec.ClusterOptionOverrides {
		optionOverridesPath := field.NewPath("spec").Child("clusterOptionOverrides").Index(idx).Child("overrides")
		errors = validatePresetPluginOptionValues(overridesForSingleCluster.Overrides, pluginDefinitionName, pluginDefinitionSpec, false, optionOverridesPath)
		allErrs = append(allErrs, errors...)
	}
	return allErrs
}

func validatePresetPluginOptionValues(
	optionValues []greenhousev1alpha1.PluginPresetPluginOptionValue,
	pluginDefinitionName string,
	pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec,
	checkRequiredOptions bool,
	optionsFieldPath *field.Path,
) field.ErrorList {

	var allErrs field.ErrorList
	var isOptionValueSet bool

	for _, pluginOption := range pluginDefinitionSpec.Options {
		isOptionValueSet = false
		for idx, val := range optionValues {
			if pluginOption.Name != val.Name {
				continue
			}
			isOptionValueSet = true
			fieldPathWithIndex := optionsFieldPath.Index(idx)

			sources := 0
			if val.Value != nil {
				sources++
			}
			if val.ValueFrom != nil {
				sources++
			}
			if val.Expression != nil {
				sources++
			}
			if sources == 0 {
				allErrs = append(allErrs, field.Required(
					fieldPathWithIndex,
					"must provide exactly one of value, valueFrom, or expression for value "+val.Name,
				))
				continue
			}

			if sources > 1 {
				allErrs = append(allErrs, field.Invalid(
					fieldPathWithIndex,
					"multiple value sources set",
					"must provide exactly one of value, valueFrom, or expression for value "+val.Name,
				))
				continue
			}

			if val.Expression != nil {
				continue
			}

			if val.ValueFrom != nil && val.ValueFrom.Ref != nil {
				allErrs = append(allErrs, field.Forbidden(
					fieldPathWithIndex.Child("valueFrom").Child("ref"),
					"valueFrom.ref is not supported; use valueFrom.secret or expression",
				))
				continue
			}

			if pluginOption.Type == greenhousev1alpha1.PluginOptionTypeSecret {
				switch {
				case val.Value != nil:
					var valStr string
					if err := json.Unmarshal(val.Value.Raw, &valStr); err != nil {
						allErrs = append(allErrs, field.TypeInvalid(fieldPathWithIndex.Child("value"), "*****", err.Error()))
					}
					if !strings.HasPrefix(valStr, VaultPrefix) {
						allErrs = append(allErrs, field.TypeInvalid(fieldPathWithIndex.Child("value"), "*****",
							fmt.Sprintf("optionValue %s of type secret without secret reference must use value with vault reference prefixed by schema %q", val.Name, VaultPrefix)))
					}
					continue
				case val.ValueFrom != nil && val.ValueFrom.Secret != nil:
					if val.ValueFrom.Secret.Name == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("secret").Child("name"),
							fmt.Sprintf("optionValue %s of type secret must reference a secret by name", val.Name)))
						continue
					}
					if val.ValueFrom.Secret.Key == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("secret").Child("key"),
							fmt.Sprintf("optionValue %s of type secret must reference a key in a secret", val.Name)))
						continue
					}
				}
				continue
			}

			if val.Value != nil {
				if err := pluginOption.IsValidValue(val.Value); err != nil {
					var v any
					if err := json.Unmarshal(val.Value.Raw, &v); err != nil {
						v = err
					}
					allErrs = append(allErrs, field.Invalid(
						fieldPathWithIndex.Child("value"), v, err.Error(),
					))
				}
			}
		}
		if checkRequiredOptions && pluginOption.Required && !isOptionValueSet {
			allErrs = append(allErrs, field.Required(optionsFieldPath,
				fmt.Sprintf("Option '%s' is required by PluginDefinition '%s'", pluginOption.Name, pluginDefinitionName)))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

// validateWaitForPluginRefs validates that the WaitFor list is unique and that each PluginRef has exactly one field set.
func validateWaitForPluginRefs(items []greenhousev1alpha1.WaitForItem, isPluginInCentralCluster bool) field.ErrorList {
	itemsPath := field.NewPath("spec", "waitFor")

	seenPluginNames := make(map[string]int, 0)
	seenPluginPresets := make(map[string]int, 0)

	var errList field.ErrorList

	for i, item := range items {
		switch {
		case item.Name != "" && item.PluginPreset != "":
			errList = append(errList, field.Invalid(itemsPath.Index(i).Child("pluginRef", "name"), item.Name,
				"pluginRef.name and pluginRef.pluginPreset are mutually exclusive"))
		case item.Name != "":
			if first, dup := seenPluginNames[item.Name]; dup {
				errList = append(errList, field.Duplicate(itemsPath.Index(first).Child("pluginRef", "name"), item.Name))
				errList = append(errList, field.Duplicate(itemsPath.Index(i).Child("pluginRef", "name"), item.Name))
			} else {
				seenPluginNames[item.Name] = i
			}
		case item.PluginPreset != "":
			if isPluginInCentralCluster {
				errList = append(errList, field.Invalid(itemsPath.Index(i).Child("pluginRef", "pluginPreset"),
					item.PluginPreset, "plugins running in the central cluster cannot have PluginPreset dependencies"))
				continue
			}
			if first, dup := seenPluginPresets[item.PluginPreset]; dup {
				errList = append(errList, field.Duplicate(itemsPath.Index(first).Child("pluginRef", "pluginPreset"), item.PluginPreset))
				errList = append(errList, field.Duplicate(itemsPath.Index(i).Child("pluginRef", "pluginPreset"), item.PluginPreset))
			} else {
				seenPluginPresets[item.PluginPreset] = i
			}
		default:
			errList = append(errList, field.Required(itemsPath.Index(i).Child("pluginRef"),
				"either pluginRef.name or pluginRef.pluginPreset must be set"))
		}
	}
	return errList
}
