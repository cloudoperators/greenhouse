// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

const testPEM = `-----BEGIN CERTIFICATE-----
MIIDBTCCAe2gAwIBAgIIAL9NC2OnBfUw
-----END CERTIFICATE-----`

func oidcClusterSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-cluster",
			Namespace: "my-org",
			Annotations: map[string]string{
				greenhouseapis.SecretAPIServerURLAnnotation: "https://api.my-cluster.example.com",
			},
		},
		Type: greenhouseapis.SecretTypeOIDCConfig,
		Data: map[string][]byte{
			greenhouseapis.SecretAPIServerCAKey: []byte(base64.StdEncoding.EncodeToString([]byte(testPEM))),
		},
	}
}

func testCluster() *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", Namespace: "my-org"},
	}
}

func fluxAccessTestPhase(t *testing.T, secret *corev1.Secret, cluster *greenhousev1alpha1.Cluster) *Phase {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, greenhousev1alpha1.AddToScheme(scheme))
	return &Phase{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build(),
		ClusterSecret: secret,
	}
}

func TestBuildFluxAccessData_ContainsAllKeysFluxRequires(t *testing.T) {
	data, err := buildFluxAccessData(oidcClusterSecret())

	require.NoError(t, err)
	assert.Equal(t, "generic", data[fluxmeta.KubeConfigKeyProvider])
	assert.Equal(t, "https://api.my-cluster.example.com", data[fluxmeta.KubeConfigKeyAddress])
	assert.Equal(t, greenhouseapis.OIDCAudience, data[fluxmeta.KubeConfigKeyAudiences])
	assert.Equal(t, "my-cluster", data[fluxmeta.KubeConfigKeyServiceAccountName])
}

func TestBuildFluxAccessData_DecodesCACertificateToPEM(t *testing.T) {
	data, err := buildFluxAccessData(oidcClusterSecret())

	require.NoError(t, err)
	assert.Equal(t, testPEM, data[fluxmeta.KubeConfigKeyCACert])
}

func TestBuildFluxAccessData_ErrorsWhenAPIServerURLAnnotationMissing(t *testing.T) {
	secret := oidcClusterSecret()
	delete(secret.Annotations, greenhouseapis.SecretAPIServerURLAnnotation)

	_, err := buildFluxAccessData(secret)

	require.Error(t, err)
	assert.Contains(t, err.Error(), greenhouseapis.SecretAPIServerURLAnnotation)
}

func TestBuildFluxAccessData_ErrorsWhenCACertificateNotBase64(t *testing.T) {
	secret := oidcClusterSecret()
	secret.Data[greenhouseapis.SecretAPIServerCAKey] = []byte("not-base64!!!")

	_, err := buildFluxAccessData(secret)

	require.Error(t, err)
}

func TestEnsureFluxAccess_CreatesConfigMapOwnedByCluster(t *testing.T) {
	cluster := testCluster()
	p := fluxAccessTestPhase(t, oidcClusterSecret(), cluster)

	_, err := p.ensureFluxAccess(cluster)(context.Background())
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	require.NoError(t, p.Client.Get(context.Background(),
		types.NamespacedName{Name: "my-cluster", Namespace: "my-org"}, cm))
	assert.Equal(t, "generic", cm.Data[fluxmeta.KubeConfigKeyProvider])
	assert.Equal(t, testPEM, cm.Data[fluxmeta.KubeConfigKeyCACert])
	require.Len(t, cm.OwnerReferences, 1)
	assert.Equal(t, "Cluster", cm.OwnerReferences[0].Kind)
	assert.Equal(t, "my-cluster", cm.OwnerReferences[0].Name)
}

func TestEnsureFluxAccess_IsIdempotentAndReflectsSecretChanges(t *testing.T) {
	secret := oidcClusterSecret()
	cluster := testCluster()
	p := fluxAccessTestPhase(t, secret, cluster)

	_, err := p.ensureFluxAccess(cluster)(context.Background())
	require.NoError(t, err)

	secret.Annotations[greenhouseapis.SecretAPIServerURLAnnotation] = "https://api.new.example.com"
	_, err = p.ensureFluxAccess(cluster)(context.Background())
	require.NoError(t, err)

	list := &corev1.ConfigMapList{}
	require.NoError(t, p.Client.List(context.Background(), list))
	require.Len(t, list.Items, 1)
	assert.Equal(t, "https://api.new.example.com", list.Items[0].Data[fluxmeta.KubeConfigKeyAddress])
}

func TestEnsureFluxAccess_WritesNothingWhenSecretIncomplete(t *testing.T) {
	secret := oidcClusterSecret()
	delete(secret.Annotations, greenhouseapis.SecretAPIServerURLAnnotation)
	cluster := testCluster()
	p := fluxAccessTestPhase(t, secret, cluster)

	_, err := p.ensureFluxAccess(cluster)(context.Background())

	require.Error(t, err)
	list := &corev1.ConfigMapList{}
	require.NoError(t, p.Client.List(context.Background(), list))
	assert.Empty(t, list.Items)
}

func TestEnsureCreatePhases_OIDCClusterRunsFluxAccessLast(t *testing.T) {
	cluster := testCluster()
	p := fluxAccessTestPhase(t, oidcClusterSecret(), cluster)

	phases := p.EnsureCreatePhases(cluster)
	require.NotEmpty(t, phases)

	_, err := phases[len(phases)-1](context.Background())
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	require.NoError(t, p.Client.Get(context.Background(),
		types.NamespacedName{Name: "my-cluster", Namespace: "my-org"}, cm))
	assert.Equal(t, "generic", cm.Data[fluxmeta.KubeConfigKeyProvider])
}

func TestEnsureFluxAccess_SetsTrueConditionOnSuccess(t *testing.T) {
	cluster := testCluster()
	p := fluxAccessTestPhase(t, oidcClusterSecret(), cluster)

	_, err := p.ensureFluxAccess(cluster)(context.Background())
	require.NoError(t, err)

	condition := cluster.Status.GetConditionByType(greenhousev1alpha1.FluxAccessReady)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
}

func TestEnsureFluxAccess_SetsFalseConditionWhenSecretIncomplete(t *testing.T) {
	secret := oidcClusterSecret()
	delete(secret.Annotations, greenhouseapis.SecretAPIServerURLAnnotation)
	cluster := testCluster()
	p := fluxAccessTestPhase(t, secret, cluster)

	_, err := p.ensureFluxAccess(cluster)(context.Background())
	require.Error(t, err)

	condition := cluster.Status.GetConditionByType(greenhousev1alpha1.FluxAccessReady)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Contains(t, condition.Message, greenhouseapis.SecretAPIServerURLAnnotation)
}
