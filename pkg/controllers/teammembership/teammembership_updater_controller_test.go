package teammembership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	teamName = "test-team-1"
)

var _ = Describe("TeammembershipUpdaterController", func() {
	When("reconciling a teammembership", func() {
		It("should update existing TM without users", func() {
			By("creating a test Team")
			err := test.K8sClient.Create(test.Ctx, &greenhousev1alpha1.Team{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "Team",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      teamName,
					Namespace: test.TestNamespace,
				},
				Spec: greenhousev1alpha1.TeamSpec{
					Description:    "Test Team 1",
					MappedIDPGroup: "SOME_IDP_GROUP_NAME",
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a Team")

			By("ensuring the Team has been created")
			testTeamName := types.NamespacedName{Name: teamName, Namespace: test.TestNamespace}
			testTeam := &greenhousev1alpha1.Team{}
			Eventually(func() error {
				return test.K8sClient.Get(test.Ctx, testTeamName, testTeam)
			}).Should(Succeed(), "the Team should be created")

			By("creating a test TeamMembership")
			err = test.K8sClient.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      teamName,
					Namespace: test.TestNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("ensuring the TeamMembership has been created")
			testTeamMembershipName := types.NamespacedName{Name: teamName, Namespace: test.TestNamespace}
			testTeamMembership := &greenhousev1alpha1.TeamMembership{}
			Eventually(func() error {
				return test.K8sClient.Get(test.Ctx, testTeamMembershipName, testTeamMembership)
			}).Should(Succeed(), "the TeamMembership should be created")

			By("ensuring Team users have been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := test.K8sClient.List(test.Ctx, teamMemberships)
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(len(teamMemberships.Items)).To(Equal(1), "there should be exactly one TeamMembership")
				g.Expect(len(teamMemberships.Items[0].Spec.Members)).To(Equal(2), "the TeamMembership should have exactly 2 Members")
			}).Should(Succeed(), "the TeamMembership should be reconciled")
		})
	})
})
