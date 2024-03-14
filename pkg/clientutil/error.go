// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import apierrors "k8s.io/apimachinery/pkg/api/errors"

// IgnoreAlreadyExists returns nil on IsAlreadyExists errors.
func IgnoreAlreadyExists(err error) error {
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
