// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// useFluxAccessConfigMap reports whether the HelmRelease should use the Flux access
// ConfigMap. It falls back to the kubeconfig Secret while the ConfigMap is absent.
func useFluxAccessConfigMap(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (bool, error) {
	if plugin.Spec.ClusterName == "" {
		return false, nil
	}

	cluster := &greenhousev1alpha1.Cluster{}
	switch err := c.Get(ctx, types.NamespacedName{Name: plugin.Spec.ClusterName, Namespace: plugin.Namespace}, cluster); {
	case apierrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	if cluster.Annotations[greenhouseapis.ClusterConnectivityAnnotation] != greenhouseapis.ClusterConnectivityOIDC {
		return false, nil
	}

	configMap := &corev1.ConfigMap{}
	switch err := c.Get(ctx, types.NamespacedName{Name: plugin.Spec.ClusterName, Namespace: plugin.Namespace}, configMap); {
	case apierrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
