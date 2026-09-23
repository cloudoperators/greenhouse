// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

func minimalKubeconfig(t *testing.T) []byte {
	t.Helper()
	cfg := clientcmdapi.NewConfig()
	cfg.Clusters["test"] = &clientcmdapi.Cluster{Server: "https://api.example.com"}
	cfg.AuthInfos["test"] = &clientcmdapi.AuthInfo{Token: "test-token"}
	cfg.Contexts["test"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test"}
	cfg.CurrentContext = "test"
	b, err := clientcmd.Write(*cfg)
	require.NoError(t, err)
	return b
}

func TestNewRemoteKubeCfg_KubeConfig(t *testing.T) {
	ctx := context.Background()
	cluster := &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-org"},
	}
	secret := &corev1.Secret{
		Type: greenhouseapis.SecretTypeKubeConfig,
		Data: map[string][]byte{
			greenhouseapis.GreenHouseKubeConfigKey: minimalKubeconfig(t),
		},
	}

	c := &mocks.MockClient{}
	c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			*obj.(*corev1.Secret) = *secret
			return nil
		})

	cfg, err := lifecycle.NewRemoteKubeCfg(ctx, c, cluster)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com", cfg.Host)
}

func TestNewRemoteKubeCfg_WorkloadIdentity(t *testing.T) {
	ctx := context.Background()
	cluster := &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-org",
			Annotations: map[string]string{
				greenhouseapis.ClusterWorkloadIdentityAnnotation: greenhouseapis.ClusterWorkloadIdentityEnabled,
			},
		},
	}
	ca := base64.StdEncoding.EncodeToString([]byte("dummy-ca"))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-org",
			Annotations: map[string]string{
				greenhouseapis.SecretAPIServerURLAnnotation: "https://api.remote.example.com",
			},
		},
		Data: map[string][]byte{
			greenhouseapis.SecretAPIServerCAKey: []byte(ca),
		},
	}

	subResource := &mocks.MockSubResourceClient{}
	subResource.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ client.Object, subRes client.Object, _ ...client.SubResourceCreateOption) error {
			subRes.(*authenticationv1.TokenRequest).Status.Token = "wi-token"
			return nil
		})

	c := &mocks.MockClient{}
	c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			*obj.(*corev1.Secret) = *secret
			return nil
		})
	c.On("SubResource", "token").Return(subResource)

	cfg, err := lifecycle.NewRemoteKubeCfg(ctx, c, cluster)
	require.NoError(t, err)
	require.Equal(t, "https://api.remote.example.com", cfg.Host)
	require.Equal(t, "wi-token", cfg.BearerToken)
	require.Equal(t, []byte("dummy-ca"), cfg.CAData)
}
