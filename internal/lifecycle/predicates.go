// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func IgnoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil {
				return false
			}
			if e.ObjectNew == nil {
				return false
			}
			if !cmp.Equal(e.ObjectOld.GetAnnotations(), e.ObjectNew.GetAnnotations()) || !cmp.Equal(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels()) {
				return true
			}
			oldObj := e.ObjectOld.(RuntimeObject)
			newObj := e.ObjectNew.(RuntimeObject)
			conditionsEqual := cmp.Equal(oldObj.GetConditions(), newObj.GetConditions())
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.ObjectNew.GetGeneration() == e.ObjectOld.GetGeneration() && conditionsEqual && reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers()) {
				// On delete event setupStatus changes to Deleted and deletion timestamp is set
				// In such cases return true else false
				return e.ObjectNew.GetDeletionTimestamp() != nil
			}
			return true
		},
	}
}
