// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import "sigs.k8s.io/controller-runtime/pkg/client"

// EnsureAnnotation sets the annotation key to value on the object.
// if the value is an empty string, the annotation is removed.
func EnsureAnnotation(obj client.Object, key, value string) {
	if value == "" {
		delete(obj.GetAnnotations(), key)
		return
	}
	a := obj.GetAnnotations()
	if a == nil {
		a = make(map[string]string, 1)
	}
	a[key] = value
	obj.SetAnnotations(a)
}
