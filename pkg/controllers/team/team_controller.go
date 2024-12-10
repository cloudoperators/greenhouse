package team

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

type TeamReconciler struct {
	client.Client
	recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhouseapisv1alpha1.Team{}).
		Complete(r)
}

func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhouseapisv1alpha1.Team{}, r, r.setStatus())
}

func (r *TeamReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *TeamReconciler) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *TeamReconciler) setStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		team, ok := object.(*greenhouseapisv1alpha1.Team)
		if !ok {
			return
		}

		var members []greenhouseapisv1alpha1.User
		teamMembershipList := new(greenhouseapisv1alpha1.TeamMembershipList)

		err := r.List(ctx, teamMembershipList)
		if err != nil {
			ctrl.Log.Error(err, "Failed to list team memberships")
			return
		}

		for _, member := range teamMembershipList.Items {
			if !hasOwnerReference(member.OwnerReferences, team.Kind, team.Name) {
				continue
			}

			members = append(members, member.Spec.Members...)
		}

		team.Status.Members = members
	}
}

func hasOwnerReference(ownerReferences []v1.OwnerReference, kind, name string) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == kind && ownerReference.Name == name {
			return true
		}
	}

	return false
}
