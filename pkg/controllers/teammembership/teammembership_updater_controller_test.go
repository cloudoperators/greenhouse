// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	firstTeamName          = "test-team-1"
	secondTeamName         = "test-team-2"
	validIdpGroupName      = "SOME_IDP_GROUP_NAME"
	otherValidIdpGroupName = "SOME_OTHER_IDP_GROUP_NAME"
	nonExistingGroupName   = "NON_EXISTING_GROUP_NAME"
)

var (
	setup *test.TestSetup
)

var _ = Describe("TeammembershipUpdaterController", func() {
	When("reconciling a teammembership", func() {
		BeforeEach(func() {
			By("creating new test setup")
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
			createTestOrgWithSecret(setup.Namespace())
		})

		It("should create TeamMembership when Team is created", func() {
			By("creating a test Team")
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			ownerRef := metav1.OwnerReference{
				APIVersion:         greenhousev1alpha1.GroupVersion.String(),
				Kind:               "Team",
				UID:                team.UID,
				Name:               team.Name,
				Controller:         nil,
				BlockOwnerDeletion: nil,
			}

			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")
				teamMembership := teamMemberships.Items[0]
				g.Expect(teamMemberships.Items[0].OwnerReferences).To(ContainElement(ownerRef), "TeamMembership should have set team as owner reference")
				g.Expect(teamMembership.Spec.Members).To(HaveLen(2), "the TeamMembership should have exactly two Members")
				g.Expect(teamMembership.Status.LastChangedTime).ToNot(BeNil(), "TeamMembership status should have updated LastChangedTime")
				scimAccessReadyCondition := teamMembership.Status.GetConditionByType(greenhousev1alpha1.ScimAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.ScimAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue))
				readyCondition := teamMembership.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Type).To(Equal(greenhousev1alpha1.ReadyCondition))
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			}).Should(Succeed(), "TeamMembership should be reconciled")
		})

		It("should update existing TM without users", func() {
			By("creating a test TeamMembership")
			createTeamMembership(firstTeamName, nil)

			By("creating a test Team")
			createTeam(firstTeamName, validIdpGroupName)

			By("ensuring the TeamMembership has been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")
				firstTeamMembership := teamMemberships.Items[0]
				g.Expect(firstTeamMembership.Spec.Members).To(HaveLen(2), "the TeamMembership should have exactly two Members")
				g.Expect(firstTeamMembership.Status.LastChangedTime).ToNot(BeNil(), "TeamMembership status should have updated LastChangedTime")
				scimAccessReadyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ScimAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.ScimAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue))
				readyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Type).To(Equal(greenhousev1alpha1.ReadyCondition))
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			}).Should(Succeed(), "the TeamMembership should be reconciled")
		})

		It("should update existing TM with users", func() {
			By("creating a test TeamMembership with 1 existing user")
			createTeamMembership(firstTeamName, []greenhousev1alpha1.User{
				{
					ID:        "I12345",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
				},
			})

			By("creating a test Team")
			createTeam(firstTeamName, validIdpGroupName)

			By("ensuring the TeamMembership has been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")
				g.Expect(teamMemberships.Items[0].Spec.Members).To(HaveLen(2), "the TeamMembership should have exactly two Members")
				g.Expect(teamMemberships.Items[0].Status.LastChangedTime).ToNot(BeNil(), "TeamMembership status should have updated LastChangedTime")
			}).Should(Succeed(), "the TeamMembership should be reconciled")
		})

		It("should update multiple TMs", func() {
			By("creating a test TeamMembership with 1 existing user")
			createTeamMembership(firstTeamName, []greenhousev1alpha1.User{
				{
					ID:        "I12345",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
				},
			})

			By("creating first test Team")
			firstTeam := createTeam(firstTeamName, validIdpGroupName)

			By("creating second test Team")
			secondTeam := setup.CreateTeam(test.Ctx, secondTeamName, test.WithMappedIDPGroup(otherValidIdpGroupName))

			By("ensuring two TeamMemberships have been created")
			teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
			Eventually(func(g Gomega) {
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(2), "there should be exactly two TeamMemberships")
			}).Should(Succeed(), "two TeamMemberships should have been created")

			firstOwnerRef := metav1.OwnerReference{
				APIVersion:         greenhousev1alpha1.GroupVersion.String(),
				Kind:               "Team",
				UID:                firstTeam.UID,
				Name:               firstTeam.Name,
				Controller:         nil,
				BlockOwnerDeletion: nil,
			}
			secondOwnerRef := metav1.OwnerReference{
				APIVersion:         greenhousev1alpha1.GroupVersion.String(),
				Kind:               "Team",
				UID:                secondTeam.UID,
				Name:               secondTeam.Name,
				Controller:         nil,
				BlockOwnerDeletion: nil,
			}

			By("ensuring both TeamMemberships have been reconciled")
			Eventually(func(g Gomega) {
				g.Expect(teamMemberships.Items[0].Spec.Members).To(HaveLen(2), "first TeamMembership should have 2 users")
				g.Expect(teamMemberships.Items[1].Spec.Members).To(HaveLen(3), "second TeamMembership should have 3 users")
				g.Expect(teamMemberships.Items[0].OwnerReferences).To(ContainElement(firstOwnerRef), "first TeamMembership should have set first team as owner reference")
				g.Expect(teamMemberships.Items[1].OwnerReferences).To(ContainElement(secondOwnerRef), "second TeamMembership should have set second team as owner reference")
			}).Should(Succeed(), "both TeamMemberships should be reconciled")
		})

		It("should do nothing if Team has no mappedIdpGroup", func() {
			By("creating a test Team without mappedIdpGroup")
			setup.CreateTeam(test.Ctx, firstTeamName)

			By("creating a test Team with valid mappedIdpGroup")
			secondTeam := setup.CreateTeam(test.Ctx, secondTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			By("ensuring there is only one TeamMembership")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")
				g.Expect(teamMemberships.Items[0].Name).To(Equal(secondTeam.Name), "only second team should have created TeamMembership")
			}).Should(Succeed(), "there should be only one TeamMembership")
		})

		It("should delete existing TM if team has no mappedIDPGroup", func() {
			By("creating a test TeamMembership")
			createTeamMembership(firstTeamName, nil)

			By("creating a test Team without mappedIdpGroup")
			createTeam(firstTeamName, "")

			By("ensuring the TeamMembership has been deleted")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(BeEmpty(), "there should be exactly zero TeamMemberships")
			}).Should(Succeed(), "TeamMembership should have been deleted")
		})

		It("should log error on update non existing idp group", func() {
			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with non existing idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(nonExistingGroupName))

			By("ensuring logger was called correctly")
			failedProcessingLog := "failed processing team-membership for team"
			reasonLog := "no mapped group found for " + nonExistingGroupName
			Eventually(func(g Gomega) {
				g.Expect(tee.Contents()).To(ContainSubstring(failedProcessingLog), "logger should log failed processing")
				g.Expect(tee.Contents()).To(ContainSubstring(reasonLog), "logger should log reason")
			}).Should(Succeed(), "logger should be called correctly")

			By("ensuring TeamMemberships have not been created")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(BeEmpty(), "there should be exactly zero TeamMemberships")
			}).Should(Succeed(), "the TeamMemberships should not have been created")
		})

		It("should log error on upstream error", func() {
			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with faulty idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup("GROUP_NAME_ERROR_404"))

			By("ensuring logger was called correctly")
			failedProcessingLog := "failed processing team-membership for team"
			reasonLog := "could not retrieve TeamMembers from"
			Eventually(func(g Gomega) {
				g.Expect(tee.Contents()).To(ContainSubstring(failedProcessingLog), "logger should log failed processing")
				g.Expect(tee.Contents()).To(ContainSubstring(reasonLog), "logger should log reason")
			}).Should(Succeed(), "logger should be called correctly")

			By("ensuring TeamMemberships have not been created")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(BeEmpty(), "there should be exactly zero TeamMemberships")
			}).Should(Succeed(), "the TeamMemberships should not have been created")
		})

		It("should set ready condition to false on missing secret", func() {
			By("deleting the secret")
			var secret corev1.Secret
			err := setup.Get(test.Ctx, types.NamespacedName{Name: "test-secret", Namespace: setup.Namespace()}, &secret)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting the secret")
			err = setup.Delete(test.Ctx, &secret)
			Expect(err).ToNot(HaveOccurred(), "there must be no error deleting the secret")

			By("creating a test TeamMembership")
			createTeamMembership(firstTeamName, nil)

			By("creating a test Team with valid MappedIdpGroup")
			createTeam(firstTeamName, validIdpGroupName)

			By("ensuring TeamMemberships have been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")

				firstTeamMembership := teamMemberships.Items[0]

				scimAccessReadyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ScimAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.ScimAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(scimAccessReadyCondition.Reason).To(Equal(greenhousev1alpha1.SecretNotFoundReason), "reason should be set to SecretNotFound")
				readyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Type).To(Equal(greenhousev1alpha1.ReadyCondition))
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			}).Should(Succeed(), "TeamMemberships should have been reconciled")
		})

		It("should set ready condition to false on SCIM request failed", func() {
			By("creating a test TeamMembership")
			createTeamMembership(firstTeamName, nil)

			By("creating a test Team with invalid MappedIdpGroup")
			createTeam(firstTeamName, nonExistingGroupName)

			By("ensuring TeamMemberships have been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")

				firstTeamMembership := teamMemberships.Items[0]

				scimAccessReadyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ScimAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.ScimAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(scimAccessReadyCondition.Reason).To(Equal(greenhousev1alpha1.ScimRequestFailedReason), "reason should be set to ScimRequestFailed")
				readyCondition := firstTeamMembership.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Type).To(Equal(greenhousev1alpha1.ReadyCondition))
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			}).Should(Succeed(), "TeamMemberships should have been reconciled")
		})

		It("should log about missing SCIM config", func() {
			var organization = new(greenhousev1alpha1.Organization)
			err := setup.Get(test.Ctx, types.NamespacedName{Name: setup.Namespace()}, organization)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting organization")

			By("removing SCIM config from organization")
			organization.Spec.Authentication.SCIMConfig = nil
			Expect(setup.Update(test.Ctx, organization)).To(Succeed(), "there must be no error removing SCIM config from org")

			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with valid idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(BeEmpty(), "there should be no TeamMemberships")
				g.Expect(tee.Contents()).To(ContainSubstring("SCIM config is missing from org"), "logger should log about missing SCIM config")
			}).Should(Succeed(), "TeamMemberships should have been reconciled")
		})

		It("should update TeamMembership when one user has changed", func() {
			By("creating test TeamMembership with two users")
			originalUsers := []greenhousev1alpha1.User{
				{ // User from mock.
					ID:        "I12345",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
				},
				{ // User different from mock.
					ID:        "I99999",
					FirstName: "Some",
					LastName:  "User",
					Email:     "some.user@example.com",
				},
			}
			createTeamMembership(firstTeamName, originalUsers)

			By("creating test Team with valid idp group")
			createTeam(firstTeamName, validIdpGroupName)

			expectedUser1 := greenhousev1alpha1.User{
				ID:        "I12345",
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john.doe@example.com",
			}
			expectedUser2 := greenhousev1alpha1.User{
				ID:        "I23456",
				FirstName: "Jane",
				LastName:  "Doe",
				Email:     "jane.doe@example.com",
			}

			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(1), "there should be exactly one TeamMembership")
				teamMembership := teamMemberships.Items[0]
				g.Expect(teamMembership.Spec.Members).To(HaveLen(2), "TeamMembership should have two users")
				g.Expect(teamMembership.Spec.Members).ToNot(Equal(originalUsers), "TeamMembership users should be updated")
				g.Expect(teamMembership.Spec.Members).To(ContainElement(expectedUser1), "TeamMembership users should contain first expected user")
				g.Expect(teamMembership.Spec.Members).To(ContainElement(expectedUser2), "TeamMembership users should contain second expected user")
			}).Should(Succeed(), "TeamMembership should have been reconciled")
		})
	})
})

func createTestOrgWithSecret(namespace string) {
	By("creating a secret")
	testSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"basicAuthUser": []byte("user"),
			"basicAuthPw":   []byte("pw"),
		},
	}
	err := setup.Client.Create(test.Ctx, &testSecret)
	Expect(err).ToNot(HaveOccurred(), "there must be no error creating a secret")

	By("creating organization with name: " + namespace)
	setup.CreateOrganization(test.Ctx, namespace, func(o *greenhousev1alpha1.Organization) {
		o.Spec.Authentication = &greenhousev1alpha1.Authentication{
			SCIMConfig: &greenhousev1alpha1.SCIMConfig{
				BaseURL: groupsServer.URL,
				BasicAuthUser: &greenhousev1alpha1.ValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "test-secret",
						Key:  "basicAuthUser",
					},
				},
				BasicAuthPw: &greenhousev1alpha1.ValueFromSource{
					Secret: &greenhousev1alpha1.SecretKeyReference{
						Name: "test-secret",
						Key:  "basicAuthPw",
					},
				},
			},
		}
	})
}

func createTeamMembership(name string, members []greenhousev1alpha1.User) {
	err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
		TypeMeta: metav1.TypeMeta{
			APIVersion: greenhousev1alpha1.GroupVersion.Group,
			Kind:       "TeamMembership",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: setup.Namespace(),
		},
		Spec: greenhousev1alpha1.TeamMembershipSpec{
			Members: members,
		},
	})
	Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")
}

func createTeam(name string, mappedIDPGroup string) *greenhousev1alpha1.Team {
	team := &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: setup.Namespace(),
		},
		Spec: greenhousev1alpha1.TeamSpec{
			MappedIDPGroup: mappedIDPGroup,
		},
	}
	err := setup.Create(test.Ctx, team)
	Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")
	return team
}
