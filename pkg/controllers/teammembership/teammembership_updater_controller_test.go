package teammembership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const (
	teamName     = "test-team-1"
	idpGroupName = "SOME_IDP_GROUP_NAME"
)

var (
	setup *test.TestSetup
)

var _ = Describe("TeammembershipUpdaterController", func() {
	When("reconciling a teammembership", func() {
		BeforeEach(func() {
			By("creating new test setup")
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
		})

		It("should update existing TM without users", func() {
			By("creating a test Team")

			team := setup.CreateTeam(test.Ctx, teamName, test.WithMappedIDPGroup(idpGroupName))

			By("creating a test TeamMembership")
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      team.Name,
					Namespace: setup.Namespace(),
				},
			})
			Expect(err).NotTo(HaveOccurred(), "there must be no error creating a TeamMembership")

			By("ensuring the TeamMembership has been created")
			ensureTeamMembershipHasBeenCreated(team.Name)

			By("ensuring Team users have been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(len(teamMemberships.Items)).To(Equal(1), "there should be exactly one TeamMembership")
				g.Expect(len(teamMemberships.Items[0].Spec.Members)).To(Equal(2), "the TeamMembership should have exactly two Members")
				g.Expect(teamMemberships.Items[0].Status.LastChangedTime).ToNot(BeNil(), "TeamMembership status should have updated LastChangedTime")
			}).Should(Succeed(), "the TeamMembership should be reconciled")
		})

		It("should update existing TM with users", func() {
			By("creating a test Team")
			team := setup.CreateTeam(test.Ctx, teamName, test.WithMappedIDPGroup(idpGroupName))

			By("creating a test TeamMembership with 1 existing user")
			err := setup.Create(test.Ctx, &greenhousev1alpha1.TeamMembership{
				TypeMeta: metav1.TypeMeta{
					APIVersion: greenhousev1alpha1.GroupVersion.Group,
					Kind:       "TeamMembership",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      team.Name,
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

			By("ensuring the TeamMembership has been created")
			ensureTeamMembershipHasBeenCreated(team.Name)

			By("ensuring TeamMembership users have been reconciled")
			Eventually(func(g Gomega) {
				teamMemberships := &greenhousev1alpha1.TeamMembershipList{}
				err := setup.List(test.Ctx, teamMemberships, &client.ListOptions{Namespace: setup.Namespace()})
				g.Expect(err).ShouldNot(HaveOccurred(), "unexpected error getting TeamMemberships")
				g.Expect(len(teamMemberships.Items)).To(Equal(1), "there should be exactly one TeamMembership")
				g.Expect(len(teamMemberships.Items[0].Spec.Members)).To(Equal(2), "the TeamMembership should have exactly two Members")
				g.Expect(teamMemberships.Items[0].Status.LastChangedTime).ToNot(BeNil(), "TeamMembership status should have updated LastChangedTime")
			}).Should(Succeed(), "the TeamMembership should be reconciled")
		})
	})
})

func ensureTeamMembershipHasBeenCreated(teamName string) {
	testTeamMembershipName := types.NamespacedName{Name: teamName, Namespace: setup.Namespace()}
	testTeamMembership := &greenhousev1alpha1.TeamMembership{}
	Eventually(func() error {
		return setup.Get(test.Ctx, testTeamMembershipName, testTeamMembership)
	}).Should(Succeed(), "the TeamMembership should be created")
}
