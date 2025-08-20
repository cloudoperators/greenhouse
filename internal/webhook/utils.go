// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chartutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func SetupWebhook(mgr ctrl.Manager, obj runtime.Object, webhookFuncs WebhookFuncs) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(obj).
		WithDefaulter(setupCustomDefaulterWithManager(mgr, webhookFuncs)).
		WithValidator(setupCustomValidatorWithManager(mgr, webhookFuncs)).
		Complete()
}

type (
	defaultFunc func(ctx context.Context, c client.Client, obj runtime.Object) error
	genericFunc func(ctx context.Context, c client.Client, obj runtime.Object) (admission.Warnings, error)
	updateFunc  func(ctx context.Context, c client.Client, oldObj, curObj runtime.Object) (admission.Warnings, error)

	WebhookFuncs struct {
		DefaultFunc        defaultFunc
		ValidateCreateFunc genericFunc
		ValidateUpdateFunc updateFunc
		ValidateDeleteFunc genericFunc
	}
)

var _ admission.CustomDefaulter = &customDefaulter{}

type customDefaulter struct {
	client.Client
	defaultFunc defaultFunc
}

func setupCustomDefaulterWithManager(mgr ctrl.Manager, webhookFuncs WebhookFuncs) *customDefaulter {
	return &customDefaulter{
		Client:      mgr.GetClient(),
		defaultFunc: webhookFuncs.DefaultFunc,
	}
}

func (c *customDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	if c.defaultFunc == nil {
		return nil
	}
	return c.defaultFunc(ctx, c.Client, obj)
}

var _ admission.CustomValidator = &customValidator{}

type customValidator struct {
	client.Client
	validateCreate, validateDelete genericFunc
	validateUpdate                 updateFunc
}

func setupCustomValidatorWithManager(mgr ctrl.Manager, webhookFuncs WebhookFuncs) *customValidator {
	return &customValidator{
		Client:         mgr.GetClient(),
		validateCreate: webhookFuncs.ValidateCreateFunc,
		validateUpdate: webhookFuncs.ValidateUpdateFunc,
		validateDelete: webhookFuncs.ValidateDeleteFunc,
	}
}

func (c *customValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	logAdmissionRequest(ctx)
	if c.validateCreate == nil {
		return nil, nil
	}
	return c.validateCreate(ctx, c.Client, obj)
}

func (c *customValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	logAdmissionRequest(ctx)
	if c.validateUpdate == nil {
		return nil, nil
	}
	return c.validateUpdate(ctx, c.Client, oldObj, newObj)
}

func (c *customValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	logAdmissionRequest(ctx)
	if c.validateDelete == nil {
		return nil, nil
	}
	return c.validateDelete(ctx, c.Client, obj)
}

func ValidateImmutableField(oldValue, newValue string, path *field.Path) *field.Error {
	if oldValue != newValue {
		return field.Invalid(path, newValue, "field is immutable")
	}
	return nil
}

func ValidateURL(str string) bool {
	parsedURL, err := url.Parse(str)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}
	return parsedURL.Scheme == "https"
}

// invalidateDoubleDashes validates that the object name does not contain double dashes.
func InvalidateDoubleDashesInName(obj client.Object, l logr.Logger) error {
	if strings.Contains(obj.GetName(), "--") {
		err := apierrors.NewInvalid(
			obj.GetObjectKind().GroupVersionKind().GroupKind(),
			obj.GetName(),
			field.ErrorList{
				field.Invalid(field.NewPath("metadata", "name"), obj.GetName(), "name cannot contain double dashes"),
			},
		)
		l.Error(err, "found object name with double dashes, admission will be denied")
		return err
	}
	return nil
}

// capName validates that the name is not longer than the provided length.
func CapName(obj client.Object, l logr.Logger, length int) error {
	if len(obj.GetName()) > length {
		err := apierrors.NewInvalid(
			obj.GetObjectKind().GroupVersionKind().GroupKind(),
			obj.GetName(),
			field.ErrorList{
				field.Invalid(field.NewPath("metadata", "name"), obj.GetName(), fmt.Sprintf("name must be less than or equal to %d", length)),
			},
		)
		l.Error(err, fmt.Sprintf("found object name too long, admission will be denied, name must be less than or equal to %d", length))
		return err
	}
	return nil
}

// validateReleaseName checks if the release name is valid according to Helm's rules.
func ValidateReleaseName(name string) error {
	if name == "" {
		return nil
	}
	return chartutil.ValidateReleaseName(name)
}

func ValidatePluginOptionValues(
	optionValues []greenhousemetav1alpha1.PluginOptionValue,
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

// ValidateLabelOwnedBy validates that the owned-by label is present and that it references an existing Team.
func ValidateLabelOwnedBy(ctx context.Context, c client.Client, resourceObj v1.Object) string {
	namespace := resourceObj.GetNamespace()
	if namespace == "" {
		warnErr := field.Required(field.NewPath("metadata").Child("namespace"),
			"namespace is required to validate the owner")
		return warnErr.Error()
	}

	ownerName, ok := resourceObj.GetLabels()[greenhouseapis.LabelKeyOwnedBy]
	if !ok {
		warnErr := field.Required(field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			fmt.Sprintf("label %s is required", greenhouseapis.LabelKeyOwnedBy))
		return warnErr.Error()
	}
	if ownerName == "" {
		warnErr := field.Required(field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			fmt.Sprintf("label %s value is required", greenhouseapis.LabelKeyOwnedBy))
		return warnErr.Error()
	}

	team := new(greenhousev1alpha1.Team)
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ownerName}, team)
	switch {
	case err != nil && apierrors.IsNotFound(err):
		warnErr := field.Invalid(field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			ownerName,
			fmt.Sprintf("team %s does not exist in the resource namespace", ownerName))
		return warnErr.Error()
	case err != nil:
		warnErr := field.Invalid(field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			ownerName,
			fmt.Sprintf("team %s could not be retrieved: %s", ownerName, err.Error()))
		return warnErr.Error()
	}

	supportGroup, ok := team.Labels[greenhouseapis.LabelKeySupportGroup]
	if !ok || supportGroup != "true" {
		warnErr := field.Invalid(field.NewPath("metadata").Child("labels").Key(greenhouseapis.LabelKeyOwnedBy),
			ownerName,
			fmt.Sprintf("owner team %s should be a support group", ownerName))
		return warnErr.Error()
	}
	return ""
}

// logAdmissionRequest logs the AdmissionRequest.
// This is necessary to audit log the AdmissionRequest independently of the api server audit logs.
func logAdmissionRequest(ctx context.Context) {
	admissionRequest, err := admission.RequestFromContext(ctx)
	if err != nil {
		return
	}

	// Remove all objects from the log
	admissionRequest.Object.Raw = nil
	admissionRequest.OldObject.Raw = nil

	ctrl.Log.Info("AdmissionRequest", "Request", admissionRequest)
}
