// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/scim"
	"github.com/cloudoperators/greenhouse/internal/util"
)

const RequeueInterval = 10 * time.Minute

var (
	// exposedConditions are the conditions that are exposed in the StatusConditions of the Team.
	exposedConditions = []greenhousemetav1alpha1.ConditionType{
		greenhousemetav1alpha1.ReadyCondition,
		greenhousev1alpha1.SCIMAccessReadyCondition,
		greenhousev1alpha1.SCIMAllMembersValidCondition,
	}
)

type TeamController struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Team{}).
		// If an Organization's .Spec was changed, reconcile relevant Teams.
		Watches(&greenhousev1alpha1.Organization{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllTeamsForOrganization),
			builder.WithPredicates(
				predicate.Or(predicate.GenerationChangedPredicate{}, clientutil.PredicateOrganizationSCIMStatusChange()))).
		Complete(r)
}

func (r *TeamController) enqueueAllTeamsForOrganization(ctx context.Context, o client.Object) []ctrl.Request {
	// Team's namespace corresponds to Organization's name.
	return listTeamsAsReconcileRequests(ctx, r.Client, &client.ListOptions{Namespace: o.GetName()})
}
func listTeamsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var teamList = new(greenhousev1alpha1.TeamList)
	if err := c.List(ctx, teamList, listOpts...); err != nil {
		return nil
	}
	res := make([]ctrl.Request, len(teamList.Items))
	for idx, team := range teamList.Items {
		res[idx] = ctrl.Request{NamespacedName: client.ObjectKeyFromObject(team.DeepCopy())}
	}
	return res
}

func (r *TeamController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Team{}, r, r.setConditions())
}

func (r *TeamController) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		team, ok := resource.(*greenhousev1alpha1.Team)
		if !ok {
			logger.Error(errors.New("resource is not a Plugin"), "status setup failed")
			return
		}

		readyCondition := r.computeReadyCondition(team.Status.StatusConditions)
		team.SetCondition(readyCondition)
	}
}

func (r *TeamController) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *TeamController) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	team, ok := object.(*greenhousev1alpha1.Team)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	initTeamStatus(team)

	var organization = new(greenhousev1alpha1.Organization)
	if err := r.Get(ctx, types.NamespacedName{Name: object.GetNamespace()}, organization); err != nil {
		return ctrl.Result{}, lifecycle.Failed, client.IgnoreNotFound(err)
	}

	if team.Spec.MappedIDPGroup == "" {
		log.FromContext(ctx).Info("Team does not have MappedIdpGroup set", "team", team.Name)
		return ctrl.Result{}, lifecycle.Success, nil
	}

	orgSCIMAPIAvailableCondition := organization.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
	if orgSCIMAPIAvailableCondition == nil || !orgSCIMAPIAvailableCondition.IsTrue() {
		team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
			greenhousev1alpha1.SCIMAPIUnavailableReason, "SCIM API in Organization is unavailable"))
		return ctrl.Result{}, lifecycle.Success, nil
	}

	// Ignore organizations without SCIM configuration.
	if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
		log.FromContext(ctx).Info("SCIM config is missing from org", "Name", organization.Name, "Namespace", organization.Namespace)
		team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
			greenhousev1alpha1.SecretNotFoundReason, "SCIM config is missing from organization"))
		return ctrl.Result{}, lifecycle.Success, nil
	}

	scimClient, err := r.createSCIMClient(ctx, team.Namespace, organization.Spec.Authentication.SCIMConfig)
	if err != nil {
		team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
			greenhousev1alpha1.SCIMConfigErrorReason, err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	users, membersValidCondition, err := r.getUsersFromSCIM(ctx, scimClient, team.Spec.MappedIDPGroup)
	if err != nil {
		log.FromContext(ctx).Info("failed getting users from SCIM", "error", err)
		team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
			greenhousev1alpha1.SCIMRequestFailedReason, ""))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	team.SetCondition(membersValidCondition)

	team.Status.Members = users
	team.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAccessReadyCondition, "", ""))

	UpdateTeamMembersCountMetric(team, len(users))

	return ctrl.Result{
			RequeueAfter: wait.Jitter(RequeueInterval, 0.1),
		},
		lifecycle.Success, nil
}

func (r *TeamController) EnsureSuspended(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TeamController) createSCIMClient(
	ctx context.Context,
	namespace string,
	scimConfig *greenhousev1alpha1.SCIMConfig,
) (scim.ISCIMClient, error) {

	clientConfig, err := util.GreenhouseSCIMConfigToSCIMConfig(ctx, r.Client, scimConfig, namespace)
	if err != nil {
		return nil, err
	}

	logger := ctrl.LoggerFrom(ctx)
	return scim.NewSCIMClient(logger, clientConfig)
}

func (r *TeamController) getUsersFromSCIM(ctx context.Context, scimClient scim.ISCIMClient, mappedIDPGroup string) ([]greenhousev1alpha1.User, greenhousemetav1alpha1.Condition, error) {
	condition := greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", "")
	opts := &scim.QueryOptions{
		Filter:     scim.UserFilterByGroupDisplayName(mappedIDPGroup),
		Attributes: scim.SetAttributes(scim.AttrName, scim.AttrEmails, scim.AttrDisplayName, scim.AttrActive),
		StartID:    scim.InitialStartID,
	}
	resources, err := scimClient.GetUsers(ctx, opts)
	if err != nil {
		return nil, condition, err
	}
	users := make([]greenhousev1alpha1.User, 0)
	malformed := 0
	inactive := 0
	for _, resource := range resources {
		user := greenhousev1alpha1.User{
			ID:        resource.UserName,
			FirstName: resource.FirstName(),
			LastName:  resource.LastName(),
			Email:     resource.PrimaryEmail(),
		}
		if user.ID == "" || user.FirstName == "" || user.LastName == "" || user.Email == "" {
			malformed++
			continue
		}

		if !resource.ActiveUser() {
			inactive++
			continue
		}
		users = append(users, user)
	}
	if inactive+malformed > 0 {
		msg := fmt.Sprintf("SCIM members with issues: %d inactive, %d malformed", inactive, malformed)
		condition = greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", msg)
		return users, condition, nil
	}
	condition = greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", "")
	return users, condition, nil
}

// initTeamStatus initializes all empty Team Conditions to "unknown".
func initTeamStatus(team *greenhousev1alpha1.Team) {
	for _, t := range exposedConditions {
		if team.Status.StatusConditions.GetConditionByType(t) == nil {
			team.SetCondition(greenhousemetav1alpha1.UnknownCondition(t, "", ""))
		}
	}
}

func (r *TeamController) computeReadyCondition(
	conditions greenhousemetav1alpha1.StatusConditions,
) (readyCondition greenhousemetav1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	if conditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "SCIM access not ready"
		return readyCondition
	}
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}
