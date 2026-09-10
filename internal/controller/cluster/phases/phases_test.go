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

func TestEnsureNodesReady(t *testing.T) {
	tests := []struct {
		name          string
		mode          greenhousev1alpha1.ClusterMode
		wantCondition bool
		wantNodes     bool
	}{
		{
			name:          "workerless removes AllNodesReady condition and nodes",
			mode:          greenhousev1alpha1.ClusterModeWorkerless,
			wantCondition: false,
			wantNodes:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &greenhousev1alpha1.Cluster{}
			cluster.Spec.Mode = tc.mode
			cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", ""))
			cluster.Status.Nodes = &greenhousev1alpha1.Nodes{Total: 3}

			p := &Phase{}
			if _, err := p.ensureNodesReady(cluster)(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			hasCond := cluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady) != nil
			if hasCond != tc.wantCondition {
				t.Errorf("AllNodesReady condition present=%v, want %v", hasCond, tc.wantCondition)
			}
			hasNodes := cluster.Status.Nodes != nil
			if hasNodes != tc.wantNodes {
				t.Errorf("Status.Nodes present=%v, want %v", hasNodes, tc.wantNodes)
			}
		})
	}
}

func TestEnsureWorkloadSchedulable(t *testing.T) {
	tests := []struct {
		name          string
		mode          greenhousev1alpha1.ClusterMode
		preconditions []greenhousemetav1alpha1.Condition
		wantStatus    metav1.ConditionStatus
		wantReason    greenhousemetav1alpha1.ConditionReason
	}{
		{
			name:       "workerless sets PayloadSchedulable=False with WorkerlessCluster reason",
			mode:       greenhousev1alpha1.ClusterModeWorkerless,
			wantStatus: metav1.ConditionFalse,
			wantReason: greenhousev1alpha1.WorkerlessClusterReason,
		},
		{
			name:       "workerless ignores missing KubeConfigValid and AllNodesReady",
			mode:       greenhousev1alpha1.ClusterModeWorkerless,
			wantStatus: metav1.ConditionFalse,
			wantReason: greenhousev1alpha1.WorkerlessClusterReason,
		},
		{
			name:       "default with all conditions passing sets PayloadSchedulable=True",
			mode:       greenhousev1alpha1.ClusterModeDefault,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "default with KubeConfigValid=False sets PayloadSchedulable=False",
			mode: greenhousev1alpha1.ClusterModeDefault,
			preconditions: []greenhousemetav1alpha1.Condition{
				greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", "cert expired"),
			},
			wantStatus: metav1.ConditionFalse,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &greenhousev1alpha1.Cluster{}
			cluster.Spec.Mode = tc.mode
			for _, cond := range tc.preconditions {
				cluster.SetCondition(cond)
			}

			p := &Phase{}
			if _, err := p.ensureWorkloadSchedulable(cluster)(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			if cond == nil {
				t.Fatal("expected PayloadSchedulable condition to be set")
			}
			if cond.Status != tc.wantStatus {
				t.Errorf("PayloadSchedulable status=%v, want %v", cond.Status, tc.wantStatus)
			}
			if tc.wantReason != "" && cond.Reason != tc.wantReason {
				t.Errorf("PayloadSchedulable reason=%v, want %v", cond.Reason, tc.wantReason)
			}
		})
	}
}

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
