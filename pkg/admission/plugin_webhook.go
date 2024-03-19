// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package admission

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

// SetupPluginWebhookWithManager configures the webhook for the Plugin custom resource.
func SetupPluginWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.Plugin{},
		webhookFuncs{
			defaultFunc:        DefaultPlugin,
			validateCreateFunc: ValidateCreatePlugin,
			validateUpdateFunc: ValidateUpdatePlugin,
			validateDeleteFunc: ValidateDeletePlugin,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-plugin,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update,versions=v1alpha1,name=mplugin.kb.io,admissionReviewVersions=v1

func DefaultPlugin(ctx context.Context, c client.Client, obj runtime.Object) error {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil
	}
	if plugin.Labels == nil {
		plugin.Labels = make(map[string]string, 0)
	}
	// The label is used to help identifying Plugins, e.g. if a PluginDefinition changes.
	plugin.Labels[greenhouseapis.LabelKeyPlugin] = plugin.Spec.PluginDefinition
	plugin.Labels[greenhouseapis.LabelKeyCluster] = plugin.Spec.ClusterName

	// Default the displayName to a normalized version of metadata.name.
	if plugin.Spec.DisplayName == "" {
		normalizedName := strings.ReplaceAll(plugin.GetName(), "-", " ")
		normalizedName = strings.TrimSpace(normalizedName)
		plugin.Spec.DisplayName = normalizedName
	}

	// Default option values and merge with Plugin values.
	optionValues, err := helm.GetPluginOptionValuesForPlugin(ctx, c, plugin)
	if err != nil {
		return err
	}
	plugin.Spec.OptionValues = optionValues
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-plugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update,versions=v1alpha1,name=vplugin.kb.io,admissionReviewVersions=v1

func ValidateCreatePlugin(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validatePluginConfigOptionValues(plugin, pluginDefinition); err != nil {
		return nil, err
	}
	if err := validateClusterExists(ctx, plugin, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateUpdatePlugin(ctx context.Context, c client.Client, _, obj runtime.Object) (admission.Warnings, error) {
	pluginConfig, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	plugin := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginConfig.Spec.PluginDefinition}, plugin)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validatePluginConfigOptionValues(pluginConfig, plugin); err != nil {
		return nil, err
	}
	if err := validateClusterExists(ctx, pluginConfig, c); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateDeletePlugin(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePluginConfigOptionValues(pluginConfig *greenhousev1alpha1.Plugin, plugin *greenhousev1alpha1.PluginDefinition) error {
	var allErrs field.ErrorList
	var isOptionValueSet bool
	for _, pluginOption := range plugin.Spec.Options {
		isOptionValueSet = false
		for idx, val := range pluginConfig.Spec.OptionValues {
			if pluginOption.Name != val.Name {
				continue
			}
			// If the option is required, it must be set.
			isOptionValueSet = true
			fieldPathWithIndex := field.NewPath("spec").Child("optionValues").Index(idx)

			// Value and ValueFrom are mutually exclusive, but one must be provided.
			if (val.Value == nil && val.ValueFrom == nil) || (val.Value != nil && val.ValueFrom != nil) {
				allErrs = append(allErrs, field.Required(
					fieldPathWithIndex,
					fmt.Sprintf("must provide either value or valueFrom for value %s", val.Name),
				))
				continue
			}

			// Validate that OptionValue has a secret reference.
			if pluginOption.Type == greenhousev1alpha1.PluginOptionTypeSecret {
				switch {
				case val.Value != nil:
					allErrs = append(allErrs, field.TypeInvalid(fieldPathWithIndex.Child("value"), "*****",
						fmt.Sprintf("optionValue %s of type secret must use valueFrom to reference a secret", val.Name)))
					continue
				case val.ValueFrom != nil:
					if val.ValueFrom.Secret.Name == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("name"),
							fmt.Sprintf("optionValue %s of type secret must reference a secret by name", val.Name)))
						continue
					}
					if val.ValueFrom.Secret.Key == "" {
						allErrs = append(allErrs, field.Required(fieldPathWithIndex.Child("valueFrom").Child("key"),
							fmt.Sprintf("optionValue %s of type secret must reference a key in a secret", val.Name)))
						continue
					}
				}
				continue
			}

			// validate that the PluginConfig.OptionValue matches the type of the Plugin.Option
			if val.Value != nil {
				if err := pluginOption.IsValidValue(val.Value); err != nil {
					allErrs = append(allErrs, field.Invalid(
						fieldPathWithIndex.Child("value"), val.Value.Raw, err.Error(),
					))
				}
			}
		}
		if pluginOption.Required && !isOptionValueSet {
			allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("optionValues"),
				fmt.Sprintf("Option '%s' is required by Plugin '%s'", pluginOption.Name, pluginConfig.Spec.PluginDefinition)))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(pluginConfig.GroupVersionKind().GroupKind(), pluginConfig.Name, allErrs)
}

func validateClusterExists(ctx context.Context, plugin *greenhousev1alpha1.Plugin, c client.Client) error {
	// TODO: Enforce clusterName on Plugins with backend.
	//  We allow Plugins without cluster reference during migration.
	//  Later a Plugins with a backend must have a clusterName configured. Frontend-only plugins are allowed without a clusterName.
	if plugin.Spec.ClusterName == "" {
		return nil
	}

	clusterName := plugin.Spec.ClusterName
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := c.Get(ctx, types.NamespacedName{Namespace: plugin.ObjectMeta.Namespace, Name: clusterName}, cluster); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return field.NotFound(field.NewPath("spec").Child("clusterName"), clusterName)
		default:
			return field.InternalError(field.NewPath("spec").Child("clusterName"), err)
		}
	}
	return nil
}
