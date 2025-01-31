// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
	dexstore "github.com/cloudoperators/greenhouse/pkg/dex/store"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

var (
	// exposedConditions are the conditions that are exposed in the StatusConditions of the Organization.
	exposedConditions = []greenhousesapv1alpha1.ConditionType{
		greenhousesapv1alpha1.ReadyCondition,
		greenhousesapv1alpha1.SCIMAPIAvailableCondition,
		greenhousesapv1alpha1.ServiceProxyProvisioned,
		greenhousesapv1alpha1.OrganizationOICDConfigured,
		greenhousesapv1alpha1.OrganizationAdminTeamConfigured,
		greenhousesapv1alpha1.ServiceProxyProvisioned,
		greenhousesapv1alpha1.OrganizationDefaultTeamRolesConfigured,
		greenhousesapv1alpha1.NamespaceCreated,
		greenhousesapv1alpha1.OrganizationRBACConfigured,
	}
)

// OrganizationReconciler reconciles an Organization object
type OrganizationReconciler struct {
	client.Client
	recorder  record.EventRecorder
	Dexter    dexstore.Dexter
	Namespace string
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;watch;create;update;patch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teamroles,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dex.coreos.com,resources=connectors;oauth2clients,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	b := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&corev1.Namespace{}).
		Owns(&greenhousesapv1alpha1.Team{}).
		Owns(&greenhousesapv1alpha1.TeamRole{}).
		Owns(&greenhousesapv1alpha1.Plugin{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueOrganizationForReferencedSecret),
			builder.WithPredicates(clientutil.PredicateHasOICDConfigured())).
		Watches(&greenhousesapv1alpha1.PluginDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllOrganizationsForServiceProxyPluginDefinition),
			builder.WithPredicates(predicate.And(
				clientutil.PredicateByName(serviceProxyName),
				predicate.GenerationChangedPredicate{},
			)))
	if r.Dexter.GetBackend() == dexstore.K8s {
		b.Owns(&dexapi.Connector{}).
			Owns(&dexapi.OAuth2Client{})
	}
	return b.Complete(r)
}

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousesapv1alpha1.Organization{}, r, r.setStatus())
}

func (r *OrganizationReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil // nothing to do in that case
}

func (r *OrganizationReconciler) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	org, ok := object.(*greenhousesapv1alpha1.Organization)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	initOrganizationStatus(org)

	if err := r.reconcileNamespace(ctx, org); err != nil {
		org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.NamespaceCreated, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.NamespaceCreated, "", ""))

	if err := r.reconcileRBAC(ctx, org); err != nil {
		org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.OrganizationRBACConfigured, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.OrganizationRBACConfigured, "", ""))

	if err := r.reconcileDefaultTeamRoles(ctx, org); err != nil {
		org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.OrganizationDefaultTeamRolesConfigured, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.OrganizationDefaultTeamRolesConfigured, "", ""))

	if err := r.reconcileServiceProxy(ctx, org); err != nil {
		org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.ServiceProxyProvisioned, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.ServiceProxyProvisioned, "", ""))

	if org.Spec.Authentication != nil && org.Spec.Authentication.OIDCConfig != nil {
		if err := r.reconcileDexConnector(ctx, org); err != nil {
			org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.OrganizationOICDConfigured, greenhousesapv1alpha1.DexReconcileFailed, ""))
			return ctrl.Result{}, lifecycle.Failed, err
		}

		if err := r.reconcileOAuth2Client(ctx, org); err != nil {
			org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.OrganizationOICDConfigured, greenhousesapv1alpha1.OAuthOICDFailed, err.Error()))
			return ctrl.Result{}, lifecycle.Failed, err
		}
		org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.OrganizationOICDConfigured, "", ""))
	}

	if err := r.reconcileAdminTeam(ctx, org); err != nil {
		org.SetCondition(greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.OrganizationAdminTeamConfigured, "", err.Error()))
		return ctrl.Result{}, lifecycle.Failed, err
	}
	org.SetCondition(greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.OrganizationAdminTeamConfigured, "", ""))

	return ctrl.Result{}, lifecycle.Success, nil
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
	namespace := org.Name

	var team = new(greenhousesapv1alpha1.Team)
	team.Name = org.Name + "-admin"
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

func (r *OrganizationReconciler) reconcileRBAC(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	// NOTE: The below code is intentionally rather explicit for transparency reasons as several Kubernetes resources
	// are involved granting permissions on both cluster and namespace level based on organization, team membership and roles.
	// The PolicyRules can be found in the pkg/rbac/role.

	// RBAC for organization admins for cluster- and namespace-scoped resources.
	if err := r.reconcileClusterRole(ctx, org, admin); err != nil {
		return err
	}
	if err := r.reconcileClusterRoleBinding(ctx, org, admin); err != nil {
		return err
	}
	if err := r.reconcileRole(ctx, org, admin); err != nil {
		return err
	}
	if err := r.reconcileRoleBinding(ctx, org, admin); err != nil {
		return err
	}

	// RBAC for organization members for cluster- and namespace-scoped resources.
	if err := r.reconcileClusterRole(ctx, org, member); err != nil {
		return err
	}
	if err := r.reconcileClusterRoleBinding(ctx, org, member); err != nil {
		return err
	}
	if err := r.reconcileRole(ctx, org, member); err != nil {
		return err
	}
	if err := r.reconcileRoleBinding(ctx, org, member); err != nil {
		return err
	}

	// RBAC roles for organization cluster admins to access namespace-scoped resources.
	if err := r.reconcileRole(ctx, org, clusterAdmin); err != nil {
		return err
	}

	// RBAC roles for organization plugin admins to access namespace-scoped resources.
	if err := r.reconcileRole(ctx, org, pluginAdmin); err != nil {
		return err
	}

	return nil
}

func (r *OrganizationReconciler) checkSCIMAPIAvailability(ctx context.Context, org *greenhousesapv1alpha1.Organization) greenhousesapv1alpha1.Condition {
	if org.Spec.Authentication == nil || org.Spec.Authentication.SCIMConfig == nil {
		// SCIM Config is optional.
		return greenhousesapv1alpha1.UnknownCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMConfigNotProvidedReason, "SCIM Config not provided")
	}

	if org.Spec.MappedOrgAdminIDPGroup == "" {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, ".Spec.MappedOrgAdminIDPGroup is not set in Organization")
	}

	namespace := org.Name
	scimConfig := org.Spec.Authentication.SCIMConfig

	basicAuthUser, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthUser.Secret)
	if err != nil {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SecretNotFoundReason, "BasicAuthUser missing")
	}
	basicAuthPw, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, namespace, *scimConfig.BasicAuthPw.Secret)
	if err != nil {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SecretNotFoundReason, "BasicAuthPw missing")
	}
	config := &scim.Config{
		URL:      scimConfig.BaseURL,
		AuthType: scim.Basic,
		BasicAuth: &scim.BasicAuthConfig{
			Username: basicAuthUser,
			Password: basicAuthPw,
		},
	}
	logger := ctrl.LoggerFrom(ctx)
	scimClient, err := scim.NewSCIMClient(logger, config)
	if err != nil {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, "Failed to create SCIM client")
	}

	// verify that the SCIM API can be accessed
	opts := &scim.QueryOptions{
		Filter:             scim.GroupFilterByDisplayName(org.Spec.MappedOrgAdminIDPGroup),
		ExcludedAttributes: scim.SetAttributes(scim.AttrMembers),
	}

	groups, err := scimClient.GetGroups(ctx, opts)
	if err != nil {
		logger.Error(err, "Failed to request data from SCIM API")
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, "Failed to request data from SCIM API")
	}
	if len(groups) == 0 {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, org.Spec.MappedOrgAdminIDPGroup+" Group not found in SCIM API")
	}

	return greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, lifecycle.CreatedReason, "SCIM API is available")
}

func calculateReadyCondition(scimAPIAvailableCondition greenhousesapv1alpha1.Condition) greenhousesapv1alpha1.Condition {
	if scimAPIAvailableCondition.IsFalse() {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.ReadyCondition, greenhousesapv1alpha1.SCIMAPIUnavailableReason, "")
	}
	// If SCIM API availability is unknown, then Ready state should be True, because SCIM Config is optional.
	return greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.ReadyCondition, "", "")
}

func initOrganizationStatus(org *greenhousesapv1alpha1.Organization) {
	orgStatus := org.Status
	for _, t := range exposedConditions {
		if orgStatus.GetConditionByType(t) == nil {
			orgStatus.SetConditions(greenhousesapv1alpha1.UnknownCondition(t, "", ""))
		}
	}
}

func (r *OrganizationReconciler) setStatus() lifecycle.Conditioner {
	return func(ctx context.Context, object lifecycle.RuntimeObject) {
		org, ok := object.(*greenhousesapv1alpha1.Organization)
		if !ok {
			return
		}
		scimAPIAvailableCondition := r.checkSCIMAPIAvailability(ctx, org)
		readyCondition := calculateReadyCondition(scimAPIAvailableCondition)
		org.Status.SetConditions(scimAPIAvailableCondition, readyCondition)
	}
}
