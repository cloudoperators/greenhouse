// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

const TeamMembershipRequeueInterval = 10 * time.Minute

var (
	membersCountMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_teammembership_members_count",
			Help: "Members count in team membership",
		},
		[]string{"namespace", "team"},
	)
	// exposedConditions are the conditions that are exposed in the StatusConditions of the TeamMembership.
	exposedConditions = []greenhousev1alpha1.ConditionType{
		greenhousev1alpha1.ReadyCondition,
		greenhousev1alpha1.SCIMAccessReadyCondition,
		greenhousev1alpha1.SCIMAllMembersValidCondition,
	}
)

func init() {
	metrics.Registry.MustRegister(membersCountMetric)
}

type TeamMembershipUpdaterController struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamMembershipUpdaterController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Team{}).
		Owns(&greenhousev1alpha1.TeamMembership{}).
		// If an Organization's .Spec was changed, reconcile relevant Teams.
		Watches(&greenhousev1alpha1.Organization{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAllTeamsForOrganization),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *TeamMembershipUpdaterController) enqueueAllTeamsForOrganization(ctx context.Context, o client.Object) []ctrl.Request {
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

func (r *TeamMembershipUpdaterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousev1alpha1.Team{}, r, nil) // status function is nil because it updates the other entity inside.
}

func (r *TeamMembershipUpdaterController) EnsureDeleted(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *TeamMembershipUpdaterController) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	team, ok := object.(*greenhousev1alpha1.Team)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	var organization = new(greenhousev1alpha1.Organization)
	if err := r.Get(ctx, types.NamespacedName{Name: object.GetNamespace()}, organization); err != nil {
		return ctrl.Result{}, lifecycle.Failed, client.IgnoreNotFound(err)
	}

	teamNamespacedName := types.NamespacedName{Namespace: team.Namespace, Name: team.Name}
	var teamMembership = new(greenhousev1alpha1.TeamMembership)
	err := r.Get(ctx, teamNamespacedName, teamMembership)
	if !apierrors.IsNotFound(err) && err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	teamMembershipExists := !apierrors.IsNotFound(err)

	if team.Spec.MappedIDPGroup == "" {
		if teamMembershipExists {
			membersCountMetric.With(prometheus.Labels{
				"namespace": team.Namespace,
				"team":      team.Name,
			}).Set(float64(0))

			log.FromContext(ctx).Info("deleting TeamMembership, Team does not have MappedIdpGroup set", "team-membership", teamMembership.Name)
			err = r.Delete(ctx, teamMembership, &client.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, lifecycle.Failed, err
			}

			return ctrl.Result{}, lifecycle.Success, nil
		}

		log.FromContext(ctx).Info("Team does not have MappedIdpGroup set", "team", team.Name)
		return ctrl.Result{}, lifecycle.Success, nil
	}

	teamMembershipStatus := initTeamMembershipStatus(teamMembership)
	defer func() {
		// Set status only if TM exists.
		if teamMembership.Name != "" {
			if statusErr := r.setStatus(ctx, teamMembership, teamMembershipStatus); statusErr != nil {
				log.FromContext(ctx).Error(statusErr, "failed to set status")
			}
		}
	}()

	orgSCIMAPIAvailableCondition := organization.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
	if orgSCIMAPIAvailableCondition == nil || !orgSCIMAPIAvailableCondition.IsTrue() {
		if teamMembershipExists {
			teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
				greenhousev1alpha1.SCIMAPIUnavailableReason, "SCIM API in Organization is unavailable"))
		}
		return ctrl.Result{}, lifecycle.Success, nil
	}

	// Ignore organizations without SCIM configuration.
	if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
		log.FromContext(ctx).Info("SCIM config is missing from org", "Name", teamNamespacedName)

		c := greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "SCIM config is missing from organization")
		teamMembershipStatus.SetConditions(c)

		return ctrl.Result{}, lifecycle.Success, nil
	}

	scimClient, err := r.createSCIMClient(ctx, team.Namespace, &teamMembershipStatus, organization.Spec.Authentication.SCIMConfig)
	if err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	users, membersValidCondition, err := r.getUsersFromSCIM(scimClient, team.Spec.MappedIDPGroup)
	if err != nil {
		log.FromContext(ctx).Info("failed processing team-membership for team", "error", err)
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition, greenhousev1alpha1.SCIMRequestFailedReason, ""))
		return ctrl.Result{}, lifecycle.Failed, err
	}

	teamMembershipStatus.SetConditions(membersValidCondition, greenhousev1alpha1.TrueCondition(greenhousev1alpha1.SCIMAccessReadyCondition, "", ""))

	membersCountMetric.With(prometheus.Labels{
		"namespace": team.Namespace,
		"team":      team.Name,
	}).Set(float64(len(users)))

	teamMembership.Namespace = team.Namespace
	teamMembership.Name = team.Name
	result, err := clientutil.CreateOrPatch(ctx, r.Client, teamMembership, func() error {
		teamMembership.Spec.Members = users
		return controllerutil.SetOwnerReference(team, teamMembership, r.Scheme())
	})
	if err != nil {
		log.FromContext(ctx).Info("failed processing team-membership for team", "error", err)
		return ctrl.Result{}, lifecycle.Failed, err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created team-membership", "name", teamMembership.Name, "members count", len(teamMembership.Spec.Members))
		r.recorder.Eventf(teamMembership, corev1.EventTypeNormal, "CreatedTeamMembership", "Created TeamMembership %s", teamMembership.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated team-membership", "name", teamMembership.Name, "members count", len(teamMembership.Spec.Members))
		r.recorder.Eventf(teamMembership, corev1.EventTypeNormal, "UpdatedTeamMembership", "Updated TeamMembership %s", teamMembership.Name)
	}

	now := metav1.NewTime(time.Now())
	teamMembershipStatus.LastChangedTime = &now
	teamMembershipStatus.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.SCIMAccessReadyCondition, "", ""))
	return ctrl.Result{
			RequeueAfter: wait.Jitter(TeamMembershipRequeueInterval, 0.1),
		},
		lifecycle.Success, nil
}

func (r *TeamMembershipUpdaterController) createSCIMClient(
	ctx context.Context,
	namespace string,
	teamMembershipStatus *greenhousev1alpha1.TeamMembershipStatus,
	scimConfig *greenhousev1alpha1.SCIMConfig,
) (*scim.ScimClient, error) {

	basicAuthUser, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthUser.Secret)
	if err != nil {
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "BasicAuthUser missing"))
		return nil, err
	}
	basicAuthPw, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthPw.Secret)
	if err != nil {
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "BasicAuthPw missing"))
		return nil, err
	}
	clientConfig := scim.Config{
		RawURL:   scimConfig.BaseURL,
		AuthType: scim.Basic,
		BasicAuthConfig: &scim.BasicAuthConfig{
			BasicAuthUser: basicAuthUser,
			BasicAuthPw:   basicAuthPw,
		},
	}
	return scim.NewScimClient(clientConfig)
}

func (r *TeamMembershipUpdaterController) getUsersFromSCIM(scimClient *scim.ScimClient, mappedIDPGroup string) ([]greenhousev1alpha1.User, greenhousev1alpha1.Condition, error) {
	condition := greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", "")
	members, err := scimClient.GetTeamMembers(mappedIDPGroup)
	if err != nil {
		return nil, condition, err
	}
	users, inactive, malformed, err := scimClient.GetUsers(members)
	if err != nil {
		return nil, condition, err
	}
	if inactive+malformed > 0 {
		msg := fmt.Sprintf("SCIM members with issues: %d inactive, %d malformed", inactive, malformed)
		condition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", msg)
		return users, condition, nil
	}
	condition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.SCIMAllMembersValidCondition, "", "")
	return users, condition, nil
}

func initTeamMembershipStatus(teamMembership *greenhousev1alpha1.TeamMembership) greenhousev1alpha1.TeamMembershipStatus {
	teamMembershipStatus := teamMembership.Status.DeepCopy()
	for _, t := range exposedConditions {
		if teamMembershipStatus.GetConditionByType(t) == nil {
			teamMembershipStatus.SetConditions(greenhousev1alpha1.UnknownCondition(t, "", ""))
		}
	}
	return *teamMembershipStatus
}

func (r *TeamMembershipUpdaterController) setStatus(ctx context.Context, teamMembership *greenhousev1alpha1.TeamMembership, teamMembershipStatus greenhousev1alpha1.TeamMembershipStatus) error {
	readyCondition := r.computeReadyCondition(teamMembershipStatus.StatusConditions)
	teamMembershipStatus.StatusConditions.SetConditions(readyCondition)
	_, err := clientutil.PatchStatus(ctx, r.Client, teamMembership, func() error {
		teamMembership.Status = teamMembershipStatus
		return nil
	})
	return err
}

func (r *TeamMembershipUpdaterController) computeReadyCondition(
	conditions greenhousev1alpha1.StatusConditions,
) (readyCondition greenhousev1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)

	if conditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "SCIM access not ready"
		return readyCondition
	}
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}
