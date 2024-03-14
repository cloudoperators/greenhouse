// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// CAMReconciler reconciles a Team object
type CAMReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *CAMReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Team{}).
		Complete(r)
}

func (r *CAMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// ctx = clientutil.LogIntoContextFromRequest(ctx, req)

	// TODO: Reconcile CAM profiles, access levels for team.

	return ctrl.Result{}, nil
}
