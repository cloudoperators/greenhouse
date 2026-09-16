// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// healthyCluster is the state a cluster is in when its Secret goes bad underneath it.
func healthyCluster() *greenhousev1alpha1.Cluster {
	cluster := &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: testSecretName, Namespace: testNamespace},
	}
	cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.KubeConfigValid, "", ""))
	return cluster
}

func requireInvalidKubeConfig(t *testing.T, cluster *greenhousev1alpha1.Cluster, result lifecycle.Result, err error) {
	t.Helper()
	require.Error(t, err)
	require.True(t, result.Break, "the chain must stop")

	condition := cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
	require.NotNil(t, condition)
	require.True(t, condition.IsFalse(), "KubeConfigValid must be false when the remote client cannot be built")
	require.NotEmpty(t, condition.Message)
}

func TestCreateRemoteClientReportsInvalidKubeConfig(t *testing.T) {
	// what the workload identity path leaves behind on an OIDC secret
	secret := oidcSecret()
	delete(secret.Data, greenhouseapis.GreenHouseKubeConfigKey)

	cluster := healthyCluster()
	p := &Phase{ClusterSecret: secret}

	result, err := p.createRemoteClient(cluster)(context.Background())
	requireInvalidKubeConfig(t, cluster, result, err)
}

func TestCreateWorkloadIdentityClientReportsInvalidKubeConfig(t *testing.T) {
	secret := oidcSecret()
	secret.Data[greenhouseapis.SecretAPIServerCAKey] = []byte("not-base64!")

	cluster := healthyCluster()
	p := &Phase{ClusterSecret: secret}

	result, err := p.createWorkloadIdentityClient(cluster)(context.Background())
	requireInvalidKubeConfig(t, cluster, result, err)
}
