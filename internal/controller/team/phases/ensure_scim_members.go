// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/scim"
	"github.com/cloudoperators/greenhouse/internal/util"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureSCIMMembers(team *greenhousev1alpha1.Team) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		var organization = new(greenhousev1alpha1.Organization)
		if err := p.Get(ctx, types.NamespacedName{Name: team.Namespace}, organization); err != nil {
			return lifecycle.Break(), client.IgnoreNotFound(err)
		}

		if team.Spec.MappedIDPGroup == "" {
			log.FromContext(ctx).Info("Team does not have MappedIdpGroup set", "team", team.Name)
			return lifecycle.Break(), nil
		}

		orgSCIMAPIAvailableCondition := organization.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
		if orgSCIMAPIAvailableCondition == nil || !orgSCIMAPIAvailableCondition.IsTrue() {
			team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
				greenhousev1alpha1.SCIMAPIUnavailableReason, "SCIM API in Organization is unavailable"))
			return lifecycle.Break(), nil
		}

		if organization.Spec.Authentication == nil || organization.Spec.Authentication.SCIMConfig == nil {
			log.FromContext(ctx).Info("SCIM config is missing from org", "Name", organization.Name, "Namespace", organization.Namespace)
			team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
				greenhousev1alpha1.SecretNotFoundReason, "SCIM config is missing from organization"))
			return lifecycle.Break(), nil
		}

		scimClient, err := createSCIMClient(ctx, p.Client, team.Namespace, organization.Spec.Authentication.SCIMConfig)
		if err != nil {
			team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
				greenhousev1alpha1.SCIMConfigErrorReason, err.Error()))
			return lifecycle.Break(), err
		}

		users, membersValidCondition, err := getUsersFromSCIM(ctx, scimClient, team.Spec.MappedIDPGroup)
		if err != nil {
			log.FromContext(ctx).Info("failed getting users from SCIM", "error", err)
			team.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.SCIMAccessReadyCondition,
				greenhousev1alpha1.SCIMRequestFailedReason, ""))
			return lifecycle.Break(), err
		}
		team.SetCondition(membersValidCondition)
		team.Status.Members = users
		team.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAccessReadyCondition, "", ""))

		return lifecycle.Continue(), nil
	}
}

func createSCIMClient(ctx context.Context, c client.Client, namespace string, scimConfig *greenhousev1alpha1.SCIMConfig) (scim.ISCIMClient, error) {
	clientConfig, err := util.GreenhouseSCIMConfigToSCIMConfig(ctx, c, scimConfig, namespace)
	if err != nil {
		return nil, err
	}
	logger := ctrl.LoggerFrom(ctx)
	return scim.NewSCIMClient(logger, clientConfig)
}

func getUsersFromSCIM(ctx context.Context, scimClient scim.ISCIMClient, mappedIDPGroup string) ([]greenhousev1alpha1.User, greenhousemetav1alpha1.Condition, error) {
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
