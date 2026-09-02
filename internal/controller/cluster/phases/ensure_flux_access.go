// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"encoding/base64"
	"fmt"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const fluxAccessProviderGeneric = "generic"

func buildFluxAccessData(clusterSecret *corev1.Secret) (map[string]string, error) {
	apiServerURL := clusterSecret.GetAnnotations()[greenhouseapis.SecretAPIServerURLAnnotation]
	if apiServerURL == "" {
		return nil, fmt.Errorf("secret %s/%s is missing the %s annotation",
			clusterSecret.GetNamespace(), clusterSecret.GetName(), greenhouseapis.SecretAPIServerURLAnnotation)
	}

	// The CA is stored base64-encoded in the Secret data.
	caPEM, err := base64.StdEncoding.DecodeString(string(clusterSecret.Data[greenhouseapis.SecretAPIServerCAKey]))
	if err != nil {
		return nil, fmt.Errorf("failed decoding %s of secret %s/%s: %w",
			greenhouseapis.SecretAPIServerCAKey, clusterSecret.GetNamespace(), clusterSecret.GetName(), err)
	}

	return map[string]string{
		fluxmeta.KubeConfigKeyProvider:           fluxAccessProviderGeneric,
		fluxmeta.KubeConfigKeyAddress:            apiServerURL,
		fluxmeta.KubeConfigKeyAudiences:          greenhouseapis.OIDCAudience,
		fluxmeta.KubeConfigKeyServiceAccountName: clusterSecret.GetName(),
		fluxmeta.KubeConfigKeyCACert:             string(caPEM),
	}, nil
}

// ensureFluxAccess renders the ConfigMap Flux uses to reach an OIDC-onboarded cluster.
func (p *Phase) ensureFluxAccess(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		data, err := buildFluxAccessData(p.ClusterSecret)
		if err != nil {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.FluxAccessReady, "", err.Error()))
			return lifecycle.Break(), err
		}

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: cluster.GetName(), Namespace: cluster.GetNamespace()},
		}
		result, err := controllerutil.CreateOrUpdate(ctx, p.Client, configMap, func() error {
			configMap.Data = data
			return controllerutil.SetOwnerReference(cluster, configMap, p.Client.Scheme())
		})
		if err != nil {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.FluxAccessReady, "", err.Error()))
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("created flux access configmap", "namespace", configMap.GetNamespace(), "name", configMap.GetName())
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated flux access configmap", "namespace", configMap.GetNamespace(), "name", configMap.GetName())
		}

		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.FluxAccessReady, "", "Flux can reach the cluster via the access ConfigMap"))
		return lifecycle.Continue(), nil
	}
}
