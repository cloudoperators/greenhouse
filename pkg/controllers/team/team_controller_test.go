// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const teamName = "test-team-0000"

var _ = Describe("TeamControllerTest", Ordered, func() {
	It("Should update status of team with members", func() {
		err := test.K8sClient.Create(test.Ctx, &greenhouseapisv1alpha1.Team{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Team",
				APIVersion: greenhouseapisv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      teamName,
				Namespace: test.TestNamespace,
			},
			Spec: greenhouseapisv1alpha1.TeamSpec{
				Description:    "",
				MappedIDPGroup: "MAP_IDP_GROUP",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = test.K8sClient.Create(test.Ctx, &greenhouseapisv1alpha1.TeamMembership{
			TypeMeta: metav1.TypeMeta{
				Kind:       "TeamMembership",
				APIVersion: greenhouseapisv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-membership",
				Namespace: test.TestNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: greenhouseapisv1alpha1.GroupVersion.String(),
						Kind:       "Team",
						Name:       teamName,
						UID:        "uhuihiuh",
					},
				},
			},
			Spec: greenhouseapisv1alpha1.TeamMembershipSpec{
				Members: []greenhouseapisv1alpha1.User{
					{
						ID:        "d2a72c04-42d2-426a-942d-af9609c4cd00",
						FirstName: "John",
						LastName:  "Doe",
						Email:     "john.doe@example.com",
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func(g Gomega) {
			team := &greenhouseapisv1alpha1.Team{}
			err := test.K8sClient.Get(test.Ctx, client.ObjectKey{Name: teamName, Namespace: test.TestNamespace}, team)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(team.Status.Members).To(HaveLen(1))
		}).Should(Succeed())
	})
})
