// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// OrganizationReconciler reconciles an Organization object
type OrganizationReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&corev1.Namespace{}).
		Owns(&greenhousesapv1alpha1.Team{}).
		Complete(r)
}

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = clientutil.LogIntoContextFromRequest(ctx, req)

	var org = new(greenhousesapv1alpha1.Organization)
	if err := r.Get(ctx, req.NamespacedName, org); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileNamespace(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileAdminTeam(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileNamespace(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	var namespace = new(corev1.Namespace)
	namespace.Name = org.Name

	result, err := clientutil.CreateOrPatch(ctx, r.Client, namespace, func() error {
		return controllerutil.SetControllerReference(org, namespace, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created namespace", "name", namespace.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedNamespace", "Created namespace %s", namespace.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated namespace", "name", namespace.Name)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedNamespace", "Updated namespace %s", namespace.Name)
	}
	return nil
}

func (r *OrganizationReconciler) reconcileAdminTeam(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	orgAdminTeamName := org.Name + "-admin"
	namespace := org.Name

	var orgAdminTeam = new(greenhousesapv1alpha1.Team)
	err := r.Get(ctx, types.NamespacedName{Name: orgAdminTeamName, Namespace: namespace}, orgAdminTeam)
	if !apierrors.IsNotFound(err) && err != nil {
		return err
	}

	var team = new(greenhousesapv1alpha1.Team)
	team.Name = orgAdminTeamName
	team.Namespace = namespace

	result, err := clientutil.CreateOrPatch(ctx, r.Client, team, func() error {
		team.Spec.Description = "Admin team for the organization"
		team.Spec.MappedIDPGroup = org.Spec.MappedOrgAdminIDPGroup
		return controllerutil.SetControllerReference(org, team, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created org admin team", "name", team.Name, "teamNamespace", namespace)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedTeam", "Created Team %s in namespace %s", team.Name, namespace)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated org admin team", "name", team.Name, "teamNamespace", namespace)
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedTeam", "Updated Team %s in namespace %s", team.Name, namespace)
	}
	return nil
}
