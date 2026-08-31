// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureClusterRoleBinding(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: utils.ServiceAccountName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      utils.ServiceAccountName,
					Namespace: cluster.GetNamespace(),
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     utils.CRoleKind,
				Name:     utils.CRoleRef,
				APIGroup: rbacv1.GroupName,
			},
		}

		result, err := controllerutil.CreateOrPatch(ctx, p.RemoteClient, crb, func() error {
			return nil
		})
		if err != nil {
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("created clusterRoleBinding", "cluster", cluster.Name)
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated clusterRoleBinding", "cluster", cluster.Name)
		}
		p.crb = crb
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) ensureNamespace(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		ns := &corev1.Namespace{}
		ns.Name = cluster.GetNamespace()

		result, err := controllerutil.CreateOrPatch(ctx, p.RemoteClient, ns, func() error {
			return nil
		})
		if err != nil {
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("created namespace", "cluster", cluster.Name, "namespace", ns.Name)
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated namespace", "cluster", cluster.Name, "namespace", ns.Name)
		}
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) ensureServiceAccount(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		sa := utils.NewServiceAccount(utils.ServiceAccountName, cluster.GetNamespace())

		result, err := controllerutil.CreateOrPatch(ctx, p.RemoteClient, sa, func() error {
			if p.crb != nil {
				return controllerutil.SetOwnerReference(p.crb, sa, p.RemoteClient.Scheme())
			}
			return nil
		})
		if err != nil {
			return lifecycle.Break(), err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.FromContext(ctx).Info("created serviceAccount", "cluster", cluster.Name, "name", sa.Name)
		case controllerutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated serviceAccount", "cluster", cluster.Name, "name", sa.Name)
		}
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) ensureManagedResourcesDeployed(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		ns := &corev1.Namespace{}
		if err := p.RemoteClient.Get(ctx, client.ObjectKey{Name: cluster.GetNamespace()}, ns); err != nil {
			if apierrors.IsNotFound(err) {
				cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
					greenhousev1alpha1.ManagedResourcesDeployed, "", "namespace not found on remote cluster",
				))
				return lifecycle.Continue(), nil
			}
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		sa := &corev1.ServiceAccount{}
		if err := p.RemoteClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName, Namespace: cluster.GetNamespace()}, sa); err != nil {
			if apierrors.IsNotFound(err) {
				cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
					greenhousev1alpha1.ManagedResourcesDeployed, "", "service account not found on remote cluster",
				))
				return lifecycle.Continue(), nil
			}
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		crb := &rbacv1.ClusterRoleBinding{}
		if err := p.RemoteClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName}, crb); err != nil {
			if apierrors.IsNotFound(err) {
				cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
					greenhousev1alpha1.ManagedResourcesDeployed, "", "clusterrolebinding not found on remote cluster",
				))
				return lifecycle.Continue(), nil
			}
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.ManagedResourcesDeployed, "", "",
		))
		return lifecycle.Continue(), nil
	}
}

func (p *Phase) deleteClusterRoleBinding() lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		crb := &rbacv1.ClusterRoleBinding{}
		err := p.RemoteClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName}, crb)
		if err != nil {
			if apierrors.IsNotFound(err) || apierrors.IsUnauthorized(err) || apierrors.IsForbidden(err) {
				return lifecycle.Continue(), nil
			}
			return lifecycle.Break(), err
		}
		if err := p.RemoteClient.Delete(ctx, crb); err != nil {
			if !apierrors.IsNotFound(err) && !apierrors.IsUnauthorized(err) && !apierrors.IsForbidden(err) {
				return lifecycle.Break(), err
			}
		}
		return lifecycle.Continue(), nil
	}
}
