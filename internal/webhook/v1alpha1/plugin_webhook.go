// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chartutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// pluginsAllowedInCentralCluster is a list of PluginDefinitions that are allowed to be installed in the central cluster.
// TODO: Make this configurable on pluginDefinition level (AdminPlugin discussion) instead of maintaining a list here.
var pluginsAllowedInCentralCluster = []string{
	"alerts", "doop", "heureka", "kube-monitoring", "kubeconfig-generator", "perses", "repo-guard", "service-proxy", "teams2slack", "thanos",
}

// This is the prefix to identify secrets referenced directly from vault/openBao.
// TODO: Consume this constant from the tool integrating Greenhouse with vault/openBao, once implemented.
// TODO: Update docs once the complete flow is implemented
// https://github.com/cloudoperators/greenhouse/issues/1211
const (
	VaultPrefix string = "vault+kvv2:///"
)

// SetupPluginWebhookWithManager configures the webhook for the Plugin custom resource.
func SetupPluginWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.Plugin{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultPlugin,
			ValidateCreateFunc: ValidateCreatePlugin,
			ValidateUpdateFunc: ValidateUpdatePlugin,
			ValidateDeleteFunc: ValidateDeletePlugin,
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
		plugin.Labels = make(map[string]string)
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

	// Validate before ValidateCreatePlugin is called. Because defaulting PluginOptionValues & ReleaseName requires the PluginDefinition to be set.
	if plugin.Spec.PluginDefinition == "" {
		return field.Required(field.NewPath("spec").Child("pluginDefinition"), "field is required")
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
	// Default the ReleaseName.
	if plugin.Spec.ReleaseName == "" {
		if plugin.Status.HelmReleaseStatus != nil {
			// The Plugin was already deployed, use the Plugin's name as the release name.
			// This is the legacy behavior and needs to be honored to not break existing deployments.
			plugin.Spec.ReleaseName = plugin.Name
		} else {
			// The Plugin is newly created, use the PluginDefinition's HelmChart name as the release name.
			pluginDefinition := new(greenhousev1alpha1.ClusterPluginDefinition)
			err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
			if err != nil {
				return err
			}
			if pluginDefinition.Spec.HelmChart == nil {
				return field.InternalError(field.NewPath("spec").Child("pluginDefinition"), fmt.Errorf("PluginDefinition %s does not have a HelmChart", plugin.Spec.PluginDefinition))
			}
			plugin.Spec.ReleaseName = pluginDefinition.Spec.HelmChart.Name
		}
	}
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-plugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update;delete,versions=v1alpha1,name=vplugin.kb.io,admissionReviewVersions=v1

func ValidateCreatePlugin(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	pluginDefinition := new(greenhousev1alpha1.ClusterPluginDefinition)
	if plugin.Spec.PluginDefinition == "" {
		return nil, field.Required(field.NewPath("spec").Child("pluginDefinition"), "field is required")
	}
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, field.NotFound(field.NewPath("spec").Child("pluginDefinition"), plugin.Spec.PluginDefinition)
		}
		return nil, field.InternalError(field.NewPath("spec").Child("pluginDefinition"), err)
	}

	if err := validateReleaseName(plugin.Spec.ReleaseName); err != nil {
		return nil, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error())
	}

	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	errList := validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)
	if len(errList) > 0 {
		return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
	}
	if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition); err != nil {
		return nil, err
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, plugin)
	if labelValidationWarning != "" {
		return admission.Warnings{"Plugin should have a support-group Team set as its owner", labelValidationWarning}, nil
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
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validation.ValidateImmutableField(oldPlugin.Spec.PluginDefinition, plugin.Spec.PluginDefinition, field.NewPath("spec", "pluginDefinition"))...)

	pluginDefinition := new(greenhousev1alpha1.ClusterPluginDefinition)
	err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: plugin.Spec.PluginDefinition}, pluginDefinition)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return allWarns, field.NotFound(field.NewPath("spec").Child("pluginDefinition"), plugin.Spec.PluginDefinition)
		}
		return allWarns, field.InternalError(field.NewPath("spec").Child("pluginDefinition"), err)
	}

	if err := validateReleaseName(plugin.Spec.ReleaseName); err != nil {
		return allWarns, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error())
	}

	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, plugin)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "Plugin should have a support-group Team set as its owner", labelValidationWarning)
	}

	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	allErrs = append(allErrs, validatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath)...)

	allErrs = append(allErrs, validation.ValidateImmutableField(oldPlugin.Spec.ClusterName, plugin.Spec.ClusterName,
		field.NewPath("spec", "clusterName"))...)

	allErrs = append(allErrs, validation.ValidateImmutableField(oldPlugin.Spec.ReleaseNamespace, plugin.Spec.ReleaseNamespace,
		field.NewPath("spec", "releaseNamespace"))...)

	if oldPlugin.Spec.ReleaseName == "" && plugin.Status.HelmReleaseStatus != nil {
		if plugin.Name != plugin.Spec.ReleaseName {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("releaseName"), "ReleaseName for existing Plugin cannot be changed"))
		}
	}

	return allWarns, allErrs.ToAggregate()
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

func validatePluginOptionValues(
	optionValues []greenhousev1alpha1.PluginOptionValue,
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
			// If the option is required, it must be set.
			isOptionValueSet = true
			fieldPathWithIndex := optionsFieldPath.Index(idx)

			// Value and ValueFrom are mutually exclusive, but one must be provided.
			if (val.Value == nil && val.ValueFrom == nil) || (val.Value != nil && val.ValueFrom != nil) {
				allErrs = append(allErrs, field.Required(
					fieldPathWithIndex,
					"must provide either value or valueFrom for value "+val.Name,
				))
				continue
			}

			// Validate that OptionValue has a secret reference.
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

// validateReleaseName checks if the release name is valid according to Helm's rules.
func validateReleaseName(name string) error {
	if name == "" {
		return nil
	}
	return chartutil.ValidateReleaseName(name)
}

func validatePluginForCluster(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition) error {
	// Exclude front-end only Plugins as well as the greenhouse namespace from the below check.
	if pluginDefinition.Spec.HelmChart == nil || plugin.GetNamespace() == "greenhouse" {
		return nil
	}
	// Ensure whitelisted plugins are deployed in the organization namespace
	if plugin.Spec.ClusterName == "" && slices.Contains(pluginsAllowedInCentralCluster, plugin.Spec.PluginDefinition) {
		if plugin.Spec.ReleaseNamespace != plugin.GetNamespace() {
			return field.Forbidden(field.NewPath("spec").Child("releaseNamespace"), "plugins running in the central cluster can only be deployed in the same namespace as the plugin")
		}
		return nil
	}

	// If the PluginDefinition is not allowed in the central cluster, the Plugin must have a spec.clusterName set.
	clusterName := plugin.Spec.ClusterName
	if clusterName == "" {
		return field.Required(field.NewPath("spec").Child("clusterName"), "the clusterName must be set")
	}
	// Verify that the cluster exists.
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := c.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: clusterName}, cluster); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return field.NotFound(field.NewPath("spec").Child("clusterName"), clusterName)
		default:
			return field.InternalError(field.NewPath("spec").Child("clusterName"), err)
		}
	}
	return nil
}
