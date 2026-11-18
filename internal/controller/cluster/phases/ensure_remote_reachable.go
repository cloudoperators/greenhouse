// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// checkKubeconfigReachability probes the remote cluster to determine if it is reachable.
func (p *Phase) checkKubeconfigReachability(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if p.ClusterSecret == nil {
			return lifecycle.Continue(), nil
		}

		var rcg *clientutil.RestClientGetter
		if p.WorkloadIdentityEnabled {
			var err error
			rcg, err = clientutil.NewRestClientGetterForWI(ctx, p.Client, p.ClusterSecret, cluster.GetNamespace())
			if err != nil {
				// token minting failure is transient — don't change RemoteReachable
				return lifecycle.Continue(), nil //nolint:nilerr
			}
		} else {
			var err error
			rcg, err = clientutil.NewRestClientGetterFromSecret(p.ClusterSecret, cluster.GetNamespace())
			if err != nil {
				p.RemoteReachable = false
				return lifecycle.Continue(), nil //nolint:nilerr // unreachable cluster is not an error, cleanup phases still run
			}
		}

		if _, err := clientutil.GetKubernetesVersion(rcg); err != nil {
			p.RemoteReachable = false
			return lifecycle.Continue(), nil //nolint:nilerr // unreachable cluster is not an error, cleanup phases still run
		}
		p.RemoteReachable = true
		return lifecycle.Continue(), nil
	}
}
