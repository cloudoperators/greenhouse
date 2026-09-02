// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func fluxKubeConfigScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, greenhousev1alpha1.AddToScheme(scheme))
	return scheme
}

func clusterWithConnectivity(connectivity string) *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-cluster",
			Namespace:   "my-org",
			Annotations: map[string]string{greenhouseapis.ClusterConnectivityAnnotation: connectivity},
		},
	}
}

func fluxAccessConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", Namespace: "my-org"},
		Data:       map[string]string{"provider": "generic"},
	}
}

func pluginOnCluster(clusterName string) *greenhousev1alpha1.Plugin {
	return &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "my-plugin", Namespace: "my-org"},
		Spec:       greenhousev1alpha1.PluginSpec{ClusterName: clusterName},
	}
}

func TestUseFluxAccessConfigMap_TrueForOIDCClusterWithConfigMap(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(fluxKubeConfigScheme(t)).
		WithObjects(clusterWithConnectivity(greenhouseapis.ClusterConnectivityOIDC), fluxAccessConfigMap()).Build()

	use, err := useFluxAccessConfigMap(context.Background(), c, pluginOnCluster("my-cluster"))

	require.NoError(t, err)
	assert.True(t, use)
}

func TestUseFluxAccessConfigMap_FalseWhenConfigMapNotYetRendered(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(fluxKubeConfigScheme(t)).
		WithObjects(clusterWithConnectivity(greenhouseapis.ClusterConnectivityOIDC)).Build()

	use, err := useFluxAccessConfigMap(context.Background(), c, pluginOnCluster("my-cluster"))

	require.NoError(t, err)
	assert.False(t, use)
}

func TestUseFluxAccessConfigMap_FalseForKubeconfigOnboardedCluster(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(fluxKubeConfigScheme(t)).
		WithObjects(clusterWithConnectivity(greenhouseapis.ClusterConnectivityKubeconfig), fluxAccessConfigMap()).Build()

	use, err := useFluxAccessConfigMap(context.Background(), c, pluginOnCluster("my-cluster"))

	require.NoError(t, err)
	assert.False(t, use)
}

func TestUseFluxAccessConfigMap_FalseForCentralCluster(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(fluxKubeConfigScheme(t)).Build()

	use, err := useFluxAccessConfigMap(context.Background(), c, pluginOnCluster(""))

	require.NoError(t, err)
	assert.False(t, use)
}

func TestUseFluxAccessConfigMap_FalseWhenClusterMissing(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(fluxKubeConfigScheme(t)).Build()

	use, err := useFluxAccessConfigMap(context.Background(), c, pluginOnCluster("my-cluster"))

	require.NoError(t, err)
	assert.False(t, use)
}
