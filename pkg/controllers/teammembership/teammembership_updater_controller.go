// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership

import (
	"context"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const defaultRequeueInterval = 10 * time.Minute

var (
	membersCountMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_teammembership_members_count",
			Help: "Members count in team membership",
		},
		[]string{"namespace", "team"},
	)
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
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamMembershipUpdaterController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Team{}).
		Owns(&greenhousev1alpha1.TeamMembership{}).
		Complete(r)
}

func (r *TeamMembershipUpdaterController) createScimClient(scimConfig *greenhousev1alpha1.SCIMConfig) (*scim.ScimClient, error) {
	clientConfig := scim.Config{
		RawURL:   scimConfig.BaseURL,
		AuthType: scim.Basic,
		BasicAuthConfig: &scim.BasicAuthConfig{
			BasicAuthUser: scimConfig.BasicAuthUser,
			BasicAuthPw:   scimConfig.BasicAuthPw,
		},
	}
	scimClient, err := scim.NewScimClient(clientConfig)
	if err != nil {
		return nil, err
	}
	return scimClient, nil
}

func (r *TeamMembershipUpdaterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var organization = new(greenhousev1alpha1.Organization)
	if err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, organization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ignore organizations without SCIM configuration.
	if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
		log.Log.Info("SCIM config is missing from org", "Name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	scimClient, err := r.createScimClient(organization.Spec.Authentication.SCIMConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	var team greenhousev1alpha1.Team
	err = r.Get(ctx, req.NamespacedName, &team)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var teamMembership greenhousev1alpha1.TeamMembership
	err = r.Get(ctx, types.NamespacedName{Name: team.Name, Namespace: team.Namespace}, &teamMembership)
	if !apierrors.IsNotFound(err) && err != nil {
		return ctrl.Result{}, err
	}

	teamMembershipExists := !apierrors.IsNotFound(err)

	if team.Spec.MappedIDPGroup == "" {
		if teamMembershipExists {
			log.FromContext(ctx).Info("deleting TeamMembership, Team does not have MappedIdpGroup set", "team-membership", teamMembership.Name)
			err = r.Delete(ctx, &teamMembership, &client.DeleteOptions{})
			return ctrl.Result{}, err
		} else {
			log.FromContext(ctx).Info("Team does not have MappedIdpGroup set", "team", team.Name)
			return ctrl.Result{}, nil
		}
	}

	users, err := r.getUsersFromScim(scimClient, team.Spec.MappedIDPGroup)
	if err != nil {
		log.FromContext(ctx).Info("[Info] failed processing team-membership for team", "error", err)
		return ctrl.Result{}, err
	}

	membersCountMetric.With(prometheus.Labels{
		"namespace": team.Namespace,
		"team":      team.Name,
	}).Set(float64(len(users)))

	if teamMembershipExists && len(teamMembership.Spec.Members) == len(users) {
		// Requeue when nothing has changed.
		return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
	}

	if teamMembershipExists {
		err = r.updateTeamMembership(ctx, &team, &teamMembership, users)
	} else {
		err = r.createTeamMembershipForTeam(ctx, &team, users)
	}
	if err != nil {
		log.FromContext(ctx).Info("[Info] failed processing team-membership for team", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

func (r *TeamMembershipUpdaterController) getUsersFromScim(scimClient *scim.ScimClient, mappedIDPGroup string) ([]greenhousev1alpha1.User, error) {
	members, err := scimClient.GetTeamMembers(mappedIDPGroup)
	if err != nil {
		return nil, err
	}
	users := scimClient.GetUsers(members)
	return users, nil
}

func (r *TeamMembershipUpdaterController) createTeamMembershipForTeam(ctx context.Context, team *greenhousev1alpha1.Team, users []greenhousev1alpha1.User) error {
	now := metav1.NewTime(time.Now())

	teamMembership := new(greenhousev1alpha1.TeamMembership)
	teamMembership.Namespace = team.Namespace
	teamMembership.Name = team.Name
	teamMembership.Spec.Members = users
	teamMembership.Status.LastChangedTime = &now
	err := r.Create(ctx, teamMembership, &client.CreateOptions{})
	if err != nil {
		return err
	}
	log.FromContext(ctx).Info("created team-membership",
		"name", teamMembership.Name, "members count", len(teamMembership.Spec.Members))

	return r.updateOwnerReferenceForTeam(ctx, teamMembership, team)
}

func (r *TeamMembershipUpdaterController) updateTeamMembership(ctx context.Context, team *greenhousev1alpha1.Team, teamMembership *greenhousev1alpha1.TeamMembership, users []greenhousev1alpha1.User) error {
	teamMembership.Spec.Members = users
	err := r.Update(ctx, teamMembership, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	now := metav1.NewTime(time.Now())
	teamMembership.Status.LastChangedTime = &now
	err = r.Status().Update(ctx, teamMembership)
	if err != nil {
		return err
	}
	log.FromContext(ctx).Info("updated team-membership and its status",
		"name", teamMembership.Name, "members count", len(teamMembership.Spec.Members))

	return r.updateOwnerReferenceForTeam(ctx, teamMembership, team)
}

func (r *TeamMembershipUpdaterController) updateOwnerReferenceForTeam(ctx context.Context, teamMembership *greenhousev1alpha1.TeamMembership, team *greenhousev1alpha1.Team) error {
	team.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         greenhousev1alpha1.GroupVersion.String(),
			Kind:               "TeamMembership",
			Name:               teamMembership.GetName(),
			UID:                teamMembership.GetUID(),
			Controller:         nil,
			BlockOwnerDeletion: nil,
		},
	}
	return r.Update(ctx, team, &client.UpdateOptions{})
}
