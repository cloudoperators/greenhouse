// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package predicates

import (
	"slices"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	}
}

func PredicatePluginWithStatusChange() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldPlugin, okOld := e.ObjectOld.(*greenhousev1alpha1.Plugin)
			newPlugin, okNew := e.ObjectNew.(*greenhousev1alpha1.Plugin)
			return okOld && okNew && !equality.Semantic.DeepEqual(oldPlugin.Status, newPlugin.Status)
		},
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

// PredicateClusterReadyStatusChange triggers on updates where a Cluster's Ready condition transitions to True.
func PredicateClusterReadyStatusChange() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			oldCluster, okOld := e.ObjectOld.(*greenhousev1alpha1.Cluster)
			newCluster, okNew := e.ObjectNew.(*greenhousev1alpha1.Cluster)
			if !okOld || !okNew {
				return false
			}
			oldCondition := oldCluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			newCondition := newCluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			if newCondition == nil {
				return false
			}
			return (oldCondition == nil || !oldCondition.IsTrue()) && newCondition.IsTrue()
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

// PredicateFluxResourceReadyChanged fires only when the Flux resource Ready condition status changes.
// to avoid re-enqueuing the owner on every Flux status tick.
func PredicateFluxResourceReadyChanged() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(_ event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				return true
			}
			oldObj, okOld := e.ObjectOld.(lifecycle.CatalogObject)
			newObj, okNew := e.ObjectNew.(lifecycle.CatalogObject)
			if !okOld || !okNew {
				return false
			}
			oldReady := meta.FindStatusCondition(oldObj.GetConditions(), fluxmeta.ReadyCondition)
			newReady := meta.FindStatusCondition(newObj.GetConditions(), fluxmeta.ReadyCondition)
			if oldReady == nil && newReady == nil {
				return false
			}
			if oldReady == nil || newReady == nil {
				return true
			}
			return oldReady.Status != newReady.Status || oldReady.Reason != newReady.Reason
		},
	}
}

// PredicateFluxHelmReleaseStatus fires when a HelmRelease's Ready or Released condition changes.
// The Plugin reconciler surfaces both: Ready for deploy health and Released for uninstall failures.
// While the HelmRelease is terminating it always fires, so the owner observes the uninstall through
// to completion and can release its own finalizer.
func PredicateFluxHelmReleaseStatus() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(_ event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				return true
			}
			oldObj, okOld := e.ObjectOld.(*helmv2.HelmRelease)
			newObj, okNew := e.ObjectNew.(*helmv2.HelmRelease)
			if !okOld || !okNew {
				return false
			}
			for _, condType := range []string{fluxmeta.ReadyCondition, helmv2.ReleasedCondition} {
				if fluxConditionChanged(oldObj.GetConditions(), newObj.GetConditions(), condType) {
					return true
				}
			}
			return false
		},
	}
}

func fluxConditionChanged(oldConditions, newConditions []metav1.Condition, condType string) bool {
	o := meta.FindStatusCondition(oldConditions, condType)
	n := meta.FindStatusCondition(newConditions, condType)
	if o == nil && n == nil {
		return false
	}
	if o == nil || n == nil {
		return true
	}
	return o.Status != n.Status || o.Reason != n.Reason || o.Message != n.Message
}
