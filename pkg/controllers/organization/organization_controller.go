// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

var (
	// exposedConditions are the conditions that are exposed in the StatusConditions of the Organization.
	exposedConditions = []greenhousesapv1alpha1.ConditionType{
		greenhousesapv1alpha1.ReadyCondition,
		greenhousesapv1alpha1.SCIMAPIAvailableCondition,
	}
)

// OrganizationReconciler reconciles an Organization object
type OrganizationReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/finalizers,verbs=update
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teams,verbs=get;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&corev1.Namespace{}).
		Owns(&greenhousesapv1alpha1.Team{}).
		Complete(r)
}

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return lifecycle.Reconcile(ctx, r.Client, req.NamespacedName, &greenhousesapv1alpha1.Organization{}, r, nil)
}

func (r *OrganizationReconciler) EnsureDeleted(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	return ctrl.Result{}, lifecycle.Success, nil // nothing to do in that case
}

func (r *OrganizationReconciler) EnsureCreated(ctx context.Context, object lifecycle.RuntimeObject) (ctrl.Result, lifecycle.ReconcileResult, error) {
	org, ok := object.(*greenhousesapv1alpha1.Organization)
	if !ok {
		return ctrl.Result{}, lifecycle.Failed, errors.Errorf("RuntimeObject has incompatible type.")
	}

	orgStatus := initOrganizationStatus(org)
	defer func() {
		if statusErr := r.setStatus(ctx, org, orgStatus); statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "failed to set status")
		}
	}()

	if err := r.reconcileNamespace(ctx, org); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

	scimAPIAvailableCondition := r.checkSCIMAPIAvailability(ctx, org)
	readyCondition := calculateReadyCondition(scimAPIAvailableCondition)
	orgStatus.SetConditions(scimAPIAvailableCondition, readyCondition)

	if err := r.reconcileAdminTeam(ctx, org); err != nil {
		return ctrl.Result{}, lifecycle.Failed, err
	}

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

func (r *OrganizationReconciler) checkSCIMAPIAvailability(ctx context.Context, org *greenhousesapv1alpha1.Organization) greenhousesapv1alpha1.Condition {
	if org.Spec.Authentication == nil || org.Spec.Authentication.SCIMConfig == nil {
		// SCIM Config is optional.
		return greenhousesapv1alpha1.UnknownCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, "", "SCIM Config not provided")
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
	clientConfig := scim.Config{
		RawURL:   scimConfig.BaseURL,
		AuthType: scim.Basic,
		BasicAuthConfig: &scim.BasicAuthConfig{
			BasicAuthUser: basicAuthUser,
			BasicAuthPw:   basicAuthPw,
		},
	}
	scimClient, err := scim.NewScimClient(clientConfig)
	if err != nil {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, "Failed to create SCIM client")
	}

	_, err = scimClient.GetTeamMembers(org.Spec.MappedOrgAdminIDPGroup)
	if err != nil {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, greenhousesapv1alpha1.SCIMRequestFailedReason, "Failed to request data from SCIM API")
	}

	return greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.SCIMAPIAvailableCondition, "", "")
}

func calculateReadyCondition(scimAPIAvailableCondition greenhousesapv1alpha1.Condition) greenhousesapv1alpha1.Condition {
	if scimAPIAvailableCondition.IsFalse() {
		return greenhousesapv1alpha1.FalseCondition(greenhousesapv1alpha1.ReadyCondition, greenhousesapv1alpha1.SCIMAPIUnavailableReason, "")
	}
	// If SCIM API availability is unknown, then Ready state should be True, because SCIM Config is optional.
	return greenhousesapv1alpha1.TrueCondition(greenhousesapv1alpha1.ReadyCondition, "", "")
}

func initOrganizationStatus(org *greenhousesapv1alpha1.Organization) greenhousesapv1alpha1.OrganizationStatus {
	orgStatus := org.Status.DeepCopy()
	for _, t := range exposedConditions {
		if orgStatus.GetConditionByType(t) == nil {
			orgStatus.SetConditions(greenhousesapv1alpha1.UnknownCondition(t, "", ""))
		}
	}
	return *orgStatus
}

func (r *OrganizationReconciler) setStatus(ctx context.Context, org *greenhousesapv1alpha1.Organization, orgStatus greenhousesapv1alpha1.OrganizationStatus) error {
	_, err := clientutil.PatchStatus(ctx, r.Client, org, func() error {
		org.Status = orgStatus
		return nil
	})
	return err
}
