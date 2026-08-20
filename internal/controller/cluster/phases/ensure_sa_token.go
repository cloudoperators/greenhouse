// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureServiceAccountToken(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		cluster.SetDefaultTokenValidityIfNeeded()

		t := &utils.TokenHelper{
			InClusterClient:                    p.Client,
			RemoteClusterClient:                p.RemoteClient,
			RemoteClusterBearerTokenValidity:   time.Duration(cluster.Spec.KubeConfig.MaxTokenValidity) * time.Hour,
			RenewRemoteClusterBearerTokenAfter: p.RenewRemoteClusterBearerTokenAfter,
			SecretType:                         p.ClusterSecret.Type,
		}

		tokenRequest, err := t.GenerateTokenRequest(ctx, p.RestClientGetter, cluster, p.ClusterSecret)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to generate token", "cluster", cluster.Name)
			return lifecycle.Break(), err
		}
		if tokenRequest == nil {
			// token still valid, nothing to do
			return lifecycle.Continue(), nil
		}

		generatedKubeConfig, err := utils.GenerateNewClientKubeConfig(p.RestClientGetter, tokenRequest.Status.Token, cluster)
		if err != nil {
			return lifecycle.Break(), err
		}

		kubeConfigSecret, err := p.getKubeConfigSecret(ctx, cluster)
		if err != nil {
			return lifecycle.Break(), err
		}

		result, err := controllerutil.CreateOrUpdate(ctx, p.Client, kubeConfigSecret, func() error {
			if kubeConfigSecret.Annotations == nil {
				kubeConfigSecret.Annotations = make(map[string]string)
			}
			if kubeConfigSecret.Data == nil {
				kubeConfigSecret.Data = make(map[string][]byte)
			}
			if p.ClusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
				kubeConfigSecret.Annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation] = metav1.Now().Format(time.DateTime)
			}
			kubeConfigSecret.Data[greenhouseapis.GreenHouseKubeConfigKey] = generatedKubeConfig
			return nil
		})
		if err != nil {
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("created kubeconfig secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated kubeconfig secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		}

		cluster.Status.BearerTokenExpirationTimestamp = tokenRequest.Status.ExpirationTimestamp
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) getKubeConfigSecret(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (*corev1.Secret, error) {
	s := &corev1.Secret{}
	if err := p.Client.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetName()}, s); err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}
	return s, nil
}
