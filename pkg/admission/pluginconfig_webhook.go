// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

// pluginsAllowedInCentralCluster is a list of plugins that are allowed to be installed in the central cluster.
// TODO: Make this configurable on plugin level (AdminPlugin discussion) instead of maintaining a list here.
var pluginsAllowedInCentralCluster = []string{
	"alerts", "doop", "service-proxy", "teams2slack", "kubeconfig-generator",
}

// SetupPluginConfigWebhookWithManager configures the webhook for the PluginConfig custom resource.
func SetupPluginConfigWebhookWithManager(mgr ctrl.Manager) error {
	return setupWebhook(mgr,
		&greenhousev1alpha1.PluginConfig{},
		webhookFuncs{
			defaultFunc:        DefaultPluginConfig,
			validateCreateFunc: ValidateCreatePluginConfig,
			validateUpdateFunc: ValidateUpdatePluginConfig,
			validateDeleteFunc: ValidateDeletePluginConfig,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-pluginconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginconfigs,verbs=create;update,versions=v1alpha1,name=mpluginconfig.kb.io,admissionReviewVersions=v1

func DefaultPluginConfig(ctx context.Context, c client.Client, obj runtime.Object) error {
	pluginConfig, ok := obj.(*greenhousev1alpha1.PluginConfig)
	if !ok {
		return nil
	}
	if pluginConfig.Labels == nil {
		pluginConfig.Labels = make(map[string]string, 0)
	}
	// The label is used to help identifying PluginConfigs, e.g. if a Plugin changes.
	pluginConfig.Labels[greenhouseapis.LabelKeyPlugin] = pluginConfig.Spec.Plugin
	pluginConfig.Labels[greenhouseapis.LabelKeyCluster] = pluginConfig.Spec.ClusterName

	// Default the displayName to a normalized version of metadata.name.
	if pluginConfig.Spec.DisplayName == "" {
		normalizedName := strings.ReplaceAll(pluginConfig.GetName(), "-", " ")
		normalizedName = strings.TrimSpace(normalizedName)
		pluginConfig.Spec.DisplayName = normalizedName
	}

	// Default option values and merge with Plugin values.
	optionValues, err := helm.GetPluginOptionValuesForPluginConfig(ctx, c, pluginConfig)
	if err != nil {
		return err
	}
	pluginConfig.Spec.OptionValues = optionValues
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-pluginconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=pluginconfigs,verbs=create;update,versions=v1alpha1,name=vpluginconfig.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginConfig(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	pluginConfig, ok := obj.(*greenhousev1alpha1.PluginConfig)
	if !ok {
		return nil, nil
	}

	plugin := new(greenhousev1alpha1.Plugin)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginConfig.Spec.Plugin}, plugin)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validatePluginConfigOptionValues(pluginConfig, plugin); err != nil {
		return nil, err
	}
	if err := validatePluginConfigForCluster(ctx, c, pluginConfig, plugin); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateUpdatePluginConfig(ctx context.Context, c client.Client, old, obj runtime.Object) (admission.Warnings, error) {
	oldPluginConfig, ok := obj.(*greenhousev1alpha1.PluginConfig)
	if !ok {
		return nil, nil
	}
	pluginConfig, ok := obj.(*greenhousev1alpha1.PluginConfig)
	if !ok {
		return nil, nil
	}

	plugin := new(greenhousev1alpha1.Plugin)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: pluginConfig.Spec.Plugin}, plugin)
	if err != nil {
		// TODO: provide actual APIError
		return nil, err
	}

	if err := validatePluginConfigOptionValues(pluginConfig, plugin); err != nil {
		return nil, err
	}
	if err := validatePluginConfigForCluster(ctx, c, pluginConfig, plugin); err != nil {
		return nil, err
	}
	if err := validateImmutableField(oldPluginConfig.Spec.ClusterName, pluginConfig.Spec.ClusterName,
		field.NewPath("spec", "clusterName"),
	); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateDeletePluginConfig(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePluginConfigOptionValues(pluginConfig *greenhousev1alpha1.PluginConfig, plugin *greenhousev1alpha1.Plugin) error {
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
				fmt.Sprintf("Option '%s' is required by Plugin '%s'", pluginOption.Name, pluginConfig.Spec.Plugin)))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(pluginConfig.GroupVersionKind().GroupKind(), pluginConfig.Name, allErrs)
}

func validatePluginConfigForCluster(ctx context.Context, c client.Client, pluginConfig *greenhousev1alpha1.PluginConfig, plugin *greenhousev1alpha1.Plugin) error {
	// Exclude whitelisted and front-end only PluginConfigs as well as the greenhouse namespace from the below check.
	if slices.Contains(pluginsAllowedInCentralCluster, pluginConfig.Spec.Plugin) || plugin.Spec.HelmChart == nil || pluginConfig.GetNamespace() == "greenhouse" {
		return nil
	}

	// If the Plugin is not allowed in the central cluster, the PluginConfig must have a spec.clusterName set.
	clusterName := pluginConfig.Spec.ClusterName
	if clusterName == "" {
		return field.Required(field.NewPath("spec").Child("clusterName"), "the clusterName must be set")
	}
	// Verify that the cluster exists.
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := c.Get(ctx, types.NamespacedName{Namespace: pluginConfig.ObjectMeta.Namespace, Name: clusterName}, cluster); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return field.NotFound(field.NewPath("spec").Child("clusterName"), clusterName)
		default:
			return field.InternalError(field.NewPath("spec").Child("clusterName"), err)
		}
	}
	return nil
}
