// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// ensureCleanupIfNotSupportGroup deletes support-group resources when the label is absent.
// It is a no-op when the support-group label is set.
func (p *Phase) ensureCleanupIfNotSupportGroup(team *greenhousev1alpha1.Team) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if team.Labels[greenhouseapis.LabelKeySupportGroup] == "true" {
			return lifecycle.Continue(), nil
		}
		if err := deleteSupportGroupResources(ctx, p.Client, p.Recorder, team); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

// ensureSupportGroupResourcesDeleted is the delete-phase subroutine: always cleans up.
func (p *Phase) ensureSupportGroupResourcesDeleted(team *greenhousev1alpha1.Team) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if err := deleteSupportGroupResources(ctx, p.Client, p.Recorder, team); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

func deleteSupportGroupResources(ctx context.Context, c client.Client, recorder events.EventRecorder, team *greenhousev1alpha1.Team) error {
	resourceName := team.Name + "-sa-token-request"

	roleBinding := &rbacv1.RoleBinding{}
	if err := c.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: team.Namespace}, roleBinding); err == nil {
		if err := c.Delete(ctx, roleBinding); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
			recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "DeletedRoleBinding", "support-group label removed", "Deleted RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	role := &rbacv1.Role{}
	if err := c.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: team.Namespace}, role); err == nil {
		if err := c.Delete(ctx, role); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group role", "name", role.Name, "namespace", role.Namespace)
			recorder.Eventf(team, role, corev1.EventTypeNormal, "DeletedRole", "support-group label removed", "Deleted Role %s/%s", role.Namespace, role.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	serviceAccount := &corev1.ServiceAccount{}
	if err := c.Get(ctx, types.NamespacedName{Name: team.Name + "-sa", Namespace: team.Namespace}, serviceAccount); err == nil {
		if err := c.Delete(ctx, serviceAccount); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
			recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "DeletedServiceAccount", "support-group label removed", "Deleted ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}
