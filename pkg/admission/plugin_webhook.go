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
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

// pluginsAllowedInCentralCluster is a list of PluginDefinitions that are allowed to be installed in the central cluster.
// TODO: Make this configurable on pluginDefinition level (AdminPlugin discussion) instead of maintaining a list here.
var pluginsAllowedInCentralCluster = []string{
	"alerts", "doop", "service-proxy", "teams2slack", "kubeconfig-generator",
}

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
	delete(plugin.Labels, greenhouseapis.LabelKeyPlugin)
	plugin.Labels[greenhouseapis.LabelKeyPluginDefinition] = plugin.Spec.PluginDefinition
	plugin.Labels[greenhouseapis.LabelKeyCluster] = plugin.Spec.ClusterName

	// Default the displayName to a normalized version of metadata.name.
	if plugin.Spec.DisplayName == "" {
		normalizedName := strings.ReplaceAll(plugin.GetName(), "-", " ")
		normalizedName = strings.TrimSpace(normalizedName)
		plugin.Spec.DisplayName = normalizedName
	}

	// Default option values and merge with PluginDefinition values.
	optionValues, err := helm.GetPluginOptionValuesForPlugin(ctx, c, plugin)
	if err != nil {
		return err
	}
	plugin.Spec.OptionValues = optionValues

	// Default the ReleaseNamespace to the organization namespace if not set.
	if plugin.Spec.ReleaseNamespace == "" {
		plugin.Spec.ReleaseNamespace = plugin.GetNamespace()
	}
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

	if errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition); len(errList) > 0 {
		return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
	}
	if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition); err != nil {
		return nil, err
	}
	return nil, nil
}

func ValidateUpdatePlugin(ctx context.Context, c client.Client, old, obj runtime.Object) (admission.Warnings, error) {
	var allWarns admission.Warnings
	oldPlugin, ok := old.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	allWarns = append(allWarns, validateOwnerReference(oldPlugin)...)

	pluginDefinition := new(greenhousev1alpha1.PluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		// TODO: provide actual APIError
		return allWarns, err
	}

	if errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition); len(errList) > 0 {
		return allWarns, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
	}
	if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition); err != nil {
		return allWarns, err
	}
	if err := validateImmutableField(oldPlugin.Spec.ClusterName, plugin.Spec.ClusterName,
		field.NewPath("spec", "clusterName"),
	); err != nil {
		return allWarns, err
	}
	if err := validateImmutableField(oldPlugin.Spec.ReleaseNamespace, plugin.Spec.ReleaseNamespace,
		field.NewPath("spec", "releaseNamespace"),
	); err != nil {
		return allWarns, err
	}
	return allWarns, nil
}

func ValidateDeletePlugin(_ context.Context, _ client.Client, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validateOwnerRefernce returns a Warning if the Plugin is managed by a PluginPreset
// The user is warned that the Plugin will be reconciled to the desired state specified in the PluginPreset.
func validateOwnerReference(plugin *greenhousev1alpha1.Plugin) admission.Warnings {
	if ref := clientutil.GetOwnerReference(plugin, greenhousev1alpha1.PluginPresetKind); ref != nil {
		return admission.Warnings{fmt.Sprintf("Plugin is managed by PluginPreset '%s'. Plugin will be reconciled to the desired state specified in the PluginPreset.", ref.Name)}
	}
	return nil
}

func validatePluginOptionValues(optionValues []greenhousev1alpha1.PluginOptionValue, pluginDefinition *greenhousev1alpha1.PluginDefinition) field.ErrorList {
	var allErrs field.ErrorList
	var isOptionValueSet bool
	for _, pluginOption := range pluginDefinition.Spec.Options {
		isOptionValueSet = false
		for idx, val := range optionValues {
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

			// validate that the Plugin.OptionValue matches the type of the PluginDefinition.Option
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
				fmt.Sprintf("Option '%s' is required by PluginDefinition '%s'", pluginOption.Name, pluginDefinition.Name)))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validatePluginForCluster(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, pluginDefinition *greenhousev1alpha1.PluginDefinition) error {
	// Exclude whitelisted and front-end only Plugins as well as the greenhouse namespace from the below check.
	if slices.Contains(pluginsAllowedInCentralCluster, plugin.Spec.PluginDefinition) || pluginDefinition.Spec.HelmChart == nil || plugin.GetNamespace() == "greenhouse" {
		return nil
	}

	// If the PluginDefinition is not allowed in the central cluster, the Plugin must have a spec.clusterName set.
	clusterName := plugin.Spec.ClusterName
	if clusterName == "" {
		return field.Required(field.NewPath("spec").Child("clusterName"), "the clusterName must be set")
	}
	// Verify that the cluster exists.
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
