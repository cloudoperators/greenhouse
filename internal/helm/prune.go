// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type pruneFunc func(o runtime.Object) runtime.Object

var (
	// Prune the managedFields in the object to obfuscating the diff.
	pruneManagedFields pruneFunc = func(o runtime.Object) runtime.Object {
		metaObject, err := meta.Accessor(o)
		if err != nil {
			// The object might not declare the required metadata in which case the original object is returned.
			return o
		}
		metaObject.SetManagedFields(nil)
		return o
	}

	// Prune the last-applied annotation in the object to obfuscating the diff.
	pruneLastAppliedAnnotation pruneFunc = func(o runtime.Object) runtime.Object {
		metaObject, err := meta.Accessor(o)
		if err != nil {
			// The object might not declare the required metadata in which case the original object is returned.
			return o
		}
		delete(metaObject.GetAnnotations(), corev1.LastAppliedConfigAnnotation)

		return o
	}
)
