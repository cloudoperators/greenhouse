// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/pkg/errors"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

type TokenHelper struct {
	InClusterClient                    client.Client
	RemoteClusterClient                client.Client
	Proxy                              string
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
	SecretType                         corev1.SecretType
	OIDCServiceAccount                 string
}

type KubeConfigHelper struct {
	Host           string
	CAData         []byte
	BearerToken    string
	Username       string
	Namespace      string
	TLSServerName  string
	ProxyURL       string
	ClientCertData []byte
	ClientKeyData  []byte
}

type claims struct {
	Issuer    string   `json:"iss,omitempty"`
	Subject   string   `json:"sub,omitempty"`
	Audience  []string `json:"aud,omitempty"`
	Expiry    int64    `json:"exp,omitempty"`
	NotBefore int64    `json:"nbf,omitempty"`
	IssuedAt  int64    `json:"iat,omitempty"`
	ID        string   `json:"jti,omitempty"`
}

// RestConfigToAPIConfig converts a rest config to a clientcmdapi.Config
func (kubeconfig *KubeConfigHelper) RestConfigToAPIConfig(clusterName string) clientcmdapi.Config {
	clientConfig := clientcmdapi.NewConfig()
	clientConfig.Clusters[clusterName] = &clientcmdapi.Cluster{
		Server:                   kubeconfig.Host,
		CertificateAuthorityData: kubeconfig.CAData,
		TLSServerName:            kubeconfig.TLSServerName,
		ProxyURL:                 kubeconfig.ProxyURL,
	}
	clientConfig.Contexts[clusterName] = &clientcmdapi.Context{
		Cluster:   clusterName,
		AuthInfo:  kubeconfig.Username,
		Namespace: kubeconfig.Namespace,
	}
	clientConfig.CurrentContext = clusterName
	if kubeconfig.BearerToken != "" {
		clientConfig.AuthInfos[kubeconfig.Username] = &clientcmdapi.AuthInfo{
			Token: kubeconfig.BearerToken,
		}
	}
	if kubeconfig.ClientCertData != nil && kubeconfig.ClientKeyData != nil {
		clientConfig.AuthInfos[kubeconfig.Username] = &clientcmdapi.AuthInfo{
			ClientCertificateData: kubeconfig.ClientCertData,
			ClientKeyData:         kubeconfig.ClientKeyData,
		}
	}
	return *clientConfig
}

// GenerateTokenRequest reconciles the service account token for the remote cluster and updates the secret containing the kubeconfig
func (t *TokenHelper) GenerateTokenRequest(ctx context.Context, restClientGetter *clientutil.RestClientGetter, cluster *greenhousev1alpha1.Cluster) (*authenticationv1.TokenRequest, error) {
	var jwtPayload []byte
	var tokenInfo = &claims{}
	var actualTokenExpiry metav1.Time

	remoteRestConfig, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	jwt, err := jose.ParseSigned(remoteRestConfig.BearerToken, getAllSignatureAlgorithms())
	// If parsing the token is not possible we fall back to the old way of checking the token expiration
	if err != nil {
		if !cluster.Status.BearerTokenExpirationTimestamp.IsZero() && cluster.Status.BearerTokenExpirationTimestamp.Time.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
			log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
			return nil, nil
		}
	}

	if jwt != nil {
		jwtPayload = jwt.UnsafePayloadWithoutVerification()
	}
	err = json.Unmarshal(jwtPayload, &tokenInfo)
	// If parsing the token is not possible we fall back to the old way of checking the token expiration
	if err == nil {
		actualTokenExpiry = metav1.Unix(tokenInfo.Expiry, 0)
		if actualTokenExpiry.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
			log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
			return nil, nil
		}
	} else if !cluster.Status.BearerTokenExpirationTimestamp.IsZero() && cluster.Status.BearerTokenExpirationTimestamp.Time.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
		log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
		return nil, nil
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(int64(t.RemoteClusterBearerTokenValidity / time.Second)),
		},
	}
	// handle token request based on secret type
	switch t.SecretType {
	case greenhouseapis.SecretTypeKubeConfig:
		tokenRequest, err = t.generateKubeConfigToken(ctx, tokenRequest, cluster)
	case greenhouseapis.SecretTypeOIDCConfig:
		tokenRequest, err = t.generateOIDCToken(ctx, tokenRequest, cluster)
	}
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to generate token", "cluster", cluster.Name, "secretType", t.SecretType)
		return nil, err
	}
	return tokenRequest, nil
}

// generateKubeConfigToken - generates a service account token using the remote cluster client.Client
func (t *TokenHelper) generateKubeConfigToken(ctx context.Context, tokenRequest *authenticationv1.TokenRequest, cluster *greenhousev1alpha1.Cluster) (*authenticationv1.TokenRequest, error) {
	serviceAccount := NewServiceAccount(ServiceAccountName, cluster.GetNamespace())
	err := t.RemoteClusterClient.SubResource("token").Create(ctx, serviceAccount, tokenRequest)
	if err != nil {
		return nil, err
	}
	return tokenRequest, nil
}

// generateOIDCToken generates a service account token using the inCluster client.Client with an audience
func (t *TokenHelper) generateOIDCToken(ctx context.Context, tokenRequest *authenticationv1.TokenRequest, cluster *greenhousev1alpha1.Cluster) (*authenticationv1.TokenRequest, error) {
	tokenRequest.Spec.Audiences = []string{greenhouseapis.OIDCAudience}
	serviceAccount := NewServiceAccount(cluster.GetName(), cluster.GetNamespace())
	err := t.InClusterClient.SubResource("token").Create(ctx, serviceAccount, tokenRequest)
	if err != nil {
		return nil, err
	}
	return tokenRequest, nil
}

// GenerateNewClientKubeConfig generates a kubeconfig for the client to access the cluster from REST config coming from the secret
func GenerateNewClientKubeConfig(restConfigGetter *clientutil.RestClientGetter, bearerToken string, cluster *greenhousev1alpha1.Cluster) ([]byte, error) {
	restConfig, err := restConfigGetter.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load kube clientConfig for cluster %s", cluster.GetName())
	}
	// TODO: replace overwrite with https://github.com/kubernetes/kubernetes/pull/119398 after 1.30 upgrade
	kubeConfigGenerator := &KubeConfigHelper{
		Host:        restConfig.Host,
		CAData:      restConfig.CAData,
		BearerToken: bearerToken,
		Username:    ServiceAccountName,
		Namespace:   cluster.GetNamespace(),
	}
	kubeconfigByte, err := clientcmd.Write(kubeConfigGenerator.RestConfigToAPIConfig(cluster.Name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate kubeconfig for cluster %s", cluster.GetName())
	}
	return kubeconfigByte, nil
}
