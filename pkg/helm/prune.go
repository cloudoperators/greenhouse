// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		annotations := metaObject.GetAnnotations()
		if annotations != nil {
			delete(annotations, corev1.LastAppliedConfigAnnotation)
		}
		metaObject.SetAnnotations(annotations)
		return o
	}
)
