// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// PredicatePluginWithHelmSpec filters Plugins without an HelmChart specification.
var PredicatePluginWithHelmSpec = func() predicate.Funcs {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		if plugin, ok := o.(*greenhousev1alpha1.Plugin); ok {
			return plugin.Spec.HelmChart != nil
		}
		return false
	})
}

// PredicateFilterBySecretType filters secrets by the given type.
func PredicateFilterBySecretType(secretType corev1.SecretType) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		if secret, ok := o.(*corev1.Secret); ok {
			return secret.Type == secretType
		}
		return false
	})
}

// PredicateSecretContainsKey filters secrets by the given key.
func PredicateSecretContainsKey(key string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		s, ok := o.(*corev1.Secret)
		if !ok {
			return false
		}
		return IsSecretContainsKey(s, key)
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

func PredicateByName(name string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == name
	})
}

// LabelSelectorPredicate constructs a Predicate from a LabelSelector.
// Only objects matching the LabelSelector will be admitted.
// Credit https://github.com/kubernetes-sigs/controller-runtime/blob/v0.10.1/pkg/predicate/predicate.go#L323-L333.
func LabelSelectorPredicate(s metav1.LabelSelector) predicate.Predicate {
	selector, err := metav1.LabelSelectorAsSelector(&s)
	if err != nil {
		return predicate.Funcs{}
	}
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return selector.Matches(labels.Set(o.GetLabels()))
	})
}
