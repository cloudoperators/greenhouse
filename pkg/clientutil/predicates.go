// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
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

func PredicateHasFinalizer(finalizer string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return controllerutil.ContainsFinalizer(o, finalizer)
	})
}

func PredicateHasOICDConfigured() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		org, ok := o.(*greenhousev1alpha1.Organization)
		if !ok {
			return false
		}

		if org.Spec.Authentication == nil || org.Spec.Authentication.OIDCConfig == nil {
			return false
		}

		return true
	})
}
