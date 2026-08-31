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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
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
	crb                                *rbacv1.ClusterRoleBinding
}

func CreateRemoteClient(secret *corev1.Secret, namespace string) (*clientutil.RestClientGetter, client.Client, error) {
	rcg, err := clientutil.NewRestClientGetterFromSecret(secret, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("building rest client getter: %w", err)
	}
	rc, err := clientutil.NewK8sClientFromRestClientGetter(rcg)
	if err != nil {
		return nil, nil, fmt.Errorf("building remote k8s client: %w", err)
	}
	return rcg, rc, nil
}

func (p *Phase) ensureSecretFinalizerRemoved() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if controllerutil.RemoveFinalizer(p.ClusterSecret, lifecycle.CommonCleanupFinalizer) {
			if err := p.Client.Update(ctx, p.ClusterSecret); err != nil {
				return lifecycle.Break(), err
			}
		}
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) EnsureDeletePhases(cluster *greenhousev1alpha1.Cluster) []lifecycle.SubRoutine {
	phases := []lifecycle.SubRoutine{
		p.ensurePluginsDeleted(cluster),
	}
	if p.ClusterSecret != nil && p.ClusterSecret.Type != greenhouseapis.SecretTypeOIDCConfig {
		phases = append(phases, p.deleteClusterRoleBinding())
	}
	phases = append(phases, p.ensureSecretFinalizerRemoved())
	return phases
}

func (p *Phase) EnsureCreatePhases(cluster *greenhousev1alpha1.Cluster) []lifecycle.SubRoutine {
	if p.ClusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
		return []lifecycle.SubRoutine{
			p.ensureServiceAccountToken(cluster),
			p.ensureConnectivity(cluster),
			p.ensurePermissions(cluster),
			p.ensureNodesReady(cluster),
			p.ensureWorkloadSchedulable(cluster),
			p.ensureDiscoveryCache(cluster),
		}
	}
	return []lifecycle.SubRoutine{
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
