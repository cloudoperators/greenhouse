// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

// Webhook for the PluginDefinition custom resource.

func SetupPluginDefinitionWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.PluginDefinition{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultPluginDefinition,
			ValidateCreateFunc: ValidateCreatePluginDefinition,
			ValidateUpdateFunc: ValidateUpdatePluginDefinition,
			ValidateDeleteFunc: ValidateDeletePluginDefinition,
		},
	)
}

func SetupClusterPluginDefinitionWebhookWithManager(mgr ctrl.Manager) error {
	return webhook.SetupWebhook(mgr,
		&greenhousev1alpha1.ClusterPluginDefinition{},
		webhook.WebhookFuncs{
			DefaultFunc:        DefaultPluginDefinition,
			ValidateCreateFunc: ValidateCreateClusterPluginDefinition,
			ValidateUpdateFunc: ValidateUpdateClusterPluginDefinition,
			ValidateDeleteFunc: ValidateDeleteClusterPluginDefinition,
		},
	)
}

//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-plugindefinition,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugindefinitions,verbs=create;update,versions=v1alpha1,name=mplugindefinition.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/mutate-greenhouse-sap-v1alpha1-clusterplugindefinition,mutating=true,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusterplugindefinitions,verbs=create;update,versions=v1alpha1,name=mclusterplugindefinition.kb.io,admissionReviewVersions=v1

func DefaultPluginDefinition(_ context.Context, _ client.Client, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-plugindefinition,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=plugindefinitions,verbs=create;update;delete,versions=v1alpha1,name=vplugindefinition.kb.io,admissionReviewVersions=v1

func ValidateCreatePluginDefinition(_ context.Context, _ client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.PluginDefinition)
	if !ok {
		return nil, nil
	}
	return nil, validateCreate(pluginDefinition.Spec, pluginDefinition.GroupVersionKind(), pluginDefinition.GetName())
}

//+kubebuilder:webhook:path=/validate-greenhouse-sap-v1alpha1-clusterplugindefinition,mutating=false,failurePolicy=fail,sideEffects=None,groups=greenhouse.sap,resources=clusterplugindefinitions,verbs=create;update;delete,versions=v1alpha1,name=vclusterplugindefinition.kb.io,admissionReviewVersions=v1

func ValidateCreateClusterPluginDefinition(_ context.Context, _ client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.ClusterPluginDefinition)
	if !ok {
		return nil, nil
	}
	return nil, validateCreate(pluginDefinition.Spec, pluginDefinition.GroupVersionKind(), pluginDefinition.GetName())
}

func validateCreate(pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, gvk schema.GroupVersionKind, name string) error {
	if err := validatePluginDefinitionMustSpecifyHelmChartOrUIApplication(pluginDefinitionSpec, gvk, name); err != nil {
		return err
	}
	if err := validatePluginDefinitionMustSpecifyVersion(pluginDefinitionSpec); err != nil {
		return err
	}
	return validatePluginDefinitionOptionValueAndType(pluginDefinitionSpec, gvk, name)
}

func ValidateUpdatePluginDefinition(_ context.Context, _ client.Client, _, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.PluginDefinition)
	if !ok {
		return nil, nil
	}
	return nil, validateUpdate(pluginDefinition.Spec, pluginDefinition.GroupVersionKind(), pluginDefinition.GetName())
}

func ValidateUpdateClusterPluginDefinition(_ context.Context, _ client.Client, _, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.ClusterPluginDefinition)
	if !ok {
		return nil, nil
	}
	return nil, validateUpdate(pluginDefinition.Spec, pluginDefinition.GroupVersionKind(), pluginDefinition.GetName())
}

func validateUpdate(pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, gvk schema.GroupVersionKind, name string) error {
	if err := validatePluginDefinitionMustSpecifyHelmChartOrUIApplication(pluginDefinitionSpec, gvk, name); err != nil {
		return err
	}
	if err := validatePluginDefinitionMustSpecifyVersion(pluginDefinitionSpec); err != nil {
		return err
	}
	return validatePluginDefinitionOptionValueAndType(pluginDefinitionSpec, gvk, name)
}

func ValidateDeletePluginDefinition(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.PluginDefinition)
	if !ok {
		return nil, nil
	}
	return validateDelete(ctx, c, greenhouseapis.LabelKeyPluginDefinition, pluginDefinition.GetName())
}

func ValidateDeleteClusterPluginDefinition(ctx context.Context, c client.Client, o runtime.Object) (admission.Warnings, error) {
	pluginDefinition, ok := o.(*greenhousev1alpha1.ClusterPluginDefinition)
	if !ok {
		return nil, nil
	}
	return validateDelete(ctx, c, greenhouseapis.LabelKeyClusterPluginDefinition, pluginDefinition.GetName())
}

func validateDelete(ctx context.Context, c client.Client, labelKey, name string) (admission.Warnings, error) {
	list := &greenhousev1alpha1.PluginList{}
	opt := client.MatchingLabels{labelKey: name}
	if err := c.List(ctx, list, opt); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(list.Items) > 0 {
		return nil, apierrors.NewBadRequest("PluginDefinition is still in use by Plugins")
	}
	return nil, nil
}

func validatePluginDefinitionMustSpecifyVersion(pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec) error {
	if pluginDefinitionSpec.Version == "" {
		return field.Required(field.NewPath("spec", "version"), "PluginDefinition without spec.version is invalid.")
	}
	return nil
}

func validatePluginDefinitionMustSpecifyHelmChartOrUIApplication(pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, gvk schema.GroupVersionKind, name string) error {
	if pluginDefinitionSpec.HelmChart == nil && pluginDefinitionSpec.UIApplication == nil {
		return apierrors.NewInvalid(gvk.GroupKind(), name, field.ErrorList{
			field.Required(field.NewPath("spec").Child("helmChart", "uiApplication"),
				"A PluginDefinition without both spec.helmChart and spec.uiApplication is invalid."),
		})
	}
	return nil
}

// validatePluginDefinitionOptionValueAndType validates that the type and value of each PluginOption matches.
func validatePluginDefinitionOptionValueAndType(pluginDefinitionSpec greenhousev1alpha1.PluginDefinitionSpec, gvk schema.GroupVersionKind, name string) error {
	for _, option := range pluginDefinitionSpec.Options {
		if err := option.IsValid(); err != nil {
			return apierrors.NewInvalid(gvk.GroupKind(), name, field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("options").Child("name"), option.Name,
					"A PluginOption Default must match the specified Type, and defaults are not allowed in PluginOptions of the 'Secret' type."),
			})
		}
	}
	return nil
}
