// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeletionResult is the action result of a Delete call
type DeletionResult string

const ( // They should complete the sentence "Deployment default/foo has been ..."
	// DeletionResultDeleted means that the resource has been deleted
	DeletionResultDeleted DeletionResult = "deleted"
	// DeletionResultNotFound means that the resource was not found
	DeletionResultNotFound DeletionResult = "not found"
	// DeletionResultNone means that the resource has not been changed
	DeletionResultNone DeletionResult = "unchanged"
)

// Delete deletes the object if it exists, otherwise it does nothing. Returns the result and an error.
func Delete(ctx context.Context, c client.Client, obj client.Object) (DeletionResult, error) {
	key := client.ObjectKeyFromObject(obj)

	err := c.Get(ctx, key, obj)
	switch {
	case apierrors.IsNotFound(err):
		return DeletionResultNotFound, nil
	case err != nil:
		return DeletionResultNone, err
	}
	if err := c.Delete(ctx, obj); err != nil {
		return DeletionResultNone, err
	}
	return DeletionResultDeleted, nil
}
