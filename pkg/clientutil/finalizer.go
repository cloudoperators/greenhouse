// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureFinalizer ensures a finalizer is present on the object. Returns an error on failure.
func EnsureFinalizer(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if controllerutil.ContainsFinalizer(o, finalizer) {
		return nil
	}
	_, err := Patch(ctx, c, o, func() error {
		controllerutil.AddFinalizer(o, finalizer)
		return nil
	})
	return err
}

// RemoveFinalizer removes a finalizer from an object. Returns an error on failure.
func RemoveFinalizer(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if !controllerutil.ContainsFinalizer(o, finalizer) {
		return nil
	}
	_, err := Patch(ctx, c, o, func() error {
		controllerutil.RemoveFinalizer(o, finalizer)
		return nil
	})
	return err
}

func HasFinalizer(o client.Object, finalizer string) bool {
	return controllerutil.ContainsFinalizer(o, finalizer)
}
