// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const clusterK8sVersionUnknown = "unknown"

func (p *Phase) ensureConnectivity(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		kubeVersion, err := clientutil.GetKubernetesVersion(p.RestClientGetter)
		if err != nil {
			cluster.Status.KubernetesVersion = clusterK8sVersionUnknown
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.KubeConfigValid, "", err.Error(),
			))
			return lifecycle.Break(), err
		}
		cluster.Status.KubernetesVersion = kubeVersion.String()
		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.KubeConfigValid, "", "",
		))
		return lifecycle.Continue(), nil
	}
}
