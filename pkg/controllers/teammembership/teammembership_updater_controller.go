package teammembership

import (
	"context"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

type TeamMembershipUpdaterController struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamMembershipUpdaterController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhouseapisv1alpha1.TeamMembership{}).
		Complete(r)
}

func (r *TeamMembershipUpdaterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}
