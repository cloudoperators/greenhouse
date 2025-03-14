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
	"github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Test TeamRole Seeding", func() {
	const (
		orgName = "test-teamrole-seeding"
	)
	var (
		setup *test.TestSetup
	)

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "teamrole-seeding-test")
	})

	When("seeding a TeamRole", func() {
		It("should create the Organization successfully", func() {
			setup.CreateOrganization(test.Ctx, orgName)

			// ensure the organization's namespace is created
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: orgName}})

		})

		It("should have created the default TeamRoles", func() {
			for name, spec := range organization.ExportDefaultTeamRoles {
				teamRoleID := types.NamespacedName{Name: name, Namespace: orgName}
				actTeamRole := &greenhousev1alpha1.TeamRole{}
				Eventually(func() bool {
					return test.K8sClient.Get(test.Ctx, teamRoleID, actTeamRole) == nil
				}).Should(BeTrue(), "TeamRole %s must be created", name)
				Expect(actTeamRole.Spec).To(Equal(spec), "TeamRole %s must have the correct spec", name)
			}
		})
	})
})
