// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OperationResult is the action result of a Patch call
type OperationResult string

const (
	// OperationResultNone means that the resource has not been changed
	OperationResultNone OperationResult = "unchanged"
	// OperationResultUpdated means that an existing resource is updated
	OperationResultUpdated OperationResult = "updated"
)

// Patch uses a PATCH operation on an object. Returns the Result or an error.
func Patch(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	return patch(ctx, c, obj, mutate, false)
}

// PatchStatus uses a PATCH operation on an object status. Returns the Result or an error.
func PatchStatus(ctx context.Context, c client.Client, obj client.Object, mutate func() error) (OperationResult, error) {
	return patch(ctx, c, obj, mutate, true)
}

func patch(ctx context.Context, c client.Client, obj client.Object, mutate func() error, status bool) (OperationResult, error) {
	before := obj.DeepCopyObject().(client.Object)
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
