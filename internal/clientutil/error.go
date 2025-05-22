// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// IgnoreAlreadyExists returns nil on IsAlreadyExists errors.
func IgnoreAlreadyExists(err error) error {
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// IgnoreIndexerConflict returns nil on indexer conflict errors.
// This is used to ignore errors when the indexer is already set up.
func IgnoreIndexerConflict(err error) error {
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), "indexer conflict") {
		return nil
	}
	return err
}
