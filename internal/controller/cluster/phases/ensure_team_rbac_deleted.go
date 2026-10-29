// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"errors"
	"net"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureTeamRBACDeleted(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if strings.EqualFold(cluster.Annotations[greenhouseapis.AnnotationKeyDeletionPolicy], greenhouseapis.DeletionPolicyRetain) {
			return lifecycle.Continue(), nil
		}
		remoteClient, err := clientutil.NewK8sClientFromCluster(ctx, p.Client, cluster)
		if err != nil {
			if errors.As(err, new(net.Error)) || apierrors.IsTooManyRequests(err) || apierrors.IsServiceUnavailable(err) ||
				apierrors.IsInternalError(err) || apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) {
				return lifecycle.Break(), err
			}
			return breakInvalidKubeConfig(cluster, err)
		}
		roleBindings := &rbacv1.RoleBindingList{}
		err = remoteClient.List(ctx, roleBindings, client.HasLabels{greenhouseapis.LabelKeyRoleBinding})
		switch {
		case apierrors.IsUnauthorized(err), apierrors.IsForbidden(err):
			return lifecycle.Continue(), nil
		case err != nil && !errors.As(err, new(apierrors.APIStatus)):
			return breakInvalidKubeConfig(cluster, err)
		case err != nil:
			return lifecycle.Break(), err
		}
		if err := deleteTeamRBAC(ctx, remoteClient, roleBindings.Items); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

func deleteTeamRBAC(ctx context.Context, remoteClient client.Client, roleBindings []rbacv1.RoleBinding) error {
	var errs []error
	for _, roleBinding := range roleBindings {
		if err := remoteClient.Delete(ctx, &roleBinding); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs,
		remoteClient.DeleteAllOf(ctx, &rbacv1.ClusterRoleBinding{}, client.HasLabels{greenhouseapis.LabelKeyRoleBinding}),
		remoteClient.DeleteAllOf(ctx, &rbacv1.ClusterRole{}, client.HasLabels{greenhouseapis.LabelKeyRole}),
	)
	return errors.Join(errs...)
}
