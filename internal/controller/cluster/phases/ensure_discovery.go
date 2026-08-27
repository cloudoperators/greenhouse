// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureDiscoveryCache(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		dc, err := p.RestClientGetter.ToDiscoveryClient()
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to get discovery client")
			cluster.Status.DiscoveryCache = nil
			return lifecycle.RequeueAfter(10 * time.Minute), nil
		}

		preferredResources, err := dc.ServerPreferredResources()
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to get server groups")
			cluster.Status.DiscoveryCache = nil
			return lifecycle.RequeueAfter(10 * time.Minute), nil
		}

		names := make([]string, 0, len(preferredResources))
		for _, g := range preferredResources {
			names = append(names, g.GroupVersion)
		}
		sort.Strings(names)

		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(names, ","))))

		if cluster.Status.DiscoveryCache != nil && cluster.Status.DiscoveryCache.Hash == hash {
			return lifecycle.RequeueAfter(10 * time.Minute), nil
		}

		cluster.Status.DiscoveryCache = &greenhousev1alpha1.DiscoveryCache{
			Hash:   hash,
			Groups: names,
		}
		return lifecycle.RequeueAfter(10 * time.Minute), nil
	}
}
