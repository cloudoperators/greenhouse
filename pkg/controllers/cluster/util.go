// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/pkg/errors"
	"google.golang.org/grpc/status"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

var defaultRequeueInterval = 10 * time.Minute

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
	var serviceAccount = new(corev1.ServiceAccount)
	serviceAccount.Name = serviceAccountName
	serviceAccount.Namespace = cluster.GetNamespace()
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, serviceAccount, func() error {
		return nil
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

func reconcileClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
	clusterRoleBinding.Name = serviceAccountName

	var nameSpace = new(corev1.Namespace)
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: cluster.GetNamespace()}, nameSpace); err != nil {
		return err
	}

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, clusterRoleBinding, func() error {
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: cluster.GetNamespace(),
			},
		}
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: rbacv1.GroupName,
		}
		return controllerutil.SetOwnerReference(nameSpace, clusterRoleBinding, k8sClient.Scheme())
	})

	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created clusterRoleBinding", "cluster", clusterRoleBinding.Name)
		// TODO: emit event on cluster
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated clusterRoleBinding", "cluster", clusterRoleBinding.Name)
		// TODO: emit event on cluster
	}
	return nil
}

type tokenHelper struct {
	client.Client
	Proxy                              string
	HeadscaleAddress                   string
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
}

// ReconcileServiceAccountToken reconciles the service account token for the cluster and updates the secret containing the kube config
func (t *tokenHelper) ReconcileServiceAccountToken(ctx context.Context, restClientGetter *clientutil.RestClientGetter, cluster *greenhousev1alpha1.Cluster) error {
	// TODO: Do not rely on the status but actually check the token expiration.
	if !cluster.Status.BearerTokenExpirationTimestamp.IsZero() && cluster.Status.BearerTokenExpirationTimestamp.Time.After(time.Now().Add(t.RenewRemoteClusterBearerTokenAfter)) {
		log.FromContext(ctx).V(5).Info("bearer token is still valid", "cluster", cluster.Name, "expirationTimestamp", cluster.Status.BearerTokenExpirationTimestamp.Time)
		return nil
	}

	remoteRestConfig, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return err
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
	case greenhousev1alpha1.ClusterAccessModeHeadscale:
		generatedKubeConfig, err = generateNewClientKubeConfigHeadscale(ctx, restClientGetter, tokenRequestResponse.Status.Token, cluster, t.Proxy, t.HeadscaleAddress)
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

// generateNewClientKubeConfigHeadscale generates a kubeconfig for the client to access the cluster from REST config coming from the secret plus the modifications needed for headscale.
func generateNewClientKubeConfigHeadscale(_ context.Context, restConfigGetter *clientutil.RestClientGetter, bearerToken string, cluster *greenhousev1alpha1.Cluster, proxy, headscaleAddress string) ([]byte, error) {
	restConfig, err := restConfigGetter.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load kube clientConfig for cluster %s", cluster.GetName())
	}

	kubeConfigGenerator := &KubeConfigHelper{
		Host:          "https://" + headscaleAddress,
		TLSServerName: "127.0.0.1",
		ProxyURL:      proxy,
		CAData:        restConfig.CAData,
		BearerToken:   bearerToken,
		Username:      serviceAccountName,
		Namespace:     cluster.GetNamespace(),
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

// deleteNamespaceInRemoteCluster deletes the namespace ignoring NotFound errors.
func deleteNamespaceInRemoteCluster(ctx context.Context, remoteK8sClient client.Client, cluster *greenhousev1alpha1.Cluster) error {
	var namespace = new(corev1.Namespace)
	namespace.Name = cluster.GetNamespace()
	// Attempt deletion if the namespace in the remote cluster.
	err := remoteK8sClient.Delete(ctx, namespace)
	// Ignore errors if was already deleted.
	return client.IgnoreNotFound(err)
}

// ReconcileHeadscaleUser ensure a user for the cluster exists in the headscale coordination server.
func ReconcileHeadscaleUser(ctx context.Context, recorder record.EventRecorder, cluster *greenhousev1alpha1.Cluster, headscaleGRPCClient headscalev1.HeadscaleServiceClient) error {
	createResp, err := headscaleGRPCClient.CreateUser(ctx, &headscalev1.CreateUserRequest{
		Name: headscaleKeyForCluster(cluster),
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return err
		}
		switch {
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to create user %s", headscaleKeyForCluster(cluster))
		case strings.Contains(errStatus.Message(), "already exists"):
			return nil
		}
		return err
	}
	recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscaleUserCreated", "Headscale user %s created for cluster", createResp.User.Name)
	return nil
}

// ReconcilePreAuthorizationKey ensure a pre-authorization key exists for the given cluster.
func ReconcilePreAuthorizationKey(ctx context.Context, cluster *greenhousev1alpha1.Cluster, headscaleGRPCClient headscalev1.HeadscaleServiceClient, HeadscalePreAuthenticationKeyMinValidity time.Duration) (*headscalev1.PreAuthKey, error) {
	// Check whether an existing pre-authorization key can be used.
	resp, err := headscaleGRPCClient.ListPreAuthKeys(ctx, &headscalev1.ListPreAuthKeysRequest{
		User: headscaleKeyForCluster(cluster),
	})
	if err != nil {
		return nil, err
	}
	for _, key := range resp.GetPreAuthKeys() {
		if isPreAuthenticationKeyIsNotExpired(key, HeadscalePreAuthenticationKeyMinValidity) {
			return key, nil
		}
	}

	// Request a new pre-authorization key.
	expiration := time.Now().UTC().Add(7 * 24 * time.Hour)
	createPreAuth := &headscalev1.CreatePreAuthKeyRequest{
		User:       headscaleKeyForCluster(cluster),
		Reusable:   true,
		Ephemeral:  true,
		Expiration: timestamppb.New(expiration),
		AclTags:    []string{"tag:greenhouse", "tag:client", "tag:" + headscaleKeyForCluster(cluster)},
	}
	createResp, err := headscaleGRPCClient.CreatePreAuthKey(ctx, createPreAuth)
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return nil, err
		}
		switch {
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return nil, fmt.Errorf("headscale: unauthorized to create user %s", headscaleKeyForCluster(cluster))
		case strings.Contains(errStatus.Message(), "tag is invalid"):
			return nil, fmt.Errorf("headscale: failed to create PreAuthKey for user %s: %s", headscaleKeyForCluster(cluster), errStatus.Message())
		}
		return nil, errors.Wrapf(err, "failed to create PreAuthenticationKey for user %s", headscaleKeyForCluster(cluster))
	}
	log.FromContext(ctx).Info("PreAuthenticationKey issued", "user", headscaleKeyForCluster(cluster), "expireDate", createResp.PreAuthKey.Expiration.AsTime())
	return createResp.PreAuthKey, nil
}
