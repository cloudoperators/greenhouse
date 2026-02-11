// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/rbac"
)

type userGroup int

const (
	admin userGroup = iota
	member
	clusterAdmin
	pluginAdmin

	clusterRoleKind = "ClusterRole"
)

func (r *OrganizationReconciler) reconcileClusterRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var clusterRoleName string
	var clusterRoleRules []rbacv1.PolicyRule

	switch group {
	case admin:
		clusterRoleName = rbac.OrganizationAdminRoleName(org.GetName())
		clusterRoleRules = rbac.OrganizationAdminClusterRolePolicyRules(org.GetName())
	case member:
		clusterRoleName = rbac.OrganizationRoleName(org.GetName())
		clusterRoleRules = rbac.OrganizationMemberClusterRolePolicyRules(org.GetName())
	default:
		return fmt.Errorf("unknown userRole %d", group)
	}

	var clusterRole = new(rbacv1.ClusterRole)
	clusterRole.Namespace = ""
	clusterRole.Name = clusterRoleName

	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterRole, func() error {
		clusterRole.AggregationRule = nil
		clusterRole.Rules = clusterRoleRules
		return controllerutil.SetOwnerReference(org, clusterRole, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created cluster role ", "name", clusterRole.Name)
		r.recorder.Eventf(org, clusterRole, corev1.EventTypeNormal, "CreatedClusterRole", "reconciling Organization", "Created ClusterRole %s", clusterRole.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated cluster role ", "name", clusterRole.Name)
		r.recorder.Eventf(org, clusterRole, corev1.EventTypeNormal, "UpdatedClusterRole", "reconciling Organization", "Updated ClusterRole %s", clusterRole.Name)
	}
	return nil
}

func (r *OrganizationReconciler) reconcileClusterRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var clusterRoleBindingName = ""

	switch group {
	case admin:
		clusterRoleBindingName = rbac.OrganizationAdminRoleName(org.GetName())
	case member:
		clusterRoleBindingName = rbac.OrganizationRoleName(org.GetName())
	default:
		return fmt.Errorf("unknown role %d", group)
	}

	var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
	clusterRoleBinding.Namespace = ""
	clusterRoleBinding.Name = clusterRoleBindingName

	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     clusterRoleKind,
			Name:     clusterRoleBindingName,
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     clusterRoleBindingName,
			},
		}
		return controllerutil.SetOwnerReference(org, clusterRoleBinding, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created cluster role binding", "name", clusterRoleBinding.Name)
		r.recorder.Eventf(org, clusterRoleBinding, corev1.EventTypeNormal, "CreatedClusterRoleBinding", "reconciling Organization", "Created ClusterRoleBinding %s", clusterRoleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated cluster role binding", "name", clusterRoleBinding.Name)
		r.recorder.Eventf(org, clusterRoleBinding, corev1.EventTypeNormal, "UpdatedClusterRoleBinding", "reconciling Organization", "Updated ClusterRoleBinding %s", clusterRoleBinding.Name)
	}
	return nil
}

func (r *OrganizationReconciler) reconcileRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var roleName string
	var roleRules []rbacv1.PolicyRule

	switch group {
	case admin:
		roleName = rbac.OrganizationAdminRoleName(org.GetName())
		roleRules = rbac.OrganizationAdminPolicyRules()
	case member:
		roleName = rbac.OrganizationRoleName(org.GetName())
		roleRules = rbac.OrganizationMemberPolicyRules()
	case clusterAdmin:
		roleName = rbac.OrganizationClusterAdminRoleName(org.GetName())
		roleRules = rbac.OrganizationClusterAdminPolicyRules()
	case pluginAdmin:
		roleName = rbac.OrganizationPluginAdminRoleName(org.GetName())
		roleRules = rbac.OrganizationPluginAdminPolicyRules()
	default:
		return fmt.Errorf("unknown userRole %d", group)
	}

	var role = new(rbacv1.Role)
	role.Namespace = org.GetName()
	role.Name = roleName

	result, err := clientutil.CreateOrPatch(ctx, r.Client, role, func() error {
		role.Rules = roleRules
		return controllerutil.SetOwnerReference(org, role, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created role ", "namespace", role.Namespace, "name", role.Name)
		r.recorder.Eventf(org, role, corev1.EventTypeNormal, "CreatedRole", "reconciling Organization", "Created Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated role ", "namespace", role.Namespace, "name", role.Name)
		r.recorder.Eventf(org, role, corev1.EventTypeNormal, "UpdatedRole", "reconciling Organization", "Updated Role %s/%s", role.Namespace, role.Name)
	}
	return nil
}

func (r *OrganizationReconciler) reconcileRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var roleBindingName = ""

	switch group {
	case admin:
		roleBindingName = rbac.OrganizationAdminRoleName(org.GetName())
	case member:
		roleBindingName = rbac.OrganizationRoleName(org.GetName())
	default:
		return fmt.Errorf("unknown userRole %d", group)
	}

	var roleBinding = new(rbacv1.RoleBinding)
	roleBinding.Namespace = org.GetName()
	roleBinding.Name = roleBindingName

	result, err := clientutil.CreateOrPatch(ctx, r.Client, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleBindingName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     roleBindingName,
			},
		}
		return controllerutil.SetOwnerReference(org, roleBinding, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created role binding", "namespace", roleBinding.Namespace, "name", roleBinding.Name)
		r.recorder.Eventf(org, roleBinding, corev1.EventTypeNormal, "CreatedRoleBinding", "reconciling Organization", "Created RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated role binding", "namespace", roleBinding.Namespace, "name", roleBinding.Name)
		r.recorder.Eventf(org, roleBinding, corev1.EventTypeNormal, "UpdatedRoleBinding", "reconciling Organization", "Updated RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	}
	return nil
}

// reconcileCatalogServiceAccount creates the ServiceAccount for PluginDefinitionCatalog operations.
func (r *OrganizationReconciler) reconcileCatalogServiceAccount(ctx context.Context, org *greenhouseapisv1alpha1.Organization) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbac.OrgCatalogServiceAccountName(org.Name),
			Namespace: org.Name,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, serviceAccount, func() error {
		return controllerutil.SetOwnerReference(org, serviceAccount, r.Scheme())
	})
	if err != nil {
		r.recorder.Eventf(serviceAccount, org, corev1.EventTypeWarning, "FailedCatalogSA", "reconciling ServiceAccount for Catalog", "Failed to create or update ServiceAccount %s/%s: %s", serviceAccount.Namespace, serviceAccount.Name, err.Error())
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created catalog service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		r.recorder.Eventf(serviceAccount, org, corev1.EventTypeNormal, "CreatedCatalogSA", "reconciling ServiceAccount for Catalog", "Created ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated catalog service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		r.recorder.Eventf(serviceAccount, org, corev1.EventTypeNormal, "UpdatedCatalogSA", "reconciling ServiceAccount for Catalog", "Updated ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("no changes made to catalog service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
	default:
		log.FromContext(ctx).Info("unknown operation result for catalog service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace, "result", result)
	}

	return nil
}

// reconcileCatalogClusterRole creates the ClusterRole for Greenhouse organization's PluginDefinitionCatalog operations.
func (r *OrganizationReconciler) reconcileCatalogClusterRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, resources []string) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbac.OrgCatalogRoleName(org.Name),
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterRole, func() error {
		clusterRole.Rules = rbac.OrgCatalogPolicyRules(resources)
		return controllerutil.SetOwnerReference(org, clusterRole, r.Scheme())
	})
	if err != nil {
		r.recorder.Eventf(org, clusterRole, corev1.EventTypeWarning, "FailedCatalogCRole", "reconciling Organization", "Failed to create or update ClusterRole %s: %s", clusterRole.Name, err.Error())
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created catalog cluster role", "org", org.Name, "name", clusterRole.Name)
		r.recorder.Eventf(org, clusterRole, corev1.EventTypeNormal, "CreatedCatalogCRole", "reconciling Organization", "Created ClusterRole %s", clusterRole.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated catalog cluster role", "org", org.Name, "name", clusterRole.Name)
		r.recorder.Eventf(org, clusterRole, corev1.EventTypeNormal, "UpdatedCatalogCRole", "reconciling Organization", "Updated ClusterRole %s", clusterRole.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("no changes made to catalog cluster role", "org", org.Name, "name", clusterRole.Name)
	default:
		log.FromContext(ctx).Info("unknown operation result for catalog cluster role", "org", org.Name, "name", clusterRole.Name, "result", result)
	}

	return nil
}

// reconcileCatalogClusterRoleBinding creates the ClusterRoleBinding for Greenhouse organization's PluginDefinitionCatalog operations.
func (r *OrganizationReconciler) reconcileCatalogClusterRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbac.OrgCatalogRoleName(org.Name),
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     clusterRoleKind,
			Name:     rbac.OrgCatalogRoleName(org.Name),
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      rbac.OrgCatalogServiceAccountName(org.Name),
				Namespace: org.Name,
			},
		}
		return controllerutil.SetOwnerReference(org, clusterRoleBinding, r.Scheme())
	})
	if err != nil {
		r.recorder.Eventf(org, clusterRoleBinding, corev1.EventTypeWarning, "FailedCatalogCRoleBinding", "reconciling Organization", "Failed to create or update ClusterRoleBinding %s: %s", clusterRoleBinding.Name, err.Error())
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created catalog cluster roleBinding", "name", clusterRoleBinding.Name)
		r.recorder.Eventf(org, clusterRoleBinding, corev1.EventTypeNormal, "CreatedCatalogCRoleBinding", "reconciling Organization", "Created ClusterRoleBinding %s", clusterRoleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated catalog cluster roleBinding", "name", clusterRoleBinding.Name)
		r.recorder.Eventf(org, clusterRoleBinding, corev1.EventTypeNormal, "UpdatedCatalogCRoleBinding", "reconciling Organization", "Updated ClusterRoleBinding %s", clusterRoleBinding.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("no changes made to catalog cluster roleBinding", "name", clusterRoleBinding.Name)
	default:
		log.FromContext(ctx).Info("unknown operation result for catalog cluster roleBinding", "name", clusterRoleBinding.Name, "result", result)
	}

	return nil
}

// reconcileCatalogRole creates the Role for Greenhouse organization's PluginDefinitionCatalog operations.
func (r *OrganizationReconciler) reconcileCatalogRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, resources []string) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbac.OrgCatalogRoleName(org.Name),
			Namespace: org.Name,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, role, func() error {
		role.Rules = rbac.OrgCatalogPolicyRules(resources)
		return controllerutil.SetOwnerReference(org, role, r.Scheme())
	})
	if err != nil {
		r.recorder.Eventf(org, role, corev1.EventTypeWarning, "FailedCatalogRole", "reconciling Organization", "Failed to create or update Role %s/%s: %s", role.Namespace, role.Name, err.Error())
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created catalog role", "org", org.Name, "name", role.Name, "namespace", role.Namespace)
		r.recorder.Eventf(org, role, corev1.EventTypeNormal, "CreatedCatalogRole", "reconciling Organization", "Created Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated catalog role", "org", org.Name, "name", role.Name, "namespace", role.Namespace)
		r.recorder.Eventf(org, role, corev1.EventTypeNormal, "UpdatedCatalogRole", "reconciling Organization", "Updated Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("no changes made to catalog role", "org", org.Name, "name", role.Name, "namespace", role.Namespace)
	default:
		log.FromContext(ctx).Info("unknown operation result for catalog role", "org", org.Name, "name", role.Name, "namespace", role.Namespace, "result", result)
	}

	return nil
}

// reconcileCatalogRoleBinding creates the RoleBinding for Greenhouse organization's PluginDefinitionCatalog operations.
func (r *OrganizationReconciler) reconcileCatalogRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbac.OrgCatalogRoleName(org.Name),
			Namespace: org.Name,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     rbac.OrgCatalogRoleName(org.Name),
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      rbac.OrgCatalogServiceAccountName(org.Name),
				Namespace: org.Name,
			},
		}
		return controllerutil.SetOwnerReference(org, roleBinding, r.Scheme())
	})
	if err != nil {
		r.recorder.Eventf(org, roleBinding, corev1.EventTypeWarning, "FailedCatalogRoleBinding", "reconciling Organization", "Failed to create or update RoleBinding %s/%s: %s", roleBinding.Namespace, roleBinding.Name, err.Error())
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created catalog roleBinding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		r.recorder.Eventf(org, roleBinding, corev1.EventTypeNormal, "CreatedCatalogRoleBinding", "reconciling Organization", "Created RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated catalog roleBinding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		r.recorder.Eventf(org, roleBinding, corev1.EventTypeNormal, "UpdatedCatalogRoleBinding", "reconciling Organization", "Updated RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultNone:
		log.FromContext(ctx).Info("no changes made to catalog roleBinding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
	default:
		log.FromContext(ctx).Info("unknown operation result for catalog roleBinding", "name", roleBinding.Name, "namespace", roleBinding.Namespace, "result", result)
	}

	return nil
}
