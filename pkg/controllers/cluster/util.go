// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-jose/go-jose/v4"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

var (
	defaultRequeueInterval = 10 * time.Minute
	allSignatureAlgorithms = []jose.SignatureAlgorithm{
		jose.EdDSA,
		jose.HS256,
		jose.HS384,
		jose.HS512,
		jose.RS256,
		jose.RS384,
		jose.RS512,
		jose.ES256,
		jose.ES384,
		jose.ES512,
		jose.PS256,
		jose.PS384,
		jose.PS512,
	}
)

const (
	CRoleKind = "ClusterRole"
	CRoleRef  = "cluster-admin"
)

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
	Issuer     string           `json:"iss,omitempty"`
	Subject    string           `json:"sub,omitempty"`
	Audience   []string         `json:"aud,omitempty"`
	Expiry     int64            `json:"exp,omitempty"`
	NotBefore  int64            `json:"nbf,omitempty"`
	IssuedAt   int64            `json:"iat,omitempty"`
	ID         string           `json:"jti,omitempty"`
	Kubernetes kubernetesClaims `json:"kubernetes.io,omitempty"`
}

type kubernetesClaims struct {
	Namespace string `json:"namespace,omitempty"`
	Svcacct   ref    `json:"serviceaccount,omitempty"`
	Pod       *ref   `json:"pod,omitempty"`
	Secret    *ref   `json:"secret,omitempty"`
	Node      *ref   `json:"node,omitempty"`
	WarnAfter int64  `json:"warnafter,omitempty"`
}

type ref struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
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

func reconcileNamespaceInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	var namespace = new(corev1.Namespace)
	namespace.Name = cluster.GetNamespace()
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, namespace, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created namespace", "cluster", cluster.Name, "namespace", namespace.Name)
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated namespace", "cluster", cluster.Name, "namespace", namespace.Name)
		// TODO: emit event on cluster
	}
	return nil
}

func reconcileServiceAccountInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: cluster.GetNamespace(),
		},
	}
	crb := &rbacv1.ClusterRoleBinding{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: serviceAccountName}, crb)
	if err != nil {
		return err
	}
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, serviceAccount, func() error {
		return controllerutil.SetOwnerReference(crb, serviceAccount, k8sClient.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created serviceAccount", "cluster", serviceAccount.Name)
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated serviceAccount", "cluster", serviceAccount.Name)
		// TODO: emit event on cluster
	}
	return nil
}

func deleteClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccountName,
		},
	}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: serviceAccountName}, crb)
	if err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "potential err getting clusterRoleBinding")
		if apierrors.IsUnauthorized(err) || apierrors.IsNotFound(err) || apierrors.IsForbidden(err) {
			return nil
		}
		return err
	}
	err = k8sClient.Delete(ctx, crb)
	// ignore not found and forbidden errors
	if err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "potential err deleting clusterRoleBinding")
		if !apierrors.IsUnauthorized(err) && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
			return err
		}
	}
	return nil
}

func reconcileClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccountName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: cluster.GetNamespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     CRoleKind,
			Name:     CRoleRef,
			APIGroup: rbacv1.GroupName,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, clusterRoleBinding, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	// TODO: emit event on cluster
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created clusterRoleBinding", "cluster", clusterRoleBinding.Name)
	// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated clusterRoleBinding", "cluster", clusterRoleBinding.Name)
	}
	return nil
}

type tokenHelper struct {
	client.Client
	Proxy                              string
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
}

// ReconcileServiceAccountToken reconciles the service account token for the cluster and updates the secret containing the kube config
func (t *tokenHelper) ReconcileServiceAccountToken(ctx context.Context, restClientGetter *clientutil.RestClientGetter, cluster *greenhousev1alpha1.Cluster) error {
	var jwtPayload []byte
	var tokenInfo = new(claims)
	var actualTokenExpiry metav1.Time

	remoteRestConfig, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return err
	}
	jwt, err := jose.ParseSigned(remoteRestConfig.BearerToken, allSignatureAlgorithms)
	// If parsing the token is not possible we fall back to the old way of checking the token expiration
	if err != nil {
		if !cluster.Status.BearerTokenExpirationTimestamp.IsZero() && cluster.Status.BearerTokenExpirationTimestamp.Time.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
			log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
			return nil
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
			return nil
		}
	} else if !cluster.Status.BearerTokenExpirationTimestamp.IsZero() && cluster.Status.BearerTokenExpirationTimestamp.Time.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
		log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
		return nil
	}

	clientset, err := kubernetes.NewForConfig(remoteRestConfig)
	if err != nil {
		return err
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(int64(t.RemoteClusterBearerTokenValidity / time.Second)),
		},
	}
	tokenRequestResponse, err := clientset.
		CoreV1().
		ServiceAccounts(cluster.GetNamespace()).
		CreateToken(ctx, serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	var generatedKubeConfig []byte
	switch cluster.Spec.AccessMode {
	case greenhousev1alpha1.ClusterAccessModeDirect:
		generatedKubeConfig, err = generateNewClientKubeConfig(ctx, restClientGetter, tokenRequestResponse.Status.Token, cluster)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown access mode %s", cluster.Spec.AccessMode)
	}

	var kubeConfigSecret = new(corev1.Secret)
	if err := t.Get(ctx, types.NamespacedName{Namespace: cluster.GetNamespace(), Name: cluster.GetName()}, kubeConfigSecret); err != nil {
		return err
	}
	result, err := clientutil.CreateOrPatch(ctx, t.Client, kubeConfigSecret, func() error {
		kubeConfigSecret.Data[greenhouseapis.GreenHouseKubeConfigKey] = generatedKubeConfig
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated secret", "namespace", kubeConfigSecret.GetNamespace(), "name", kubeConfigSecret.GetName())
		// TODO: emit event on cluster
	}

	// TODO: Do not set the status here to avoid patching the cluster mid-way.
	// This should be done in the reconcileStatus() method of the respective cluster reconciler in the end.
	_, err = clientutil.PatchStatus(ctx, t.Client, cluster, func() error {
		cluster.Status.BearerTokenExpirationTimestamp = tokenRequestResponse.Status.ExpirationTimestamp
		return nil
	})
	return err
}

// generateNewClientKubeConfig generates a kubeconfig for the client to access the cluster from REST config coming from the secret
func generateNewClientKubeConfig(_ context.Context, restConfigGetter *clientutil.RestClientGetter, bearerToken string, cluster *greenhousev1alpha1.Cluster) ([]byte, error) {
	restConfig, err := restConfigGetter.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load kube clientConfig for cluster %s", cluster.GetName())
	}
	// TODO: replace overwrite with https://github.com/kubernetes/kubernetes/pull/119398 after 1.30 upgrade
	kubeConfigGenerator := &KubeConfigHelper{
		Host:        restConfig.Host,
		CAData:      restConfig.CAData,
		BearerToken: bearerToken,
		Username:    serviceAccountName,
		Namespace:   cluster.GetNamespace(),
	}
	kubeconfigByte, err := clientcmd.Write(kubeConfigGenerator.RestConfigToAPIConfig(cluster.Name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate kubeconfig for cluster %s", cluster.GetName())
	}
	return kubeconfigByte, nil
}

// reconcileRemoteAPIServerVersion fetches the api server version from the remote cluster and reflects it in the cluster CR
func reconcileRemoteAPIServerVersion(ctx context.Context, restConfigGetter *clientutil.RestClientGetter, k8sclient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	k8sVersion, err := clientutil.GetKubernetesVersion(restConfigGetter)
	if err != nil {
		return err
	}
	_, err = clientutil.PatchStatus(ctx, k8sclient, cluster, func() error {
		cluster.Status.KubernetesVersion = k8sVersion.String()
		return nil
	})
	return err
}
