// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// PredicateFilterBySecretTypes filters secrets by the given types.
func PredicateFilterBySecretTypes(secretTypes ...corev1.SecretType) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		if secret, ok := o.(*corev1.Secret); ok {
			return slices.Contains(secretTypes, secret.Type)
		}
		return false
	})
}

func PredicateClusterByAccessMode(accessMode greenhousev1alpha1.ClusterAccessMode) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		if cluster, ok := o.(*greenhousev1alpha1.Cluster); ok {
			return cluster.Spec.AccessMode == accessMode
		}
		return false
	})
}

func PredicateClusterIsReady() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		cluster, ok := o.(*greenhousev1alpha1.Cluster)
		if !ok {
			return false
		}
		return cluster.Status.IsReadyTrue()
	})
}

func PredicateByName(name string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == name
	})
}

func PredicateHasLabelWithValue(key, value string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetLabels()[key] == value
	})
}

func PredicatePluginWithStatusReadyChange() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			oldPlugin, okOld := e.ObjectOld.(*greenhousev1alpha1.Plugin)
			newPlugin, okNew := e.ObjectNew.(*greenhousev1alpha1.Plugin)
			if !okOld || !okNew {
				return false
			}
			oldReadyCondition := oldPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			newReadyCondition := newPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			if oldReadyCondition == nil && newReadyCondition == nil {
				return false
			}
			if oldReadyCondition == nil || newReadyCondition == nil {
				return true
			}
			return oldReadyCondition.Status != newReadyCondition.Status
		},
		DeleteFunc:  func(_ event.DeleteEvent) bool { return false },
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	}
}

func PredicateOrganizationSCIMStatusChange() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			oldOrg, okOld := e.ObjectOld.(*greenhousev1alpha1.Organization)
			newOrg, okNew := e.ObjectNew.(*greenhousev1alpha1.Organization)
			if !okOld || !okNew {
				return false
			}
			oldCondition := oldOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
			newCondition := newOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
			if newCondition == nil {
				return false
			}
			return (oldCondition == nil || oldCondition.IsFalse()) && newCondition.IsTrue() // check is the SCIMAPIAvailableCondition condition is flip to true
		},
		DeleteFunc:  func(_ event.DeleteEvent) bool { return false },
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	}
}

func PredicateIgnoreDeletingResources() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetDeletionTimestamp().IsZero()
		},
	}
}
