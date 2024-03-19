// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

type TeamMembershipUpdater struct {
	k8sClient   client.Client
	scimClient  scim.ScimClient
	namespace   string
	teamsByName map[string]greenhousesapv1alpha1.Team
}

func NewTeamMembershipUpdater(k8sClient client.Client, scimClient scim.ScimClient, namespace string) TeamMembershipUpdater {
	if namespace == "" {
		log.Print("no namespace provided, setting TeamMembershipUpdater to 'default' namespace")
		namespace = "default"
	}
	teamUpdater := TeamMembershipUpdater{k8sClient, scimClient, namespace, map[string]greenhousesapv1alpha1.Team{}}
	return teamUpdater
}

// Lists all available teams und updates or creates respective teamMemberships if team.Spec.MappedIDPGroup is present
func (t *TeamMembershipUpdater) DoUpdates() error {
	teamList := greenhousesapv1alpha1.TeamList{}

	err := t.k8sClient.List(context.TODO(), &teamList, &client.ListOptions{Namespace: t.namespace})
	if err != nil {
		return err
	}

	// create org admin team from org spec
	orgAdminTeam, err := t.getOrgAdminTeam()
	if err != nil {
		log.Printf(`[Info] failed creating org-admin team for organization %s: %s`, t.namespace, err)
	} else {
		teamList.Items = append(teamList.Items, orgAdminTeam)
	}

	var wg sync.WaitGroup
	wg.Add(len(teamList.Items))

	for _, team := range teamList.Items {
		t.teamsByName[team.Name] = team

		go func(team greenhousesapv1alpha1.Team) {
			defer wg.Done()
			err = t.processTeamMembership(team)
			if err != nil {
				log.Printf(`[Info] failed processing team-membership for team: %s`, err)
			}
		}(team)
	}
	wg.Wait()

	return t.deleteOrphanedTeamMemberships()
}

func (t *TeamMembershipUpdater) getOrgAdminTeam() (greenhousesapv1alpha1.Team, error) {
	org := new(greenhousesapv1alpha1.Organization)
	err := t.k8sClient.Get(context.TODO(), types.NamespacedName{Name: t.namespace}, org)
	if err != nil {
		return greenhousesapv1alpha1.Team{}, err
	}

	orgAdminTeam := greenhousesapv1alpha1.Team{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: org.Name,
			Name:      org.Name + "-admin",
		},
		Spec: greenhousesapv1alpha1.TeamSpec{
			MappedIDPGroup: org.Spec.MappedOrgAdminIDPGroup,
		},
	}

	return orgAdminTeam, nil
}

func (t *TeamMembershipUpdater) processTeamMembership(team greenhousesapv1alpha1.Team) error {
	teamMembership := new(greenhousesapv1alpha1.TeamMembership)

	err := t.k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: team.Namespace, Name: team.Name}, teamMembership)
	if !apierrors.IsNotFound(err) && err != nil {
		return err
	}

	teamMembership.Status.LastSyncedTime = &meta_v1.Time{Time: time.Now()}

	// Delete existing TM for team without mapped idp group
	if team.Spec.MappedIDPGroup == "" {
		if err == nil {
			return t.k8sClient.Delete(context.TODO(), teamMembership, &client.DeleteOptions{})
		}
		return nil
	}

	// create or update TM
	if apierrors.IsNotFound(err) {
		return t.createTeamMembership(team)
	}
	return t.updateTeamMembership(teamMembership, team)
}

func (t *TeamMembershipUpdater) createTeamMembership(team greenhousesapv1alpha1.Team) error {
	members, err := t.scimClient.GetTeamMembers(team.Spec.MappedIDPGroup)
	if err != nil {
		return err
	}
	users := t.scimClient.GetUsers(members)
	now := meta_v1.NewTime(time.Now())

	teamMembership := new(greenhousesapv1alpha1.TeamMembership)
	teamMembership.Namespace = team.Namespace
	teamMembership.Name = team.Name
	teamMembership.Spec.Members = users
	teamMembership.Status.LastChangedTime = &now
	err = t.k8sClient.Create(context.TODO(), teamMembership, &client.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("created team-membership: %s", team.Name)
	team.ObjectMeta.OwnerReferences = []meta_v1.OwnerReference{{APIVersion: greenhousesapv1alpha1.GroupVersion.String(), Kind: "TeamMembership", Name: teamMembership.GetName(), UID: teamMembership.GetUID(), Controller: nil, BlockOwnerDeletion: nil}}
	return t.k8sClient.Update(context.TODO(), &team, &client.UpdateOptions{})
}

func (t *TeamMembershipUpdater) updateTeamMembership(teamMembership *greenhousesapv1alpha1.TeamMembership, team greenhousesapv1alpha1.Team) error {
	members, err := t.scimClient.GetTeamMembers(team.Spec.MappedIDPGroup)
	if err != nil {
		return err
	}
	users := t.scimClient.GetUsers(members)
	now := meta_v1.NewTime(time.Now())
	teamMembership.Spec.Members = users
	teamMembership.Status.LastChangedTime = &now
	err = t.k8sClient.Update(context.TODO(), teamMembership, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	log.Printf("updated team-membership: %s", team.Name)
	team.ObjectMeta.OwnerReferences = []meta_v1.OwnerReference{{APIVersion: greenhousesapv1alpha1.GroupVersion.String(), Kind: "TeamMembership", Name: teamMembership.GetName(), UID: teamMembership.GetUID(), Controller: nil, BlockOwnerDeletion: nil}}
	return t.k8sClient.Update(context.TODO(), &team, &client.UpdateOptions{})
}

func (t *TeamMembershipUpdater) deleteOrphanedTeamMemberships() error {
	teamMembershipList := greenhousesapv1alpha1.TeamMembershipList{}
	err := t.k8sClient.List(context.TODO(), &teamMembershipList, &client.ListOptions{Namespace: t.namespace})
	if err != nil {
		return err
	}

	for _, teamMembership := range teamMembershipList.Items {
		_, ok := t.teamsByName[teamMembership.Name]
		if !ok {
			teamMembership := teamMembership
			err = t.k8sClient.Delete(context.TODO(), &teamMembership, &client.DeleteOptions{})
			if err != nil {
				log.Printf(`[Info] failed deleting orphaned teamMembership: %s`, err)
			}
			log.Printf("deleted team-membership: %s", teamMembership.Name)
		}
	}
	return nil
}
