// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

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
		nodes         *greenhousev1alpha1.Nodes
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
		{
			name:       "default with no ready nodes sets PayloadSchedulable=False",
			mode:       greenhousev1alpha1.ClusterModeDefault,
			nodes:      &greenhousev1alpha1.Nodes{Total: 3, Ready: 0},
			wantStatus: metav1.ConditionFalse,
		},
		{
			name:       "default with some nodes not ready but ready nodes exist sets PayloadSchedulable=True",
			mode:       greenhousev1alpha1.ClusterModeDefault,
			nodes:      &greenhousev1alpha1.Nodes{Total: 210, Ready: 207},
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "node observation failure (AllNodesReady=False, nil nodes) sets PayloadSchedulable=False",
			mode: greenhousev1alpha1.ClusterModeDefault,
			preconditions: []greenhousemetav1alpha1.Condition{
				greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.AllNodesReady, "", "connection refused"),
			},
			wantStatus: metav1.ConditionFalse,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &greenhousev1alpha1.Cluster{}
			cluster.Spec.Mode = tc.mode
			cluster.Status.Nodes = tc.nodes
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

func TestEnsureTeamRBACDeleted(t *testing.T) {
	unreachableKubeConfig, err := clientcmd.Write(clientcmdapi.Config{
		Clusters:       map[string]*clientcmdapi.Cluster{"remote": {Server: "https://127.0.0.1:1"}},
		Contexts:       map[string]*clientcmdapi.Context{"remote": {Cluster: "remote"}},
		CurrentContext: "remote",
	})
	require.NoError(t, err)
	unreachableSecret := oidcSecret()
	unreachableSecret.Data[greenhouseapis.GreenHouseKubeConfigKey] = unreachableKubeConfig

	tests := []struct {
		name                string
		objects             []client.Object
		deletionPolicy      string
		workloadIdentity    bool
		interceptor         interceptor.Funcs
		wantBreak           bool
		wantKubeConfigValid metav1.ConditionStatus
	}{
		{
			name:                "secret without a kubeconfig marks the kubeconfig invalid",
			objects:             []client.Object{oidcSecret()},
			wantBreak:           true,
			wantKubeConfigValid: metav1.ConditionFalse,
		},
		{
			name:                "unreachable remote cluster marks the kubeconfig invalid",
			objects:             []client.Object{unreachableSecret},
			wantBreak:           true,
			wantKubeConfigValid: metav1.ConditionFalse,
		},
		{
			name:             "failed token request on a workload identity cluster is retried",
			objects:          []client.Object{oidcSecret()},
			workloadIdentity: true,
			interceptor: interceptor.Funcs{
				SubResourceCreate: func(context.Context, client.Client, string, client.Object, client.Object, ...client.SubResourceCreateOption) error {
					return apierrors.NewServiceUnavailable("unavailable")
				},
			},
			wantBreak:           true,
			wantKubeConfigValid: metav1.ConditionTrue,
		},
		{
			name:                "retained cluster keeps its team RBAC",
			deletionPolicy:      greenhouseapis.DeletionPolicyRetain,
			wantKubeConfigValid: metav1.ConditionTrue,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := healthyCluster()
			cluster.Annotations = map[string]string{greenhouseapis.AnnotationKeyDeletionPolicy: tc.deletionPolicy}
			if tc.workloadIdentity {
				cluster.Annotations[greenhouseapis.ClusterWorkloadIdentityAnnotation] = greenhouseapis.ClusterWorkloadIdentityEnabled
			}
			p := &Phase{Client: fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(tc.objects...).WithInterceptorFuncs(tc.interceptor).Build()}

			result, err := p.ensureTeamRBACDeleted(cluster)(context.Background())
			require.Equal(t, tc.wantBreak, result.Break)
			require.Equal(t, tc.wantBreak, err != nil)
			require.Equal(t, tc.wantKubeConfigValid, cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid).Status)
		})
	}
}

func TestDeleteTeamRBACContinuesPastFailedDeletes(t *testing.T) {
	scheme := testScheme(t)
	require.NoError(t, rbacv1.AddToScheme(scheme))
	labels := map[string]string{greenhouseapis.LabelKeyRole: "team", greenhouseapis.LabelKeyRoleBinding: "team"}
	roleBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "team", Namespace: "kube-system", Labels: labels}}
	remoteClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(
			roleBinding,
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "team", Labels: labels}},
			&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "team", Labels: labels}},
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Group: rbacv1.GroupName, Resource: "rolebindings"}, "team", errors.New("denied by webhook"))
			},
		}).
		Build()

	err := deleteTeamRBAC(context.Background(), remoteClient, []rbacv1.RoleBinding{*roleBinding})
	require.True(t, apierrors.IsForbidden(err))

	clusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
	require.NoError(t, remoteClient.List(context.Background(), clusterRoleBindings))
	require.Empty(t, clusterRoleBindings.Items)
	clusterRoles := &rbacv1.ClusterRoleList{}
	require.NoError(t, remoteClient.List(context.Background(), clusterRoles))
	require.Empty(t, clusterRoles.Items)
}
