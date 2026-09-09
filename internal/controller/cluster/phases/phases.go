// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type Phase struct {
	Client                             client.Client
	Recorder                           events.EventRecorder
	RemoteClusterBearerTokenValidity   time.Duration
	RenewRemoteClusterBearerTokenAfter time.Duration
	ClusterSecret                      *corev1.Secret
	RestClientGetter                   *clientutil.RestClientGetter
	RemoteClient                       client.Client
	WorkloadIdentityEnabled            bool
	crb                                *rbacv1.ClusterRoleBinding
}

// breakInvalidKubeConfig reports the failure, otherwise the last known status survives and the cluster stays Ready.
func breakInvalidKubeConfig(cluster *greenhousev1alpha1.Cluster, err error) (lifecycle.Result, error) {
	cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", err.Error()))
	return lifecycle.Break(), err
}

func (p *Phase) createRemoteClient(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		rcg, err := clientutil.NewRestClientGetterFromSecret(p.ClusterSecret, cluster.GetNamespace())
		if err != nil {
			return breakInvalidKubeConfig(cluster, fmt.Errorf("error building rest client getter: %w", err))
		}
		rc, err := clientutil.NewK8sClientFromRestClientGetter(rcg)
		if err != nil {
			return breakInvalidKubeConfig(cluster, fmt.Errorf("error building remote k8s client: %w", err))
		}
		p.RestClientGetter = rcg
		p.RemoteClient = rc
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) createWorkloadIdentityClient(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		rcg, err := clientutil.NewRestClientGetterForWI(ctx, p.Client, p.ClusterSecret, cluster.GetNamespace())
		if err != nil {
			return breakInvalidKubeConfig(cluster, fmt.Errorf("error building workload identity rest client getter: %w", err))
		}
		rc, err := clientutil.NewK8sClientFromRestClientGetter(rcg)
		if err != nil {
			return breakInvalidKubeConfig(cluster, fmt.Errorf("error building workload identity remote k8s client: %w", err))
		}
		p.RestClientGetter = rcg
		p.RemoteClient = rc
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) EnsureDeletePhases(cluster *greenhousev1alpha1.Cluster) []lifecycle.SubRoutine {
	phases := []lifecycle.SubRoutine{
		p.ensurePluginsDeleted(cluster),
	}
	if p.ClusterSecret != nil && p.ClusterSecret.Type != greenhouseapis.SecretTypeOIDCConfig {
		phases = append([]lifecycle.SubRoutine{p.createRemoteClient(cluster)}, append(phases, p.deleteClusterRoleBinding())...)
	}
	return phases
}

func (p *Phase) EnsureCreatePhases(cluster *greenhousev1alpha1.Cluster) []lifecycle.SubRoutine {
	if p.ClusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		phases := make([]lifecycle.SubRoutine, 0, 6)
		// if workload identity feature gate is enabled there is no need to write SA token to cluster secret
		if p.WorkloadIdentityEnabled {
			phases = append(phases, p.createWorkloadIdentityClient(cluster))
		} else {
			phases = append(phases, p.createRemoteClient(cluster), p.ensureServiceAccountToken(cluster))
		}
		return append(phases,
			p.ensureConnectivity(cluster),
			p.ensurePermissions(cluster),
			p.ensureNodesReady(cluster),
			p.ensureWorkloadSchedulable(cluster),
			p.ensureDiscoveryCache(cluster),
		)
	}
	return []lifecycle.SubRoutine{
		p.createRemoteClient(cluster),
		p.ensureConnectivity(cluster),
		p.ensureClusterRoleBinding(cluster),
		p.ensureNamespace(cluster),
		p.ensureServiceAccount(cluster),
		p.ensureServiceAccountToken(cluster),
		p.ensurePermissions(cluster),
		p.ensureManagedResourcesDeployed(cluster),
		p.ensureNodesReady(cluster),
		p.ensureWorkloadSchedulable(cluster),
		p.ensureDiscoveryCache(cluster),
	}
}
