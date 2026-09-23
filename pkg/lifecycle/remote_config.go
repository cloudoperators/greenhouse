// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"encoding/base64"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func NewRemoteKubeCfg(ctx context.Context, kubeClient client.Client, cluster *greenhouseapisv1alpha1.Cluster) (*rest.Config, error) {
	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, secret); err != nil {
		return nil, err
	}
	if cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] == greenhouseapis.ClusterWorkloadIdentityEnabled {
		return getWorkloadIdentityCfg(ctx, kubeClient, secret)
	}
	return getKubeCfg(secret)
}

func getWorkloadIdentityCfg(ctx context.Context, kubeClient client.Client, secret *corev1.Secret) (*rest.Config, error) {
	certDecoded, err := base64.StdEncoding.DecodeString(string(secret.Data[greenhouseapis.SecretAPIServerCAKey]))
	if err != nil {
		return nil, fmt.Errorf("failed decoding certificate data: %w", err)
	}

	tokenRequest, err := MintServiceAccountToken(ctx, kubeClient, secret.GetNamespace(), secret.GetName())
	if err != nil {
		return nil, err
	}

	cfg := &rest.Config{
		Host:        secret.Annotations[greenhouseapis.SecretAPIServerURLAnnotation],
		BearerToken: tokenRequest.Status.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: certDecoded,
		},
	}
	return cfg, nil
}

func MintServiceAccountToken(ctx context.Context, kubeClient client.Client, namespace, name string) (*authenticationv1.TokenRequest, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{greenhouseapis.OIDCAudience},
			ExpirationSeconds: ptr.To[int64](600),
		},
	}
	if err := kubeClient.SubResource("token").Create(ctx, serviceAccount, tokenRequest); err != nil {
		return nil, fmt.Errorf("failed creating token request for workload identity: %w", err)
	}
	return tokenRequest, nil
}

func getKubeCfg(secret *corev1.Secret) (*rest.Config, error) {
	var kubeconfigBytes []byte
	switch {
	case secret.Type != greenhouseapis.SecretTypeKubeConfig && secret.Type != greenhouseapis.SecretTypeOIDCConfig:
		return nil, fmt.Errorf("secret %s/%s is not of type %s", secret.GetNamespace(), secret.GetName(), greenhouseapis.SecretTypeKubeConfig)
	case isSecretContainsKey(secret, greenhouseapis.GreenHouseKubeConfigKey):
		kubeconfigBytes = secret.Data[greenhouseapis.GreenHouseKubeConfigKey]
	case isSecretContainsKey(secret, greenhouseapis.KubeConfigKey):
		kubeconfigBytes = secret.Data[greenhouseapis.KubeConfigKey]
	default:
		return nil, fmt.Errorf("secret %s/%s does not contain a kubeconfig", secret.GetNamespace(), secret.GetName())
	}
	return clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
}

// isSecretContainsKey checks whether the given secret contains a key.
func isSecretContainsKey(s *corev1.Secret, key string) bool {
	if s.Data == nil {
		return false
	}
	v, ok := s.Data[key]
	return ok && v != nil
}
