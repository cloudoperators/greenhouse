package teammembership

import (
	"context"
	"errors"
	"sync"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TeamMembershipUpdaterController struct {
	client.Client
	recorder          record.EventRecorder
	ScimBaseURL       string
	ScimBasicAuthUser string
	ScimBasicAuthPw   string
	scimClient        *scim.ScimClient
	teamsByName       map[string]greenhousev1alpha1.Team
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammemberships/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *TeamMembershipUpdaterController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	if r.ScimBaseURL == "" {
		return errors.New("scim base URL required but not provided")
	}
	if r.ScimBasicAuthUser == "" {
		return errors.New("scim basic auth user required but not provided")
	}
	if r.ScimBasicAuthPw == "" {
		return errors.New("scim basic auth pw required but not provided")
	}

	scimConfig := scim.Config{
		RawURL:   r.ScimBaseURL,
		AuthType: scim.Basic,
		BasicAuthConfig: &scim.BasicAuthConfig{
			BasicAuthUser: r.ScimBasicAuthUser,
			BasicAuthPw:   r.ScimBasicAuthPw,
		},
	}
	scimClient, err := scim.NewScimClient(scimConfig)
	if err != nil {
		return err
	}
	r.scimClient = scimClient

	r.teamsByName = make(map[string]greenhousev1alpha1.Team)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.TeamMembership{}).
		Complete(r)
}

// Lists all available teams und updates or creates respective teamMemberships if team.Spec.MappedIDPGroup is present
func (r *TeamMembershipUpdaterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	teamList := greenhousev1alpha1.TeamList{}
	err := r.List(ctx, &teamList, &client.ListOptions{Namespace: req.Namespace})
	if err != nil {
		return ctrl.Result{}, err
	}

	// create org admin team from org spec
	orgAdminTeam, err := r.getOrgAdminTeam(ctx, req.Namespace)
	if err != nil {
		log.FromContext(ctx).Info("[INFO] failed creating org-admin team", "organization", req.Namespace, "error", err)
	} else {
		teamList.Items = append(teamList.Items, orgAdminTeam)
	}

	var wg sync.WaitGroup
	wg.Add(len(teamList.Items))

	for _, team := range teamList.Items {
		r.teamsByName[team.Name] = team

		go func(team greenhousev1alpha1.Team) {
			defer wg.Done()
			err = r.processTeamMembership(ctx, team)
			if err != nil {
				log.FromContext(ctx).Info("[Info] failed processing team-membership for team", "error", err)
			}
		}(team)
	}
	wg.Wait()

	err = r.deleteOrphanedTeamMemberships(ctx, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

func (r *TeamMembershipUpdaterController) getOrgAdminTeam(ctx context.Context, namespace string) (greenhousev1alpha1.Team, error) {
	org := new(greenhousev1alpha1.Organization)
	err := r.Get(ctx, types.NamespacedName{Name: namespace}, org)
	if err != nil {
		return greenhousev1alpha1.Team{}, err
	}

	orgAdminTeam := greenhousev1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: org.Name,
			Name:      org.Name + "-admin",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			MappedIDPGroup: org.Spec.MappedOrgAdminIDPGroup,
		},
	}

	return orgAdminTeam, nil
}

func (r *TeamMembershipUpdaterController) processTeamMembership(ctx context.Context, team greenhousev1alpha1.Team) error {
	teamMembership := new(greenhousev1alpha1.TeamMembership)

	err := r.Get(ctx, types.NamespacedName{Namespace: team.Namespace, Name: team.Name}, teamMembership)
	if !apierrors.IsNotFound(err) && err != nil {
		return err
	}

	teamMembership.Status.LastSyncedTime = &metav1.Time{Time: time.Now()}

	// Delete existing TM for team without mapped idp group
	if team.Spec.MappedIDPGroup == "" {
		if err == nil {
			return r.Delete(ctx, teamMembership, &client.DeleteOptions{})
		}
		return nil
	}

	// create or update TM
	if apierrors.IsNotFound(err) {
		return r.createTeamMembership(ctx, team)
	}
	return r.updateTeamMembership(ctx, teamMembership, team)
}

func (r *TeamMembershipUpdaterController) createTeamMembership(ctx context.Context, team greenhousev1alpha1.Team) error {
	members, err := r.scimClient.GetTeamMembers(team.Spec.MappedIDPGroup)
	if err != nil {
		return err
	}
	users := r.scimClient.GetUsers(members)
	now := metav1.NewTime(time.Now())

	teamMembership := new(greenhousev1alpha1.TeamMembership)
	teamMembership.Namespace = team.Namespace
	teamMembership.Name = team.Name
	teamMembership.Spec.Members = users
	teamMembership.Status.LastChangedTime = &now
	err = r.Create(ctx, teamMembership, &client.CreateOptions{})
	if err != nil {
		return err
	}
	log.FromContext(ctx).Info("created team-membership")
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
	return r.Update(ctx, &team, &client.UpdateOptions{})
}

func (r *TeamMembershipUpdaterController) updateTeamMembership(ctx context.Context, teamMembership *greenhousev1alpha1.TeamMembership, team greenhousev1alpha1.Team) error {
	members, err := r.scimClient.GetTeamMembers(team.Spec.MappedIDPGroup)
	if err != nil {
		return err
	}
	users := r.scimClient.GetUsers(members)
	teamMembership.Spec.Members = users
	err = r.Update(ctx, teamMembership, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	log.FromContext(ctx).Info("updated team-membership")
	now := metav1.NewTime(time.Now())
	teamMembership.Status.LastChangedTime = &now
	err = r.Status().Update(ctx, teamMembership)
	if err != nil {
		return err
	}
	log.Log.Info("updated team-membership status")

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
	return r.Update(ctx, &team, &client.UpdateOptions{})
}

func (r *TeamMembershipUpdaterController) deleteOrphanedTeamMemberships(ctx context.Context, namespace string) error {
	teamMembershipList := greenhousev1alpha1.TeamMembershipList{}
	err := r.List(ctx, &teamMembershipList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return err
	}

	for _, teamMembership := range teamMembershipList.Items {
		_, ok := r.teamsByName[teamMembership.Name]
		if !ok {
			teamMembership := teamMembership
			err = r.Delete(ctx, &teamMembership, &client.DeleteOptions{})
			if err != nil {
				log.FromContext(ctx).Info("[Info] failed deleting orphaned teamMembership", "error", err)
			}
			log.FromContext(ctx).Info("deleted team-membership", "team-membership", teamMembership.Name)
		}
	}
	return nil
}
