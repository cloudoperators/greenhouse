// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupWebhook(mgr ctrl.Manager, obj runtime.Object, webhookFuncs webhookFuncs) error {
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

	webhookFuncs struct {
		defaultFunc        defaultFunc
		validateCreateFunc genericFunc
		validateUpdateFunc updateFunc
		validateDeleteFunc genericFunc
	}
)

var _ admission.CustomDefaulter = &customDefaulter{}

type customDefaulter struct {
	client.Client
	defaultFunc defaultFunc
}

func setupCustomDefaulterWithManager(mgr ctrl.Manager, webhookFuncs webhookFuncs) *customDefaulter {
	return &customDefaulter{
		Client:      mgr.GetClient(),
		defaultFunc: webhookFuncs.defaultFunc,
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

func setupCustomValidatorWithManager(mgr ctrl.Manager, webhookFuncs webhookFuncs) *customValidator {
	return &customValidator{
		Client:         mgr.GetClient(),
		validateCreate: webhookFuncs.validateCreateFunc,
		validateUpdate: webhookFuncs.validateUpdateFunc,
		validateDelete: webhookFuncs.validateDeleteFunc,
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

func validateImmutableField(oldValue, newValue string, path *field.Path) *field.Error {
	if oldValue != newValue {
		return field.Invalid(path, newValue, "field is immutable")
	}
	return nil
}

func validateURL(str string) bool {
	parsedURL, err := url.Parse(str)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}
	return parsedURL.Scheme == "https"
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

// invalidateDoubleDashes validates that the object name does not contain double dashes.
func invalidateDoubleDashesInName(obj client.Object, l logr.Logger) error {
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

func capName(obj client.Object, l logr.Logger, length int) error {
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
