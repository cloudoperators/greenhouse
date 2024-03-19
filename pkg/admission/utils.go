// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
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
	if c.validateCreate == nil {
		return nil, nil
	}
	return c.validateCreate(ctx, c.Client, obj)
}

func (c *customValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	if c.validateUpdate == nil {
		return nil, nil
	}
	return c.validateUpdate(ctx, c.Client, oldObj, newObj)
}

func (c *customValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	if c.validateDelete == nil {
		return nil, nil
	}
	return c.validateDelete(ctx, c.Client, obj)
}
