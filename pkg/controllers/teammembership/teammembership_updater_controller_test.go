// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package teammembership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var _ = Describe("TeammembershipUpdaterController", func() {
	When("reconciling a teammembership", func() {
		BeforeEach(func() {
			By("creating new test setup")
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
			createTestOrgWithSecret(setup.Namespace())
		})

		It("should update existing TM without users", func() {
			By("creating a test TeamMembership")
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("creating a test Team")
			err = setup.Create(test.Ctx, &greenhousev1alpha1.Team{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.TeamSpec{
					MappedIDPGroup: validIdpGroupName,
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")

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

		It("should update existing TM with users", func() {
			By("creating a test TeamMembership with 1 existing user")
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
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
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("creating a test Team")
			err = setup.Create(test.Ctx, &greenhousev1alpha1.Team{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.TeamSpec{
					MappedIDPGroup: validIdpGroupName,
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")

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
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
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
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("creating first test Team")
			err = setup.Create(test.Ctx, &greenhousev1alpha1.Team{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.TeamSpec{
					MappedIDPGroup: validIdpGroupName,
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")

			By("creating second test Team")
			setup.CreateTeam(test.Ctx, secondTeamName, test.WithMappedIDPGroup(otherValidIdpGroupName))

			By("ensuring two TeamMemberships have been created")
			teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
			Eventually(func(g Gomega) {
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(teamMemberships.Items).To(HaveLen(2), "there should be exactly two TeamMemberships")
			}).Should(Succeed(), "two TeamMemberships should have been created")

			By("ensuring both TeamMemberships have been reconciled")
			Eventually(func(g Gomega) {
				g.Expect(teamMemberships.Items[0].Spec.Members).To(HaveLen(2), "first Team should have 2 users")
				g.Expect(teamMemberships.Items[1].Spec.Members).To(HaveLen(3), "second Team should have 3 users")
				teams := &greenhousev1alpha1.TeamList{}
				err := setup.List(test.Ctx, teams, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Teams")
				g.Expect(teams.Items[0].GetOwnerReferences()).ToNot(BeNil(), "first Team should have set owner reference")
				g.Expect(teams.Items[1].GetOwnerReferences()).ToNot(BeNil(), "first Team should have set owner reference")
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
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("creating a test Team without mappedIdpGroup")
			err = setup.Create(test.Ctx, &greenhousev1alpha1.Team{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      firstTeamName,
					Namespace: setup.Namespace(),
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")

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
	})
})
