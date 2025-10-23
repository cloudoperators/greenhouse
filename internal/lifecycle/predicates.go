// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// IgnoreStatusUpdatePredicate returns a predicate that filters out update events
// where only the status, finalizers, annotations, or labels have changed.
func IgnoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			// Reconcile if annotations/labels changed.
			if !cmp.Equal(e.ObjectOld.GetAnnotations(), e.ObjectNew.GetAnnotations()) ||
				!cmp.Equal(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels()) {
				return true
			}
			conditionsEqual := conditionsEqual(e.ObjectOld, e.ObjectNew)
			// Ignore pure status updates (generation unchanged, conditions equal, finalizers equal).
			if e.ObjectNew.GetGeneration() == e.ObjectOld.GetGeneration() &&
				conditionsEqual &&
				reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers()) {
				// But let delete-in-progress through.
				return e.ObjectNew.GetDeletionTimestamp() != nil
			}
			return true
		},
	}
}

func conditionsEqual(oldObj, newObj client.Object) bool {
	// flux obj: []v1.Condition
	if fluxOldObj, ok := oldObj.(CatalogObject); ok {
		if fluxNewObj, ok2 := newObj.(CatalogObject); ok2 {
			return cmp.Equal(fluxOldObj.GetConditions(), fluxNewObj.GetConditions())
		}
		// kind mismatch
		return false
	}

	// Greenhouse: StatusConditions
	if oldGhObj, ok := oldObj.(RuntimeObject); ok {
		if newGhObj, ok2 := newObj.(RuntimeObject); ok2 {
			return cmp.Equal(oldGhObj.GetConditions(), newGhObj.GetConditions())
		}
		return false
	}
	// No recognizable conditions provider
	return false
}
