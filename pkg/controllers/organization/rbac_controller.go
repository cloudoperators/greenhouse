// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/rbac"
)

type userGroup int

const (
	admin userGroup = iota
	member
)

// RBACReconciler reconciles an Organization object and manages RBAC permissions based on organization and team membership.
type RBACReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/finalizers,verbs=update
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *RBACReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhouseapisv1alpha1.Organization{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}

func (r *RBACReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = clientutil.LogIntoContextFromRequest(ctx, req)

	var org = new(greenhouseapisv1alpha1.Organization)
	if err := r.Get(ctx, req.NamespacedName, org); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// NOTE: The below code is intentionally rather explicit for transparency reasons as several Kubernetes resources
	// are involved granting permissions on both cluster and namespace level based on organization, team membership and roles.
	// The PolicyRules can be found in the pkg/rbac/role.

	// RBAC for organization admins for cluster- and namespace-scoped resources.
	if err := r.reconcileClusterRole(ctx, org, admin); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileClusterRoleBinding(ctx, org, admin); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileRole(ctx, org, admin); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileRoleBinding(ctx, org, admin); err != nil {
		return ctrl.Result{}, err
	}

	// RBAC for organization members for cluster- and namespace-scoped resources.
	if err := r.reconcileClusterRole(ctx, org, member); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileClusterRoleBinding(ctx, org, member); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileRole(ctx, org, member); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileRoleBinding(ctx, org, member); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RBACReconciler) reconcileClusterRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var clusterRoleName string
	var clusterRoleRules []rbacv1.PolicyRule

	switch group {
	case admin:
		clusterRoleName = rbac.GetAdminRoleNameForOrganization(org.GetName())
		clusterRoleRules = rbac.MakePolicyRulesForOrganizationAdminClusterRole(org.GetName())
	case member:
		clusterRoleName = rbac.GetOrganizationRoleName(org.GetName())
		clusterRoleRules = rbac.MakePolicyRulesForOrganizationMemberClusterRole(org.GetName())
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
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedClusterRole", "Created ClusterRole %s", clusterRole.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated cluster role ", "name", clusterRole.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedClusterRole", "Updated ClusterRole %s", clusterRole.Name)
	}
	return nil
}

func (r *RBACReconciler) reconcileClusterRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var clusterRoleBindingName = ""

	switch group {
	case admin:
		clusterRoleBindingName = rbac.GetAdminRoleNameForOrganization(org.GetName())
	case member:
		clusterRoleBindingName = rbac.GetOrganizationRoleName(org.GetName())
	default:
		return fmt.Errorf("unknown role %d", group)
	}

	var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
	clusterRoleBinding.Namespace = ""
	clusterRoleBinding.Name = clusterRoleBindingName

	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
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
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedClusterRoleBinding", "Created ClusterRoleBinding %s", clusterRoleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated cluster role binding", "name", clusterRoleBinding.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedClusterRoleBinding", "Updated ClusterRoleBinding %s", clusterRoleBinding.Name)
	}
	return nil
}

func (r *RBACReconciler) reconcileRole(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var roleName string
	var roleRules []rbacv1.PolicyRule

	switch group {
	case admin:
		roleName = rbac.GetAdminRoleNameForOrganization(org.GetName())
		roleRules = rbac.MakePolicyRulesForOrganizationAdminRole()
	case member:
		roleName = rbac.GetOrganizationRoleName(org.GetName())
		roleRules = rbac.MakePolicyRulesForOrganizationMemberRole()
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
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedRole", "Created Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated role ", "namespace", role.Namespace, "name", role.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedRole", "Updated Role %s/%s", role.Namespace, role.Name)
	}
	return nil
}

func (r *RBACReconciler) reconcileRoleBinding(ctx context.Context, org *greenhouseapisv1alpha1.Organization, group userGroup) error {
	var roleBindingName = ""

	switch group {
	case admin:
		roleBindingName = rbac.GetAdminRoleNameForOrganization(org.GetName())
	case member:
		roleBindingName = rbac.GetOrganizationRoleName(org.GetName())
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
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedRoleBinding", "Created RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated role binding", "namespace", roleBinding.Namespace, "name", roleBinding.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedRoleBinding", "Updated RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	}
	return nil
}
