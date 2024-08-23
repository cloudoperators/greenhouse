// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/scim"
)

const (
	TeamMembershipUpdateInterval = 1 * time.Second
	TeamMembershipUpdateTimeout  = 30 * time.Second
)

type TeamMebershipUpdaterTests struct {
	k8sClient             client.Client
	scimClient            *scim.ScimClient
	teamMembershipUpdater TeamMembershipUpdater
}

func TestTeamUpdaterSuite(t *testing.T) {
	kubeBuilderAssets := os.Getenv("KUBEBUILDER_ASSETS")
	assert.Equal(t, kubeBuilderAssets != "", true, "KUBEBUILDER_ASSET env needs to be set for envtest to run")
	absPathConfigBasePath, err := clientutil.FindDirUpwards(".", "charts", 10)
	assert.NilError(t, err)
	crdPaths := []string{filepath.Join(absPathConfigBasePath, "manager", "crds")}

	envTest := &envtest.Environment{
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := envTest.Start()
	assert.NilError(t, err)

	defer func() {
		err := envTest.Stop()
		if err != nil {
			log.Printf("error stopping envTest: %s", err)
		}
	}()

	k8sClient, err := NewK8sClient(cfg)
	assert.NilError(t, err)

	groupsServer := scim.ReturnDefaultGroupResponseMockServer()
	defer groupsServer.Close()

	scimConfig := scim.Config{RawURL: groupsServer.URL, AuthType: scim.Basic, BasicAuthConfig: &scim.BasicAuthConfig{BasicAuthUser: "user", BasicAuthPw: "pw"}}
	scimClient, err := scim.NewScimClient(scimConfig)
	assert.NilError(t, err)

	teamMembershipUpdater := NewTeamMembershipUpdater(k8sClient, *scimClient, "default")

	tmTests := TeamMebershipUpdaterTests{k8sClient: k8sClient, scimClient: scimClient, teamMembershipUpdater: teamMembershipUpdater}

	t.Run("UpdateExistingTMWithoutUsers", tmTests.TestUpdateExistingTMWithoutUsers)
	t.Run("UpdateExistingTMWithUser", tmTests.TestUpdateExistingTMWithUser)
	t.Run("UpdateMultipleTMs", tmTests.TestUpdateMultipleTMs)
	t.Run("DoNothingIfTeamHasNoMappedIDPGroup", tmTests.TestDoNothingIfTeamHasNoMappedIDPGroup)
	t.Run("DeleteExistingTMIfTeamHasNoMappedIDPGroup", tmTests.TestDeleteExistingTMIfTeamHasNoMappedIDPGroup)
	t.Run("DeleteOrphanedTMs", tmTests.TestDeleteOrphanedTMs)
	t.Run("LogErrorOnUpdateNonExistingGroupName", tmTests.TestLogErrorOnUpdateNonExistingGroupName)
	t.Run("LogErrorOnUpStreamError", tmTests.TestLogErrorOnUpStreamError)
	t.Run("CreateTMForOrgAdmins", tmTests.TestCreateTMForOrgAdmins)
}

func (tmTests *TeamMebershipUpdaterTests) TestUpdateExistingTMWithoutUsers(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "SOME_IDP_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(ctx, &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 1 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 1 TeamMemberships %v", teamMemberships))
	assert.Equal(t, len(teamMemberships.Items[0].Spec.Members), 2, fmt.Sprintf("Should have exactly 2 users %v", teamMemberships.Items[0].Spec.Members))
}

func (tmTests *TeamMebershipUpdaterTests) TestUpdateExistingTMWithUser(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team",
			MappedIDPGroup: "SOME_IDP_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamMembershipSpec{
			Members: []greenhousev1alpha1.User{
				{
					ID:        "I12345",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
				},
			},
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true, func(ctx context.Context) (done bool, err error) {
		err = tmTests.k8sClient.List(ctx, &teamMemberships, &client.ListOptions{})
		if err != nil {
			return false, err
		}
		if len(teamMemberships.Items) != 1 {
			return false, nil
		}
		return true, nil
	})
	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 1 TeamMemberships %v", teamMemberships))
	assert.Equal(t, len(teamMemberships.Items[0].Spec.Members), 2, fmt.Sprintf("Should have exactly 2 users %v", teamMemberships.Items[0].Spec.Members))
}

func (tmTests *TeamMebershipUpdaterTests) TestUpdateMultipleTMs(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	team1 := &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team",
			MappedIDPGroup: "SOME_IDP_GROUP_NAME",
		},
	}
	err = tmTests.k8sClient.Create(context.TODO(), team1, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team 2",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-2",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 2",
			MappedIDPGroup: "SOME_OTHER_IDP_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamMembershipSpec{
			Members: []greenhousev1alpha1.User{
				{
					ID:        "I12345",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
				},
			},
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(ctx, &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 2 {
				return false, nil
			}
			return true, nil
		})
	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 1 TeamMemberships %v", teamMemberships))
	assert.Equal(t, len(teamMemberships.Items[0].Spec.Members), 2, fmt.Sprintf("First Team should have exactly 2 users %v", teamMemberships.Items[0].Spec.Members))
	assert.Equal(t, len(teamMemberships.Items[1].Spec.Members), 3, fmt.Sprintf("Second Team should have exactly 3 users %v", teamMemberships.Items[0].Spec.Members))

	teams := greenhousev1alpha1.TeamList{}
	err = tmTests.k8sClient.List(context.TODO(), &teams, &client.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, teams.Items[0].GetOwnerReferences() != nil, true, "Should set owner reference on team")
	assert.Equal(t, teams.Items[1].GetOwnerReferences() != nil, true, "Should set owner reference on team")
}

func (tmTests *TeamMebershipUpdaterTests) TestDoNothingIfTeamHasNoMappedIDPGroup(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description: "Test Team 1",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 0 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 0 TeamMemberships %v", teamMemberships))
}

func (tmTests *TeamMebershipUpdaterTests) TestDeleteExistingTMIfTeamHasNoMappedIDPGroup(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description: "Test Team 1",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(ctx, &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 0 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 0 TeamMemberships %v", teamMemberships))
}

func (tmTests *TeamMebershipUpdaterTests) TestDeleteOrphanedTMs(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "SOME_IDP_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-2",
			Namespace: "default",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	// Need fresh Updater for this test
	teamMembershipUpdater := NewTeamMembershipUpdater(tmTests.k8sClient, *tmTests.scimClient, "default")
	err = teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 1 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 1 TeamMemberships %v", teamMemberships))
	assert.Equal(t, teamMemberships.Items[0].ObjectMeta.Name, "test-team-1", "Only TM for test-team-1 should remain")
}

func (tmTests *TeamMebershipUpdaterTests) TestLogErrorOnUpdateNonExistingGroupName(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "NON_EXISTING_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err, "Should only log non existing team")
	assert.Equal(t, strings.Contains(buf.String(), "[Info] failed processing team-membership for team: no mapped group found for NON_EXISTING_GROUP_NAME"), true)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 0 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 0 TeamMemberships %v", teamMemberships))
}

func (tmTests *TeamMebershipUpdaterTests) TestLogErrorOnUpStreamError(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Team",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-team-1",
			Namespace: "default",
		},
		Spec: greenhousev1alpha1.TeamSpec{
			Description:    "Test Team 1",
			MappedIDPGroup: "GROUP_NAME_ERROR_404",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err, "Should only log upstream error")
	assert.Equal(t, strings.Contains(buf.String(), "[Info] failed processing team-membership for team: could not retrieve TeamMembers from"), true)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 0 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 0 TeamMemberships %v", teamMemberships))
}

func (tmTests *TeamMebershipUpdaterTests) TestCreateTMForOrgAdmins(t *testing.T) {
	err := tmTests.cleanUpTests()
	assert.NilError(t, err)

	err = tmTests.k8sClient.Create(context.TODO(), &greenhousev1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "Organization",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: greenhousev1alpha1.OrganizationSpec{
			Description:            "Test Org",
			MappedOrgAdminIDPGroup: "SOME_IDP_GROUP_NAME",
		},
	}, &client.CreateOptions{})
	assert.NilError(t, err)

	err = tmTests.teamMembershipUpdater.DoUpdates()
	assert.NilError(t, err)

	teamMemberships := greenhousev1alpha1.TeamMembershipList{}
	pollErr := wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(ctx, &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 1 {
				return false, nil
			}
			return true, nil
		})

	assert.NilError(t, pollErr, fmt.Sprintf("Should list exactly 1 TeamMemberships %v", teamMemberships))
	assert.Equal(t, len(teamMemberships.Items[0].Spec.Members), 2, fmt.Sprintf("Should have exactly 2 users %v", teamMemberships.Items[0].Spec.Members))
}

func (tmTests *TeamMebershipUpdaterTests) cleanUpTests() error {
	teamMemberships := greenhousev1alpha1.TeamMembershipList{}

	err := tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
	if !apierrors.IsNotFound(err) && err != nil {
		return err
	}
	for _, teamMembership := range teamMemberships.Items {
		err = tmTests.k8sClient.Delete(context.TODO(), &teamMembership, &client.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	teams := greenhousev1alpha1.TeamList{}
	err = tmTests.k8sClient.List(context.TODO(), &teams, &client.ListOptions{})
	if !apierrors.IsNotFound(err) && err != nil {
		return err
	}
	for _, team := range teams.Items {
		err = tmTests.k8sClient.Delete(context.TODO(), &team, &client.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	err = wait.PollUntilContextTimeout(context.Background(), TeamMembershipUpdateInterval, TeamMembershipUpdateTimeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = tmTests.k8sClient.List(context.TODO(), &teamMemberships, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			err = tmTests.k8sClient.List(context.TODO(), &teams, &client.ListOptions{})
			if err != nil {
				return false, err
			}
			if len(teamMemberships.Items) != 0 && len(teams.Items) != 0 {
				return false, nil
			}
			return true, nil
		})
	if err != nil {
		return err
	}
	return nil
}
