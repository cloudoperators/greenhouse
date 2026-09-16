// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

const (
	testSecretName = "test-cluster"
	testNamespace  = "test-org"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, greenhousev1alpha1.AddToScheme(scheme))
	return scheme
}

// oidcSecret returns an OIDC-type secret carrying the api-server-url annotation and a
// base64-encoded CA, mirroring what the onboarding flow persists.
func oidcSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				greenhouseapis.SecretAPIServerURLAnnotation: "https://api.remote.example.com",
			},
		},
		Type: greenhouseapis.SecretTypeOIDCConfig,
		Data: map[string][]byte{
			greenhouseapis.SecretAPIServerCAKey: []byte(base64.StdEncoding.EncodeToString([]byte("dummy-ca"))),
		},
	}
}

func notFound() error {
	return apierrors.NewNotFound(schema.GroupResource{}, "")
}

func TestBootstrapEnsureCreatePhases(t *testing.T) {
	tests := []struct {
		name          string
		secretType    corev1.SecretType
		workloadID    bool
		wantPhaseslen int
	}{
		{
			name:          "it should generate a kubeconfig and requeue for OIDC without workload identity",
			secretType:    greenhouseapis.SecretTypeOIDCConfig,
			workloadID:    false,
			wantPhaseslen: 4, // oidcKubeConfig, cluster, ownerReferences, requeue
		},
		{
			name:          "it should write a configmap and not requeue for OIDC with workload identity",
			secretType:    greenhouseapis.SecretTypeOIDCConfig,
			workloadID:    true,
			wantPhaseslen: 3, // wiConfigMap, cluster, ownerReferences
		},
		{
			name:          "it should requeue for KubeConfig",
			secretType:    greenhouseapis.SecretTypeKubeConfig,
			workloadID:    true,
			wantPhaseslen: 3, // cluster, ownerReferences, requeue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := oidcSecret()
			secret.Type = tt.secretType
			p := &BootstrapPhase{
				Secret:                  secret,
				WorkloadIdentityEnabled: tt.workloadID,
			}
			phases := p.EnsureCreatePhases()
			require.Len(t, phases, tt.wantPhaseslen)
		})
	}
}

func TestBootstrapEnsureDeletePhases(t *testing.T) {
	p := &BootstrapPhase{Secret: oidcSecret()}
	require.Len(t, p.EnsureDeletePhases(), 1)
}

func TestEnsureCluster(t *testing.T) {
	tests := []struct {
		name                 string
		secretType           corev1.SecretType
		workloadID           bool
		wantConnectivity     string
		wantWorkloadIdentity bool
	}{
		{
			name:             "it should create the cluster for a kubeconfig secret",
			secretType:       greenhouseapis.SecretTypeKubeConfig,
			workloadID:       false,
			wantConnectivity: greenhouseapis.ClusterConnectivityKubeconfig,
		},
		{
			name:                 "it should create the cluster for an OIDC secret with workload identity",
			secretType:           greenhouseapis.SecretTypeOIDCConfig,
			workloadID:           true,
			wantConnectivity:     greenhouseapis.ClusterConnectivityOIDC,
			wantWorkloadIdentity: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := testScheme(t)
			secret := oidcSecret()
			secret.Type = tt.secretType

			c := &mocks.MockClient{}
			c.On("Scheme").Return(scheme)
			c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(notFound())

			var createdCluster *greenhousev1alpha1.Cluster
			c.On("Create", mock.Anything, mock.Anything, mock.Anything).
				Return(func(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
					createdCluster = obj.(*greenhousev1alpha1.Cluster)
					return nil
				})

			p := &BootstrapPhase{Client: c, Scheme: scheme, Secret: secret, WorkloadIdentityEnabled: tt.workloadID}
			res, err := p.ensureCluster()(ctx)

			require.NoError(t, err)
			require.Equal(t, lifecycle.Continue(), res)
			require.NotNil(t, createdCluster)
			require.Equal(t, greenhousev1alpha1.ClusterAccessModeDirect, createdCluster.Spec.AccessMode)
			require.Equal(t, tt.wantConnectivity, createdCluster.GetAnnotations()[greenhouseapis.ClusterConnectivityAnnotation])
			if tt.wantWorkloadIdentity {
				require.Equal(t, greenhouseapis.ClusterWorkloadIdentityEnabled, createdCluster.GetAnnotations()[greenhouseapis.ClusterWorkloadIdentityAnnotation])
			} else {
				require.NotContains(t, createdCluster.GetAnnotations(), greenhouseapis.ClusterWorkloadIdentityAnnotation)
			}
		})
	}
}

// TestEnsureWorkloadIdentityConfigMap asserts the WI subroutine creates the SA, writes the
// Flux ConfigMap with the expected keys, and never mints a token.
func TestEnsureWorkloadIdentityConfigMap(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	secret := oidcSecret()

	c := &mocks.MockClient{}
	c.On("Scheme").Return(scheme)
	c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(notFound())

	var writtenCM *corev1.ConfigMap
	c.On("Create", mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
			if cm, ok := obj.(*corev1.ConfigMap); ok {
				writtenCM = cm
			}
			return nil
		})

	p := &BootstrapPhase{Client: c, Scheme: scheme, Secret: secret, WorkloadIdentityEnabled: true}
	res, err := p.ensureWorkloadIdentityConfigMap()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)
	require.NotNil(t, writtenCM)
	require.Equal(t, "https://api.remote.example.com", writtenCM.Data[fluxmeta.KubeConfigKeyAddress])
	require.Equal(t, greenhouseapis.OIDCAudience, writtenCM.Data[fluxmeta.KubeConfigKeyAudiences])
	require.Equal(t, testSecretName, writtenCM.Data[fluxmeta.KubeConfigKeyServiceAccountName])
	require.Equal(t, "generic", writtenCM.Data[fluxmeta.KubeConfigKeyProvider])
	require.Equal(t, "dummy-ca", writtenCM.Data[fluxmeta.KubeConfigKeyCACert])
	// no token subresource is ever requested under workload identity
	c.AssertNotCalled(t, "SubResource", mock.Anything)
}

// TestEnsureWorkloadIdentityConfigMapDropsStaleKubeConfig asserts migration cleanup: a
// pre-existing greenhousekubeconfig key and its generated timestamp annotation are removed.
func TestEnsureWorkloadIdentityConfigMapDropsStaleKubeConfig(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	secret := oidcSecret()
	secret.Data[greenhouseapis.GreenHouseKubeConfigKey] = []byte("stale")
	secret.Annotations[greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation] = "2026-01-01 00:00:00"

	c := &mocks.MockClient{}
	c.On("Scheme").Return(scheme)
	c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(notFound())
	c.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	var updatedSecret *corev1.Secret
	c.On("Update", mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
			if s, ok := obj.(*corev1.Secret); ok {
				updatedSecret = s
			}
			return nil
		})

	p := &BootstrapPhase{Client: c, Scheme: scheme, Secret: secret, WorkloadIdentityEnabled: true}
	res, err := p.ensureWorkloadIdentityConfigMap()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)
	require.NotNil(t, updatedSecret)
	require.NotContains(t, updatedSecret.Data, greenhouseapis.GreenHouseKubeConfigKey)
	require.NotContains(t, updatedSecret.Annotations, greenhouseapis.SecretOIDCConfigGeneratedOnAnnotation)
}

// TestEnsureClusterDeleted covers cluster already gone and cluster available to be deleted flows
func TestEnsureClusterDeleted(t *testing.T) {
	tests := []struct {
		name       string
		getErr     error
		wantDelete bool
		wantResult lifecycle.Result
	}{
		{
			name:       "it should continue without delete when the cluster is already gone",
			getErr:     notFound(),
			wantDelete: false,
			wantResult: lifecycle.Continue(),
		},
		{
			name:       "it should delete the cluster and block finalizer removal with a requeue when present",
			getErr:     nil,
			wantDelete: true,
			wantResult: lifecycle.RequeueAfter(10 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			secret := oidcSecret()
			c := &mocks.MockClient{}
			c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.getErr)
			if tt.wantDelete {
				c.EXPECT().Delete(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			p := &BootstrapPhase{Client: c, Scheme: testScheme(t), Secret: secret}
			res, err := p.ensureClusterDeleted()(ctx)

			require.NoError(t, err)
			require.Equal(t, tt.wantResult, res)
			if tt.wantDelete {
				c.AssertCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
			} else {
				c.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// TestRequeue asserts the requeue subroutine schedules the default interval.
func TestRequeue(t *testing.T) {
	p := &BootstrapPhase{Secret: oidcSecret()}
	res, err := p.requeue()(context.Background())
	require.NoError(t, err)
	require.Equal(t, lifecycle.RequeueAfter(utils.DefaultRequeueInterval), res)
}

// TestEnsureOwnerReferencesSkipsWhenClusterDeleting skips setting the owner reference while the Cluster is being deleted.
func TestEnsureOwnerReferencesSkipsWhenClusterDeleting(t *testing.T) {
	ctx := context.Background()
	secret := oidcSecret()
	now := metav1.Now()

	c := &mocks.MockClient{}
	c.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			obj.(*greenhousev1alpha1.Cluster).DeletionTimestamp = &now
			return nil
		})

	p := &BootstrapPhase{Client: c, Scheme: testScheme(t), Secret: secret}
	res, err := p.ensureOwnerReferences()(ctx)

	require.NoError(t, err)
	require.Equal(t, lifecycle.Continue(), res)
	// owner reference is not patched onto the secret while the cluster is terminating
	c.AssertNotCalled(t, "Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

var _ client.Client = (*mocks.MockClient)(nil)
