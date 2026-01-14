// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"errors"
)

type reconcileRunKey struct{}

type reconcileRun struct {
	objectCopy RuntimeObject
}

// createContextFromRuntimeObject create a new context with a copy of the object attached.
func createContextFromRuntimeObject(ctx context.Context, object RuntimeObject) context.Context {
	return context.WithValue(ctx, reconcileRunKey{}, &reconcileRun{
		objectCopy: object.DeepCopyObject().(RuntimeObject), //nolint:errcheck
	})
}

func getRunFromContext(ctx context.Context) (*reconcileRun, error) {
	val, ok := ctx.Value(reconcileRunKey{}).(*reconcileRun)
	if !ok {
		return nil, errors.New("could  not extract *reconcileRun from given context")
	}

	return val, nil
}

// getOriginalResourceFromContext - returns the unmodified version of the RuntimeObject
func getOriginalResourceFromContext(ctx context.Context) (RuntimeObject, error) {
	reconcileRun, err := getRunFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// create another copy so that context can not be modified by accident
	return reconcileRun.objectCopy.DeepCopyObject().(RuntimeObject), nil //nolint:errcheck
}
