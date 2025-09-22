// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
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

var _ = Describe("TeamController", Ordered, func() {
	Context("reconciling with valid SCIM config in Organization", func() {
		BeforeEach(func() {
			By("creating new test setup")
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
			createTestOrgWithSecret(setup.Namespace())
		})

		It("should update Members when Team is created", func() {
			By("creating a test Team")
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				g.Expect(team.Status.Members).To(HaveLen(2), "the Team should have exactly two Members")
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.SCIMAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(team.Status.StatusConditions.IsReadyTrue()).To(BeTrue())
			}).Should(Succeed(), "Team should have the team members")
		})

		It("should update Members when multiple Teams are created", func() {
			By("creating first test Team")
			firstTeam := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			By("creating second test Team")
			secondTeam := setup.CreateTeam(test.Ctx, secondTeamName, test.WithMappedIDPGroup(otherValidIdpGroupName))

			By("ensuring that the first Team has been updated")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(firstTeam), firstTeam)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				g.Expect(firstTeam.Status.Members).To(HaveLen(2), "the first team should have exactly two Members")
				scimAccessReadyCondition := firstTeam.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.SCIMAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(firstTeam.Status.StatusConditions.IsReadyTrue()).To(BeTrue())
			}).Should(Succeed(), "First team should have the team members")

			By("ensuring that the second Team has been updated")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(secondTeam), secondTeam)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				g.Expect(secondTeam.Status.Members).To(HaveLen(3), "the second team should have exactly three Members")
				scimAccessReadyCondition := secondTeam.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil())
				g.Expect(scimAccessReadyCondition.Type).To(Equal(greenhousev1alpha1.SCIMAccessReadyCondition))
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(secondTeam.Status.StatusConditions.IsReadyTrue()).To(BeTrue())
			}).Should(Succeed(), "Second team should have the team members")
		})

		It("should log error when adding Team without idp group", func() {
			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with non existing idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(nonExistingGroupName))

			By("ensuring logger was called correctly")
			failedGettingUsersLog := "failed getting users from SCIM"
			reasonLog := "unexpected status code 404"
			Eventually(func(g Gomega) {
				g.Expect(tee.Contents()).To(ContainSubstring(failedGettingUsersLog), "logger should log failed getting users")
				g.Expect(tee.Contents()).To(ContainSubstring(reasonLog), "logger should log reason")
			}).Should(Succeed(), "logger should be called correctly")
		})

		It("should log error when adding Team with non existing idp group", func() {
			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with non existing idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(nonExistingGroupName))

			By("ensuring logger was called correctly")
			failedGettingUsersLog := "failed getting users from SCIM"
			reasonLog := "unexpected status code 404"
			Eventually(func(g Gomega) {
				g.Expect(tee.Contents()).To(ContainSubstring(failedGettingUsersLog), "logger should log failed getting users")
				g.Expect(tee.Contents()).To(ContainSubstring(reasonLog), "logger should log reason")
			}).Should(Succeed(), "logger should be called correctly")
		})

		It("should log error on upstream error", func() {
			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating a Team with faulty idp group")
			setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup("GROUP_NAME_ERROR_404"))

			By("ensuring logger was called correctly")
			failedGettingUsersLog := "failed getting users from SCIM"
			reasonLog := "unexpected status code 404"
			Eventually(func(g Gomega) {
				g.Expect(tee.Contents()).To(ContainSubstring(failedGettingUsersLog), "logger should log failed getting users")
				g.Expect(tee.Contents()).To(ContainSubstring(reasonLog), "logger should log reason")
			}).Should(Succeed(), "logger should be called correctly")
		})

		It("should set ready condition to false on missing secret", func() {
			By("deleting the secret")
			secret := &corev1.Secret{}
			err := setup.Get(test.Ctx, types.NamespacedName{Name: "test-secret", Namespace: setup.Namespace()}, secret)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting the secret")
			err = setup.Delete(test.Ctx, secret)
			Expect(err).ToNot(HaveOccurred(), "there must be no error deleting the secret")

			By("creating a test Team with valid MappedIdpGroup")
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			By("ensuring Team has been reconciled")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")

				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "condition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse), "SCIMAccessReadyCondition should be set to false")
				g.Expect(scimAccessReadyCondition.Reason).To(Equal(greenhousev1alpha1.SCIMConfigErrorReason), "reason should be set to SCIMConfigErrorReason")
				g.Expect(scimAccessReadyCondition.Message).To(Equal("secret for '.SCIMConfig.BasicAuthUser' is missing: Secret \"test-secret\" not found"))
				readyCondition := team.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil())
				g.Expect(readyCondition.Type).To(Equal(greenhousemetav1alpha1.ReadyCondition))
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(readyCondition.Message).To(Equal("SCIM access not ready"))
			}).Should(Succeed(), "Team should have been reconciled")
		})

		It("should set ready condition to false on SCIM request failed", func() {
			By("creating a test Team with invalid MappedIdpGroup")
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(nonExistingGroupName))

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "condition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse), "SCIMAccessReadyCondition should be set to false")
				g.Expect(team.Status.StatusConditions.IsReadyTrue()).To(BeFalse(), "Ready condition should be set to false")
			}).Should(Succeed())
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
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "condition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse), "SCIMAccessReadyCondition should be set to false")
				g.Expect(scimAccessReadyCondition.Message).To(Equal("SCIM config is missing from organization"))
				readyCondition := team.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "Ready condition should not be nil")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse), "Ready condition should be set to false")
				g.Expect(tee.Contents()).To(ContainSubstring("SCIM config is missing from org"), "logger should log about missing SCIM config")
			}).Should(Succeed())
		})

		It("should delete the team after all", func() {
			team := &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Namespace: setup.Namespace(), Name: firstTeamName}, team)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "Team should be deleted")
			}).Should(Succeed())

			team = setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Namespace: setup.Namespace(), Name: firstTeamName}, team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting Team")
				g.Expect(team.Name).To(BeEquivalentTo(firstTeamName))
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "condition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "SCIMAccessReadyCondition should be set to true")
				g.Expect(team.Status.StatusConditions.IsReadyTrue()).To(BeTrue(), "Ready condition should be set to true")
			}).Should(Succeed())

			err := setup.Delete(test.Ctx, team)
			Expect(err).ToNot(HaveOccurred(), "there must be no error deleting Team")

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Namespace: setup.Namespace(), Name: firstTeamName}, team)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "Team should be deleted")
			}).Should(Succeed())
		})
	})

	Context("reconciling with missing SCIM config in Organization", func() {
		BeforeEach(func() {
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
			setup.CreateOrganization(test.Ctx, setup.Namespace())
		})

		It("should reconcile Teams when Org's SCIM config changes", func() {
			By("creating first Team")
			team := setup.CreateTeam(test.Ctx, firstTeamName, test.WithMappedIDPGroup(validIdpGroupName))

			By("checking Team status")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting first Team")
				g.Expect(team.Status.Members).To(BeNil(), "the Team should have no Members")
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "SCIMAccessReadyCondition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionFalse), "SCIMAccessReadyCondition should be set to false")
				g.Expect(scimAccessReadyCondition.Reason).To(Equal(greenhousev1alpha1.SCIMAPIUnavailableReason), "reason should be set to SCIMAPIUnavailable")
			}).Should(Succeed(), "Team should reflect missing secret in status")

			By("creating missing secret")
			createSecretForSCIMConfig(setup.Namespace())

			var organization = &greenhousev1alpha1.Organization{}
			Expect(
				setup.Get(test.Ctx, types.NamespacedName{Name: setup.Namespace(), Namespace: ""}, organization),
			).To(Succeed(), "there should be no error getting the organization")

			By("updating Organization Status")
			updateOrganizationStatusWithSCIMAvailability(organization)

			By("updating SCIMConfig in Organization")
			organization.Spec.Authentication = &greenhousev1alpha1.Authentication{
				SCIMConfig: &greenhousev1alpha1.SCIMConfig{
					BaseURL: usersServer.URL + "/scim",
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
			Expect(setup.Update(test.Ctx, organization)).To(Succeed(), "there should be no error updating Organization")

			By("ensuring Team is updated after change in Organization")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, client.ObjectKeyFromObject(team), team)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting first Team")
				g.Expect(team.Status.Members).To(HaveLen(2), "the Team should have exactly two Members")
				scimAccessReadyCondition := team.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.SCIMAccessReadyCondition)
				g.Expect(scimAccessReadyCondition).ToNot(BeNil(), "SCIMAccessReadyCondition should not be nil")
				g.Expect(scimAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "SCIMAccessReadyCondition should be set to true")
			}).Should(Succeed(), "Team should be reconciled")
		})
	})
})

func createTestOrgWithSecret(namespace string) {
	By("creating a secret")
	createSecretForSCIMConfig(namespace)

	By("creating organization with name: " + namespace)
	org := setup.CreateOrganization(test.Ctx, namespace, func(o *greenhousev1alpha1.Organization) {
		o.Spec.Authentication = &greenhousev1alpha1.Authentication{
			SCIMConfig: &greenhousev1alpha1.SCIMConfig{
				BaseURL: usersServer.URL + "/scim",
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

	updateOrganizationStatusWithSCIMAvailability(org)
}

func updateOrganizationStatusWithSCIMAvailability(org *greenhousev1alpha1.Organization) {
	orgStatus := org.Status
	orgStatus.SetConditions(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.SCIMAPIAvailableCondition, "", ""))
	_, err := clientutil.PatchStatus(test.Ctx, setup.Client, org, func() error {
		org.Status = orgStatus
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error patching org status")
}

func createSecretForSCIMConfig(namespace string) {
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
	err := setup.Create(test.Ctx, &testSecret)
	Expect(err).ToNot(HaveOccurred(), "there must be no error creating a secret")
}
