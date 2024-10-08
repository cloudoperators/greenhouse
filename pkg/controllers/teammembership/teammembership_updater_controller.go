// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		greenhousev1alpha1.ScimAccessReadyCondition,
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
	var organization = new(greenhousev1alpha1.Organization)
	if err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, organization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var team = new(greenhousev1alpha1.Team)
	if err := r.Get(ctx, req.NamespacedName, team); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var teamMembership = new(greenhousev1alpha1.TeamMembership)
	err := r.Get(ctx, types.NamespacedName{Name: team.Name, Namespace: team.Namespace}, teamMembership)
	if !apierrors.IsNotFound(err) && err != nil {
		return ctrl.Result{}, err
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
			return ctrl.Result{}, err
		} else {
			log.FromContext(ctx).Info("Team does not have MappedIdpGroup set", "team", team.Name)
			return ctrl.Result{}, nil
		}
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

	// Ignore organizations without SCIM configuration.
	if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
		log.FromContext(ctx).Info("SCIM config is missing from org", "Name", req.NamespacedName)

		c := greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ScimAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "SCIM config is missing from organization")
		teamMembershipStatus.SetConditions(c)

		return ctrl.Result{}, nil
	}

	scimClient, err := r.createScimClient(ctx, req.Namespace, &teamMembershipStatus, organization.Spec.Authentication.SCIMConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	users, err := r.getUsersFromScim(scimClient, team.Spec.MappedIDPGroup)
	if err != nil {
		log.FromContext(ctx).Info("failed processing team-membership for team", "error", err)
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ScimAccessReadyCondition, greenhousev1alpha1.ScimRequestFailedReason, ""))
		return ctrl.Result{}, err
	}

	teamMembershipStatus.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ScimAccessReadyCondition, "", ""))

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
		return ctrl.Result{}, err
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
	teamMembershipStatus.SetConditions(greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ScimAccessReadyCondition, "", ""))
	return ctrl.Result{RequeueAfter: TeamMembershipRequeueInterval}, nil
}

func (r *TeamMembershipUpdaterController) createScimClient(
	ctx context.Context,
	namespace string,
	teamMembershipStatus *greenhousev1alpha1.TeamMembershipStatus,
	scimConfig *greenhousev1alpha1.SCIMConfig,
) (*scim.ScimClient, error) {

	basicAuthUser, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthUser.Secret)
	if err != nil {
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ScimAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "BasicAuthUser missing"))
		return nil, err
	}
	basicAuthPw, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthPw.Secret)
	if err != nil {
		teamMembershipStatus.SetConditions(greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ScimAccessReadyCondition, greenhousev1alpha1.SecretNotFoundReason, "BasicAuthPw missing"))
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

func (r *TeamMembershipUpdaterController) getUsersFromScim(scimClient *scim.ScimClient, mappedIDPGroup string) ([]greenhousev1alpha1.User, error) {
	members, err := scimClient.GetTeamMembers(mappedIDPGroup)
	if err != nil {
		return nil, err
	}
	users := scimClient.GetUsers(members)
	return users, nil
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

	if conditions.GetConditionByType(greenhousev1alpha1.ScimAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "SCIM access not ready"
		return readyCondition
	}
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}
