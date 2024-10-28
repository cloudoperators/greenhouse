// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Test Organization reconciliation", Ordered, func() {
	const (
		someIdpGroupName    = "SOME-IDP-GROUP"
		anotherIdpGroupName = "ANOTHER-IDP-GROUP"
	)
	var (
		setup *test.TestSetup
	)

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
	})

	When("reconciling an organization", func() {
		It("should create a namespace for new organization", func() {
			testOrgName := "test-org-1"
			setup.CreateOrganization(test.Ctx, testOrgName)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testOrgName}})
		})

		It("should create admin team for organization", func() {
			testOrgName := "test-org-2"
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = someIdpGroupName
			})
			b := true
			ownerRef := metav1.OwnerReference{
				APIVersion:         greenhousev1alpha1.GroupVersion.String(),
				Kind:               "Organization",
				UID:                testOrg.UID,
				Name:               testOrg.Name,
				Controller:         &b,
				BlockOwnerDeletion: &b,
			}

			var team = &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "Admin Team should be created for organization")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(someIdpGroupName), "Admin Team should have the same idp group name as organization")
				g.Expect(team.OwnerReferences).Should(ContainElement(ownerRef), "Admin Team must have the correct owner reference")
			}).Should(Succeed(), "Admin team should be created for organization")
		})

		It("should update admin team when MappedOrgAdminIDPGroup in org changes", func() {
			testOrgName := "test-org-3"
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = someIdpGroupName
			})

			By("updating MappedOrgAdminIDPGroup in Organization")
			testOrg.Spec.MappedOrgAdminIDPGroup = anotherIdpGroupName
			err := setup.Update(test.Ctx, testOrg)
			Expect(err).ToNot(HaveOccurred(), "there must be no error updating the organization")

			var team = &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(anotherIdpGroupName), "Admin team should be updated with new IDPGroup")
			}).Should(Succeed(), "Admin team should be updated with new IDPGroup")
		})

		It("should update admin team when MappedIDPGroup in team changes", func() {
			testOrgName := "test-org-4"
			setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = someIdpGroupName
			})
			var team = &greenhousev1alpha1.Team{}
			Eventually(func() error {
				return setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
			}).ShouldNot(HaveOccurred(), "there should be no error getting org admin team")

			By("changing MappedIDPGroup in Team")
			team.Spec.MappedIDPGroup = anotherIdpGroupName
			Expect(setup.Update(test.Ctx, team)).To(Succeed(), "there must be no error updating the team")

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(someIdpGroupName), "Admin team should be updated with organization IDPGroup")
			}).Should(Succeed(), "Admin team should be updated with organization IDPGroup")
		})

		It("should recreate org admin team if deleted", func() {
			testOrgName := "test-org-5"
			setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = someIdpGroupName
			})
			var team = &greenhousev1alpha1.Team{}
			Eventually(func() error {
				return setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
			}).ShouldNot(HaveOccurred(), "there should be no error getting org admin team")

			By("deleting org admin team")
			test.EventuallyDeleted(test.Ctx, setup.Client, team)

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
			}).Should(Succeed(), "Org admin team should be recreated")
		})
	})
})
