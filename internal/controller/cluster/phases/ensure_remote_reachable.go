// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// checkKubeconfigReachability probes the remote cluster using the kubeconfig secret.
func (p *Phase) checkKubeconfigReachability(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		if p.ClusterSecret == nil || p.ClusterSecret.Type == greenhouseapis.SecretTypeOIDCConfig {
			return lifecycle.Continue(), nil
		}
		rcg, err := clientutil.NewRestClientGetterFromSecret(p.ClusterSecret, cluster.GetNamespace())
		if err != nil {
			p.RemoteReachable = false
			return lifecycle.Continue(), nil
		}
		if _, err := clientutil.GetKubernetesVersion(rcg); err != nil {
			p.RemoteReachable = false
			return lifecycle.Continue(), nil
		}
		p.RemoteReachable = true
		return lifecycle.Continue(), nil
	}
}
