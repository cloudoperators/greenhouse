// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

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
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// pluginsAllowedInCentralCluster is a list of PluginDefinitions that are allowed to be installed in the central cluster.
// TODO: Make this configurable on pluginDefinition level (AdminPlugin discussion) instead of maintaining a list here.
var pluginsAllowedInCentralCluster = []string{
	"alerts", "doop", "heureka", "kube-monitoring", "kubeconfig-generator", "perses", "repo-guard", "service-proxy", "teams2slack", "thanos",
}

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

	// Migrate the deprecated PluginDefinition reference
	//nolint:staticcheck
	if plugin.Spec.PluginDefinitionRef.Name == "" && plugin.Spec.PluginDefinition != "" {
		//nolint:staticcheck
		plugin.Spec.PluginDefinitionRef.Name = plugin.Spec.PluginDefinition
	}

	// Validate PluginDefinitionRef before ValidateCreatePlugin is called. Because defaulting PluginOptionValues & ReleaseName requires the PluginDefinition to be set.
	if plugin.Spec.PluginDefinitionRef.Name == "" {
		return field.Required(field.NewPath("spec").Child("pluginDefinitionRef").Child("name"), "field is required")
	}

	// Migrate the deprecated PluginDefinition reference
	if plugin.Spec.PluginDefinitionRef.Kind == "" {
		if plugin.Spec.PluginDefinitionRef.Namespace != "" {
			return field.Required(field.NewPath("spec").Child("pluginDefinitionRef").Child("kind"), "field is required")
		}

		// Check if PluginDefinition exists in Plugin's namespace
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.PluginDefinitionRef.Name}, pluginDefinition)
		if err == nil {
			plugin.Spec.PluginDefinitionRef.Namespace = plugin.Namespace
			plugin.Spec.PluginDefinitionRef.Kind = "PluginDefinition"
		} else {
			// Check if ClusterPluginDefinition exists
			clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
			err = c.Get(ctx, types.NamespacedName{Name: plugin.Spec.PluginDefinitionRef.Name}, clusterPluginDefinition)
			if err != nil {
				return err
			}
			plugin.Spec.PluginDefinitionRef.Kind = "ClusterPluginDefinition"
		}
	}

	// Default the ReleaseName.
	if plugin.Spec.ReleaseName == "" {
		if plugin.Status.HelmReleaseStatus != nil {
			// The Plugin was already deployed, use the Plugin's name as the release name.
			// This is the legacy behavior and needs to be honored to not break existing deployments.
			plugin.Spec.ReleaseName = plugin.Name
		} else {
			// The Plugin is newly created, use the PluginDefinition's HelmChart name as the release name.
			pluginDefinitionSpec, err := util.EffectivePluginDefinitionSpecFromPlugin(ctx, c, plugin)
			if err != nil {
				return err
			}
			if pluginDefinitionSpec.HelmChart == nil {
				return field.InternalError(field.NewPath("spec").Child("pluginDefinition"), fmt.Errorf("PluginDefinition %s does not have a HelmChart", plugin.Spec.PluginDefinition))
			}
			plugin.Spec.ReleaseName = pluginDefinitionSpec.HelmChart.Name
		}
	}

	// The label is used to help identifying Plugins, e.g. if a PluginDefinition changes.
	delete(plugin.Labels, greenhouseapis.LabelKeyPlugin)
	plugin.Labels[greenhouseapis.LabelKeyPluginDefinition] = plugin.Spec.PluginDefinitionRef.Name
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

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-plugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugins,verbs=create;update;delete,versions=v1alpha1,name=vplugin.kb.io,admissionReviewVersions=v1

func ValidateCreatePlugin(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error) {
	plugin, ok := obj.(*greenhousev1alpha1.Plugin)
	if !ok {
		return nil, nil
	}

	// Ensure PluginDefinitionRef is set correctly
	if fieldErr := validatePluginDefinitionReferenceForPlugin(plugin); fieldErr != nil {
		return nil, fieldErr
	}

	if err := webhook.ValidateReleaseName(plugin.Spec.ReleaseName); err != nil {
		return nil, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error())
	}

	// ensure PluginDefinition exists, validate OptionValues and Plugin for Cluster
	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	switch plugin.Spec.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: plugin.Spec.PluginDefinitionRef.Namespace,
			Name:      plugin.Spec.PluginDefinitionRef.Name,
		}, pluginDefinition)
		if apierrors.IsNotFound(err) {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s does not exist in namespace %s", plugin.Spec.PluginDefinitionRef.Name, plugin.Spec.PluginDefinitionRef.Namespace))
		} else if err != nil {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s could not be retrieved from namespace %s: %s", plugin.Spec.PluginDefinitionRef.Name, plugin.Spec.PluginDefinitionRef.Namespace, err.Error()))
		}
		// validate OptionValues defined by the Plugin
		if errList := webhook.ValidatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath); len(errList) > 0 {
			return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
		}
		// validate Plugin for Cluster
		if err := validatePluginForCluster(ctx, c, plugin, pluginDefinition.Spec); err != nil {
			return nil, err
		}
	case "ClusterPluginDefinition":
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: "",
			Name:      plugin.Spec.PluginDefinitionRef.Name,
		}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s does not exist", plugin.Spec.PluginDefinitionRef.Name))
		} else if err != nil {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s could not be retrieved: %s", plugin.Spec.PluginDefinitionRef.Name, err.Error()))
		}
		// validate OptionValues defined by the Plugin
		if errList := webhook.ValidatePluginOptionValues(plugin.Spec.OptionValues, clusterPluginDefinition.Name, clusterPluginDefinition.Spec, true, optionsFieldPath); len(errList) > 0 {
			return nil, apierrors.NewInvalid(plugin.GroupVersionKind().GroupKind(), plugin.Name, errList)
		}
		// validate Plugin for Cluster
		if err := validatePluginForCluster(ctx, c, plugin, clusterPluginDefinition.Spec); err != nil {
			return nil, err
		}
	default:
		return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "kind"), plugin.Spec.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind")
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
	labelValidationWarning := webhook.ValidateLabelOwnedBy(ctx, c, plugin)
	if labelValidationWarning != "" {
		allWarns = append(allWarns, "Plugin should have a support-group Team set as its owner", labelValidationWarning)
	}

	allErrs := field.ErrorList{}

	//nolint:staticcheck
	if err := webhook.ValidateImmutableField(oldPlugin.Spec.PluginDefinition, plugin.Spec.PluginDefinition, field.NewPath("spec", "pluginDefinition")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPlugin.Spec.PluginDefinitionRef.Name, plugin.Spec.PluginDefinitionRef.Name, field.NewPath("spec", "pluginDefinitionRef", "name")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPlugin.Spec.PluginDefinitionRef.Kind, plugin.Spec.PluginDefinitionRef.Kind, field.NewPath("spec", "pluginDefinitionRef", "kind")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := webhook.ValidateImmutableField(oldPlugin.Spec.PluginDefinitionRef.Namespace, plugin.Spec.PluginDefinitionRef.Namespace, field.NewPath("spec", "pluginDefinitionRef", "namespace")); err != nil {
		allErrs = append(allErrs, err)
	}

	allErrs = append(allErrs, validation.ValidateImmutableField(oldPlugin.Spec.ClusterName, plugin.Spec.ClusterName,
		field.NewPath("spec", "clusterName"))...)

	allErrs = append(allErrs, validation.ValidateImmutableField(oldPlugin.Spec.ReleaseNamespace, plugin.Spec.ReleaseNamespace,
		field.NewPath("spec", "releaseNamespace"))...)

	if oldPlugin.Spec.ReleaseName == "" &&
		plugin.Status.HelmReleaseStatus != nil &&
		plugin.Name != plugin.Spec.ReleaseName {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("releaseName"), "ReleaseName for existing Plugin cannot be changed"))
	}

	if err := webhook.ValidateReleaseName(plugin.Spec.ReleaseName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("releaseName"), plugin.Spec.ReleaseName, err.Error()))
	}

	if len(allErrs) > 0 {
		return allWarns, allErrs.ToAggregate()
	}

	// ensure PluginDefinition exists, validate OptionValues
	optionsFieldPath := field.NewPath("spec").Child("optionValues")
	switch plugin.Spec.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: plugin.Spec.PluginDefinitionRef.Namespace,
			Name:      plugin.Spec.PluginDefinitionRef.Name,
		}, pluginDefinition)
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s does not exist in namespace %s", plugin.Spec.PluginDefinitionRef.Name, plugin.Spec.PluginDefinitionRef.Namespace)))
			break
		}
		if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced PluginDefinition %s could not be retrieved from namespace %s: %s", plugin.Spec.PluginDefinitionRef.Name, plugin.Spec.PluginDefinitionRef.Namespace, err.Error())))
			break
		}
		// validate OptionValues defined by the Plugin
		if errList := webhook.ValidatePluginOptionValues(plugin.Spec.OptionValues, pluginDefinition.Name, pluginDefinition.Spec, true, optionsFieldPath); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	case "ClusterPluginDefinition":
		clusterPluginDefinition := &greenhousev1alpha1.ClusterPluginDefinition{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: "",
			Name:      plugin.Spec.PluginDefinitionRef.Name,
		}, clusterPluginDefinition)
		if apierrors.IsNotFound(err) {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s does not exist", plugin.Spec.PluginDefinitionRef.Name))
		}
		if err != nil {
			return nil, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "name"), plugin.Spec.PluginDefinitionRef.Name,
				fmt.Sprintf("referenced ClusterPluginDefinition %s could not be retrieved: %s", plugin.Spec.PluginDefinitionRef.Name, err.Error()))
		}
		// validate OptionValues defined by the Plugin
		if errList := webhook.ValidatePluginOptionValues(plugin.Spec.OptionValues, clusterPluginDefinition.Name, clusterPluginDefinition.Spec, true, optionsFieldPath); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "kind"), plugin.Spec.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind"))
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

func validatePluginForCluster(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin, pluginDefinitionSpec greenhousemetav1alpha1.PluginDefinitionTemplateSpec) error {
	// Exclude front-end only Plugins as well as the greenhouse namespace from the below check.
	if pluginDefinitionSpec.HelmChart == nil || plugin.GetNamespace() == "greenhouse" {
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

func validatePluginDefinitionReferenceForPlugin(p *greenhousev1alpha1.Plugin) *field.Error {
	// Require at least one
	//nolint:staticcheck
	if p.Spec.PluginDefinitionRef.Name == "" && p.Spec.PluginDefinition == "" {
		return field.Required(field.NewPath("spec", "pluginDefinitionRef", "name"), "either pluginDefinitionRef or pluginDefinition must be set")
	}

	// If both set, they must match
	//nolint:staticcheck
	if p.Spec.PluginDefinitionRef.Name != "" && p.Spec.PluginDefinition != "" &&
		//nolint:staticcheck
		p.Spec.PluginDefinitionRef.Name != p.Spec.PluginDefinition {
		//nolint:staticcheck
		return field.Invalid(field.NewPath("spec", "pluginDefinition"), p.Spec.PluginDefinition, "pluginDefinitionRef.name does not match deprecated pluginDefinition")
	}

	// Validate Kind and Namespace
	switch p.Spec.PluginDefinitionRef.Kind {
	case "PluginDefinition":
		if p.Spec.PluginDefinitionRef.Namespace == "" {
			return field.Required(field.NewPath("spec", "pluginDefinitionRef", "namespace"), "pluginDefinitionRef.namespace must be set when kind is PluginDefinition")
		}
	case "ClusterPluginDefinition":
		if p.Spec.PluginDefinitionRef.Namespace != "" {
			return field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "namespace"), p.Spec.PluginDefinitionRef.Namespace, "pluginDefinitionRef.namespace must be empty when kind is ClusterPluginDefinition")
		}
	default:
		return field.Invalid(field.NewPath("spec", "pluginDefinitionRef", "kind"), p.Spec.PluginDefinitionRef.Kind, "unsupported pluginDefinitionRef.kind")
	}

	return nil
}
