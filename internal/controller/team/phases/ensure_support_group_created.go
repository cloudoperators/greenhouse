// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

// ensureSupportGroupResources reconciles the SA, Role, and RoleBinding for support-group teams.
// It is a no-op when the support-group label is absent.
func (p *Phase) ensureSupportGroupResources(team *greenhousev1alpha1.Team) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		if team.Labels[greenhouseapis.LabelKeySupportGroup] != "true" {
			return lifecycle.Continue(), nil
		}
		if err := reconcileSupportGroupServiceAccount(ctx, p.Client, p.Recorder, team); err != nil {
			return lifecycle.Break(), err
		}
		if err := reconcileSupportGroupRole(ctx, p.Client, p.Recorder, team); err != nil {
			return lifecycle.Break(), err
		}
		if err := reconcileSupportGroupRoleBinding(ctx, p.Client, p.Recorder, team); err != nil {
			return lifecycle.Break(), err
		}
		return lifecycle.Continue(), nil
	}
}

func reconcileSupportGroupServiceAccount(ctx context.Context, c client.Client, recorder events.EventRecorder, team *greenhousev1alpha1.Team) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      team.Name + "-sa",
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, c, serviceAccount, func() error {
		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}
		serviceAccount.Labels[greenhouseapis.LabelKeyOwnedBy] = team.Name
		return controllerutil.SetControllerReference(team, serviceAccount, c.Scheme())
	})
	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "CreatedServiceAccount", "reconciling support group service account", "Created ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "UpdatedServiceAccount", "reconciling support group service account", "Updated ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	}
	return nil
}

func reconcileSupportGroupRole(ctx context.Context, c client.Client, recorder events.EventRecorder, team *greenhousev1alpha1.Team) error {
	saName := team.Name + "-sa"
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      team.Name + "-sa-token-request",
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, c, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"serviceaccounts/token"},
				Verbs:         []string{"create"},
				ResourceNames: []string{saName},
			},
		}
		return controllerutil.SetControllerReference(team, role, c.Scheme())
	})
	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group role", "name", role.Name, "namespace", role.Namespace)
		recorder.Eventf(team, role, corev1.EventTypeNormal, "CreatedRole", "reconciling support group role", "Created Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group role", "name", role.Name, "namespace", role.Namespace)
		recorder.Eventf(team, role, corev1.EventTypeNormal, "UpdatedRole", "reconciling support group role", "Updated Role %s/%s", role.Namespace, role.Name)
	}
	return nil
}

func reconcileSupportGroupRoleBinding(ctx context.Context, c client.Client, recorder events.EventRecorder, team *greenhousev1alpha1.Team) error {
	saName := team.Name + "-sa"
	roleName := team.Name + "-sa-token-request"
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, c, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     "support-group:" + team.Name,
			},
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: team.Namespace,
			},
		}
		return controllerutil.SetControllerReference(team, roleBinding, c.Scheme())
	})
	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "CreatedRoleBinding", "reconciling support group role binding", "Created RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "UpdatedRoleBinding", "reconciling support group role binding", "Updated RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	}
	return nil
}
