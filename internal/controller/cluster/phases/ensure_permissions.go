// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensurePermissions(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(p.RestClientGetter)
		if err != nil {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PermissionsVerified, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		missing := common.CheckClientClusterPermission(ctx, remoteClient, "", corev1.NamespaceAll)
		if len(missing) > 0 {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PermissionsVerified, "", "missing cluster admin permission",
			))
			return lifecycle.Continue(), nil
		}

		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.PermissionsVerified, "", "ServiceAccount has cluster admin permissions",
		))
		return lifecycle.Continue(), nil
	}
}
