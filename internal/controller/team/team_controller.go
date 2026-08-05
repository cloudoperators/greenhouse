// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/scim"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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
	recorder events.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get
//+kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *TeamController) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorder(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Team{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
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

func (r *TeamController) EnsureDeleted(_ context.Context, obj lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	team := obj.(*greenhousev1alpha1.Team)
	deleteTeamMembersCountMetric(team)
	return ctrl.Result{}, lifecycle.Success, nil
}

func (r *TeamController) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	team, ok := object.(*greenhousev1alpha1.Team)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	initTeamStatus(team)

	// Clean up support-group resources immediately if the label is not set, regardless of SCIM readiness.
	if team.Labels[greenhouseapis.LabelKeySupportGroup] != "true" {
		if err := r.ensureSupportGroupResourcesDeleted(ctx, team); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}

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

	updateTeamMembersCountMetric(team, len(users))

	// Reconcile ServiceAccount, Role, and RoleBinding for support-group teams.
	if team.Labels[greenhouseapis.LabelKeySupportGroup] == "true" {
		if err := r.reconcileSupportGroupServiceAccount(ctx, team); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
		if err := r.reconcileSupportGroupRole(ctx, team); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
		if err := r.reconcileSupportGroupRoleBinding(ctx, team); err != nil {
			return ctrl.Result{}, lifecycle.Failed, err
		}
	}

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

func (r *TeamController) reconcileSupportGroupServiceAccount(ctx context.Context, team *greenhousev1alpha1.Team) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      team.Name + "-sa",
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, serviceAccount, func() error {
		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}
		serviceAccount.Labels[greenhouseapis.LabelKeyOwnedBy] = team.Name
		return controllerutil.SetControllerReference(team, serviceAccount, r.Scheme())
	})

	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		r.recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "CreatedServiceAccount", "reconciling support group service account", "Created ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
		r.recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "UpdatedServiceAccount", "reconciling support group service account", "Updated ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	}

	return nil
}

func (r *TeamController) reconcileSupportGroupRole(ctx context.Context, team *greenhousev1alpha1.Team) error {
	saName := team.Name + "-sa"
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      team.Name + "-sa-token-request",
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"serviceaccounts/token"},
				Verbs:         []string{"create"},
				ResourceNames: []string{saName},
			},
		}
		return controllerutil.SetControllerReference(team, role, r.Scheme())
	})
	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group role", "name", role.Name, "namespace", role.Namespace)
		r.recorder.Eventf(team, role, corev1.EventTypeNormal, "CreatedRole", "reconciling support group role", "Created Role %s/%s", role.Namespace, role.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group role", "name", role.Name, "namespace", role.Namespace)
		r.recorder.Eventf(team, role, corev1.EventTypeNormal, "UpdatedRole", "reconciling support group role", "Updated Role %s/%s", role.Namespace, role.Name)
	}

	return nil
}

func (r *TeamController) reconcileSupportGroupRoleBinding(ctx context.Context, team *greenhousev1alpha1.Team) error {
	saName := team.Name + "-sa"
	roleName := team.Name + "-sa-token-request"
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: team.Namespace,
		},
	}

	result, err := clientutil.CreateOrPatch(ctx, r.Client, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     "support-group:" + team.Name,
			},
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: team.Namespace,
			},
		}
		return controllerutil.SetControllerReference(team, roleBinding, r.Scheme())
	})
	if err != nil {
		return err
	}

	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		r.recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "CreatedRoleBinding", "reconciling support group role binding", "Created RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
		r.recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "UpdatedRoleBinding", "reconciling support group role binding", "Updated RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
	}

	return nil
}

func (r *TeamController) ensureSupportGroupResourcesDeleted(ctx context.Context, team *greenhousev1alpha1.Team) error {
	resourceName := team.Name + "-sa-token-request"

	roleBinding := &rbacv1.RoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: team.Namespace}, roleBinding); err == nil {
		if err := r.Delete(ctx, roleBinding); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group role binding", "name", roleBinding.Name, "namespace", roleBinding.Namespace)
			r.recorder.Eventf(team, roleBinding, corev1.EventTypeNormal, "DeletedRoleBinding", "support-group label removed", "Deleted RoleBinding %s/%s", roleBinding.Namespace, roleBinding.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	role := &rbacv1.Role{}
	if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: team.Namespace}, role); err == nil {
		if err := r.Delete(ctx, role); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group role", "name", role.Name, "namespace", role.Namespace)
			r.recorder.Eventf(team, role, corev1.EventTypeNormal, "DeletedRole", "support-group label removed", "Deleted Role %s/%s", role.Namespace, role.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	serviceAccount := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Name: team.Name + "-sa", Namespace: team.Namespace}, serviceAccount); err == nil {
		if err := r.Delete(ctx, serviceAccount); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
		} else {
			log.FromContext(ctx).Info("deleted support group service account", "name", serviceAccount.Name, "namespace", serviceAccount.Namespace)
			r.recorder.Eventf(team, serviceAccount, corev1.EventTypeNormal, "DeletedServiceAccount", "support-group label removed", "Deleted ServiceAccount %s/%s", serviceAccount.Namespace, serviceAccount.Name)
		}
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
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
