// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"errors"
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
			return breakInvalidKubeConfig(cluster, err)
		}
		err = deleteTeamRBAC(ctx, remoteClient)
		switch {
		case err == nil, apierrors.IsUnauthorized(err), apierrors.IsForbidden(err):
			return lifecycle.Continue(), nil
		case !errors.As(err, new(apierrors.APIStatus)):
			return breakInvalidKubeConfig(cluster, err)
		default:
			return lifecycle.Break(), err
		}
	}
}

func deleteTeamRBAC(ctx context.Context, remoteClient client.Client) error {
	roleBindings := &rbacv1.RoleBindingList{}
	if err := remoteClient.List(ctx, roleBindings, client.HasLabels{greenhouseapis.LabelKeyRoleBinding}); err != nil {
		return err
	}
	for _, roleBinding := range roleBindings.Items {
		if err := remoteClient.Delete(ctx, &roleBinding); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	if err := remoteClient.DeleteAllOf(ctx, &rbacv1.ClusterRoleBinding{}, client.HasLabels{greenhouseapis.LabelKeyRoleBinding}); err != nil {
		return err
	}
	return remoteClient.DeleteAllOf(ctx, &rbacv1.ClusterRole{}, client.HasLabels{greenhouseapis.LabelKeyRole})
}
