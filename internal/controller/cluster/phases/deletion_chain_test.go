// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func kubeconfigSecret(t *testing.T, server string) *corev1.Secret {
	t.Helper()
	kubeconfig, err := clientcmd.Write(clientcmdapi.Config{
		Clusters:       map[string]*clientcmdapi.Cluster{"remote": {Server: server}},
		Contexts:       map[string]*clientcmdapi.Context{"remote": {Cluster: "remote"}},
		CurrentContext: "remote",
	})
	require.NoError(t, err)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecretName, Namespace: testNamespace},
		Type:       greenhouseapis.SecretTypeKubeConfig,
		Data:       map[string][]byte{greenhouseapis.GreenHouseKubeConfigKey: kubeconfig},
	}
}

func pluginForCluster(name, namespace, clusterName string) *greenhousev1alpha1.Plugin {
	return &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyCluster: clusterName},
		},
	}
}

func runDeletePhases(t *testing.T, p *Phase, cluster *greenhousev1alpha1.Cluster) lifecycle.Result {
	t.Helper()
	var result lifecycle.Result
	for _, phase := range p.EnsureDeletePhases(cluster) {
		var err error
		result, err = phase(context.Background())
		if result.Break || err != nil {
			return result
		}
	}
	return result
}

func TestEnsureDeletePhasesChain(t *testing.T) {
	tests := []struct {
		name                string
		secret              *corev1.Secret
		remoteReachable     bool
		wantPluginRetain    bool
		wantRemoteReachable bool
	}{
		{
			name:                "missing secret marks remote unreachable and retains plugins",
			secret:              nil,
			remoteReachable:     false,
			wantPluginRetain:    true,
			wantRemoteReachable: false,
		},
		{
			name:                "unreachable endpoint marks remote unreachable and retains plugins",
			secret:              kubeconfigSecret(t, "https://127.0.0.1:1"),
			remoteReachable:     true,
			wantPluginRetain:    true,
			wantRemoteReachable: false,
		},
		{
			name:                "OIDC cluster relies on existing RemoteReachable flag",
			secret:              oidcSecret(),
			remoteReachable:     false,
			wantPluginRetain:    true,
			wantRemoteReachable: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := healthyCluster()
			if !tc.remoteReachable {
				cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", "stale"))
			}

			plugin := pluginForCluster("test-plugin", testNamespace, testSecretName)

			builder := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(plugin)
			if tc.secret != nil {
				builder = builder.WithObjects(tc.secret)
			}
			c := builder.Build()

			p := &Phase{
				Client:          c,
				ClusterSecret:   tc.secret,
				RemoteReachable: tc.remoteReachable,
			}

			runDeletePhases(t, p, cluster)

			require.Equal(t, tc.wantRemoteReachable, p.RemoteReachable)

			updatedPlugin := &greenhousev1alpha1.Plugin{}
			require.NoError(t, c.Get(context.Background(), types.NamespacedName{Name: "test-plugin", Namespace: testNamespace}, updatedPlugin))
			hasRetain := updatedPlugin.Annotations[greenhouseapis.AnnotationKeyDeletionPolicy] == greenhouseapis.DeletionPolicyRetain
			require.Equal(t, tc.wantPluginRetain, hasRetain)
		})
	}
}
