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

var _ = Describe("Test Organization reconciliation", func() {
	const (
		testOrgName  = "test-org-1"
		idpGroupName = "SOME-IDP-GROUP"
	)
	var (
		setup    *test.TestSetup
		ownerRef metav1.OwnerReference
	)

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
	})

	When("reconciling an organization", func() {
		It("should create a namespace for new organization", func() {
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testOrgName}})
			ownerRef = metav1.OwnerReference{
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
				Kind:       "Organization",
				UID:        testOrg.UID,
				Name:       testOrg.Name,
			}
		})

		It("should create admin team for organization", func() {
			var team = &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "Admin Team should be created for organization")
			}).Should(Succeed(), "Admin team should be created for organization")

			Eventually(team.OwnerReferences).Should(ContainElement(ownerRef), "Admin Team must have the correct owner reference")
		})

		It("should update admin team when MappedOrgAdminIDPGroup changes", func() {
			var org = &greenhousev1alpha1.Organization{}
			err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName, Namespace: ""}, org)
			Expect(err).ToNot(HaveOccurred(), "there must be no error getting the organization")

			By("updating MappedOrgAdminIDPGroup in Organization")
			org.Spec.MappedOrgAdminIDPGroup = idpGroupName
			err = setup.Update(test.Ctx, org)
			Expect(err).ToNot(HaveOccurred(), "there must be no error updating the organization")

			var team = &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(idpGroupName), "Admin team should be updated with new IDPGroup")
			}).Should(Succeed(), "Admin team should be updated with new IDPGroup")
			Eventually(team.OwnerReferences).Should(ContainElement(ownerRef), "Admin Team must have the correct owner reference")
		})
	})
})
