// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OperationResult is the action result of a CreateOrUpdate call
type OperationResult string

const ( // They should complete the sentence "Deployment default/foo has been ..."
	// OperationResultNone means that the resource has not been changed
	OperationResultNone OperationResult = "unchanged"
	// OperationResultCreated means that a new resource is created
	OperationResultCreated OperationResult = "created"
	// OperationResultUpdated means that an existing resource is updated
	OperationResultUpdated OperationResult = "updated"
)

// CreateOrPatch creates or patches the given object using the mutate function. Returns the result and an error.
func CreateOrPatch(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	if err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return OperationResultNone, err
		}
		if err := mutate(); err != nil {
			return OperationResultNone, errors.Wrap(err, "mutating object failed")
		}
		if err := c.Create(ctx, obj); err != nil {
			return OperationResultNone, IgnoreAlreadyExists(err)
		}
		return OperationResultCreated, nil
	}
	if o, err := meta.Accessor(obj); err == nil {
		if o.GetDeletionTimestamp() != nil {
			return OperationResultNone, fmt.Errorf("the resource %s/%s already exists but is marked for deletion", o.GetNamespace(), o.GetName())
		}
	}

	return patch(ctx, c, obj, mutate, false)
}

func CreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	if err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return OperationResultNone, err
		}
		if err := mutate(); err != nil {
			return OperationResultNone, errors.Wrap(err, "mutating object failed")
		}
		if err := c.Create(ctx, obj); err != nil {
			return OperationResultNone, IgnoreAlreadyExists(err)
		}
		return OperationResultCreated, nil
	}
	if o, err := meta.Accessor(obj); err == nil {
		if o.GetDeletionTimestamp() != nil {
			return OperationResultNone, fmt.Errorf("the resource %s/%s already exists but is marked for deletion", o.GetNamespace(), o.GetName())
		}
	}
	return update(ctx, c, obj, mutate)
}

// Patch uses a PATCH operation on an object. Returns the Result or an error.
func Patch(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	return patch(ctx, c, obj, mutate, false)
}

// PatchStatus uses a PATCH operation on an object status. Returns the Result or an error.
func PatchStatus(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	return patch(ctx, c, obj, mutate, true)
}

func patch(ctx context.Context, c client.Client, obj client.Object, mutate func() error, status bool) (OperationResult, error) {
	before := obj.DeepCopyObject().(client.Object) //nolint:errcheck
	if err := mutate(); err != nil {
		return OperationResultNone, errors.Wrap(err, "mutating object failed")
	}
	if equality.Semantic.DeepEqual(before, obj) {
		return OperationResultNone, nil
	}
	patch := client.MergeFrom(before)
	logPatch(ctx, obj, patch)

	var err error
	if status {
		err = c.Status().Patch(ctx, obj, patch)
	} else {
		err = c.Patch(ctx, obj, patch)
	}
	if err != nil {
		return OperationResultNone, err
	}

	return OperationResultUpdated, nil
}

func update(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	if err := mutate(); err != nil {
		return OperationResultNone, errors.Wrap(err, "mutating object failed")
	}
	err := c.Update(ctx, obj)
	if err != nil {
		return OperationResultNone, err
	}
	return OperationResultUpdated, nil
}

func logPatch(ctx context.Context, obj client.Object, patch client.Patch) {
	patchData, err := patch.Data(obj)
	if err != nil {
		// Ignore the error and omit the log.
		return
	}
	patchDataString := string(patchData)
	if obj.GetObjectKind().GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("Secret") {
		patchDataString = "[REDACTED]"
	}
	objKey := fmt.Sprintf("%s/%s/%s",
		obj.GetObjectKind().GroupVersionKind().GroupVersion().String(), obj.GetObjectKind().GroupVersionKind().Kind, client.ObjectKeyFromObject(obj).String(),
	)
	log.FromContext(ctx).V(5).Info("patching object",
		"object", objKey, "type", string(patch.Type()), "patch", patchDataString,
	)
}
