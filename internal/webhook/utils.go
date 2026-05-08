// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func SetupWebhook[T runtime.Object](mgr ctrl.Manager, obj T, webhookFuncs WebhookFuncs[T]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).
		WithDefaulter(setupCustomDefaulterWithManager(mgr, webhookFuncs)).
		WithValidator(setupCustomValidatorWithManager(mgr, webhookFuncs)).
		Complete()
}

type (
	defaultFunc[T runtime.Object] func(ctx context.Context, c client.Client, obj T) error
	genericFunc[T runtime.Object] func(ctx context.Context, c client.Client, obj T) (admission.Warnings, error)
	updateFunc[T runtime.Object]  func(ctx context.Context, c client.Client, oldObj, curObj T) (admission.Warnings, error)

	WebhookFuncs[T runtime.Object] struct {
		DefaultFunc        defaultFunc[T]
		ValidateCreateFunc genericFunc[T]
		ValidateUpdateFunc updateFunc[T]
		ValidateDeleteFunc genericFunc[T]
	}
)

type customDefaulter[T runtime.Object] struct {
	client.Client
	defaultFunc defaultFunc[T]
}

func setupCustomDefaulterWithManager[T runtime.Object](mgr ctrl.Manager, webhookFuncs WebhookFuncs[T]) *customDefaulter[T] {
	return &customDefaulter[T]{
		Client:      mgr.GetClient(),
		defaultFunc: webhookFuncs.DefaultFunc,
	}
}

func (c *customDefaulter[T]) Default(ctx context.Context, obj T) error {
	if c.defaultFunc == nil {
		return nil
	}
	return c.defaultFunc(ctx, c.Client, obj)
}

type customValidator[T runtime.Object] struct {
	client.Client
	validateCreate, validateDelete genericFunc[T]
	validateUpdate                 updateFunc[T]
}

func setupCustomValidatorWithManager[T runtime.Object](mgr ctrl.Manager, webhookFuncs WebhookFuncs[T]) *customValidator[T] {
	return &customValidator[T]{
		Client:         mgr.GetClient(),
		validateCreate: webhookFuncs.ValidateCreateFunc,
		validateUpdate: webhookFuncs.ValidateUpdateFunc,
		validateDelete: webhookFuncs.ValidateDeleteFunc,
	}
}

func (c *customValidator[T]) ValidateCreate(ctx context.Context, obj T) (admission.Warnings, error) {
	logAdmissionRequest(ctx)
	if c.validateCreate == nil {
		return nil, nil
	}
	return c.validateCreate(ctx, c.Client, obj)
}

func (c *customValidator[T]) ValidateUpdate(ctx context.Context, oldObj, newObj T) (admission.Warnings, error) {
	logAdmissionRequest(ctx)
	if c.validateUpdate == nil {
		return nil, nil
	}
	return c.validateUpdate(ctx, c.Client, oldObj, newObj)
}

func (c *customValidator[T]) ValidateDelete(ctx context.Context, obj T) (admission.Warnings, error) {
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
