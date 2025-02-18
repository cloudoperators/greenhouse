// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/scim"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Test Organization reconciliation", Ordered, func() {
	const (
		validIdpGroupName      = "SOME_IDP_GROUP_NAME"
		otherValidIdpGroupName = "ANOTHER_IDP_GROUP"
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
				org.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
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
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(validIdpGroupName), "Admin Team should have the same idp group name as organization")
				g.Expect(team.OwnerReferences).Should(ContainElement(ownerRef), "Admin Team must have the correct owner reference")
			}).Should(Succeed(), "Admin team should be created for organization")
		})

		It("should update admin team when MappedOrgAdminIDPGroup in org changes", func() {
			testOrgName := "test-org-3"
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
			})

			var team = &greenhousev1alpha1.Team{}
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(validIdpGroupName), "Admin team should be creted with valid IDPGroup")
			}).Should(Succeed(), "Admin team should be created with valid IDPGroup")

			By("updating MappedOrgAdminIDPGroup in Organization")
			_, err := clientutil.Patch(test.Ctx, test.K8sClient, testOrg, func() error {
				testOrg.Spec.MappedOrgAdminIDPGroup = otherValidIdpGroupName
				return nil
			})
			Expect(err).To(Succeed(), "there must be no error updating the organization")

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(otherValidIdpGroupName), "Admin team should be updated with new IDPGroup")
			}).Should(Succeed(), "Admin team should be updated with new IDPGroup")
		})

		It("should update admin team when MappedIDPGroup in team changes", func() {
			testOrgName := "test-org-4"
			setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
			})
			var team = &greenhousev1alpha1.Team{}
			Eventually(func() error {
				return setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
			}).ShouldNot(HaveOccurred(), "there should be no error getting org admin team")

			By("changing MappedIDPGroup in Team")
			_, err := clientutil.Patch(test.Ctx, test.K8sClient, team, func() error {
				team.Spec.MappedIDPGroup = otherValidIdpGroupName
				return nil
			})

			Expect(err).To(Succeed(), "there must be no error updating the team")

			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName + "-admin", Namespace: testOrgName}, team)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting org admin team")
				g.Expect(team.Spec.MappedIDPGroup).To(Equal(validIdpGroupName), "Admin team should be updated with organization IDPGroup")
			}).Should(Succeed(), "Admin team should be updated with organization IDPGroup")
		})

		It("should recreate org admin team if deleted", func() {
			testOrgName := "test-org-5"
			setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
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

		It("should set correct status condition when creating Organization with SCIM Config as BasicAuth", func() {
			By("creating Organization with SCIM Config")
			testOrgName := setup.Namespace()

			By("creating secret for SCIM Config")
			createSecretForSCIMConfig(testOrgName)

			By("creating Organization with SCIM Config")
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName,
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
				},
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.Authentication = &greenhousev1alpha1.Authentication{
						SCIMConfig: &greenhousev1alpha1.SCIMConfig{
							BaseURL:  groupsServer.URL + "/scim",
							AuthType: scim.Basic,
							BasicAuthUser: greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "test-secret",
									Key:  "basicAuthUser",
								},
							},
							BasicAuthPw: greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "test-secret",
									Key:  "basicAuthPw",
								},
							},
						},
					}
				},
			)

			By("checking Organization status")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				scimAPIAvailableCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
				g.Expect(scimAPIAvailableCondition).ToNot(BeNil(), "SCIMAPIAvailableCondition should be set on Organization")
				g.Expect(scimAPIAvailableCondition.IsTrue()).To(BeTrue(), "SCIMAPIAvailableCondition should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")
		})

		It("should set correct status condition when creating Organization with SCIM Config as BearerToken", func() {
			By("creating Organization with SCIM Config")
			testOrgName := setup.Namespace()

			By("creating secret for SCIM Config")
			createSecretForSCIMConfig(testOrgName)

			By("creating Organization with SCIM Config")
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName,
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
				},
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.Authentication = &greenhousev1alpha1.Authentication{
						SCIMConfig: &greenhousev1alpha1.SCIMConfig{
							BaseURL:  groupsServer.URL + "/scim",
							AuthType: scim.BearerToken,
							BearerToken: greenhousev1alpha1.ValueFromSource{
								Secret: &greenhousev1alpha1.SecretKeyReference{
									Name: "test-secret",
									Key:  "bearerToken",
								},
							},
						},
					}
				},
			)

			By("checking Organization status")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				scimAPIAvailableCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
				g.Expect(scimAPIAvailableCondition).ToNot(BeNil(), "SCIMAPIAvailableCondition should be set on Organization")
				g.Expect(scimAPIAvailableCondition.IsTrue()).To(BeTrue(), "SCIMAPIAvailableCondition should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")
		})

		It("should set correct status condition after updating Organization with SCIM Config", func() {
			By("creating Organization without SCIM Config")
			testOrgName := setup.Namespace()
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName, func(org *greenhousev1alpha1.Organization) {
				org.Spec.MappedOrgAdminIDPGroup = validIdpGroupName
			})

			By("checking Organization status")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				scimAPIAvailableCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
				g.Expect(scimAPIAvailableCondition).ToNot(BeNil(), "SCIMAPIAvailableCondition should be set on Organization")
				g.Expect(scimAPIAvailableCondition.Status).To(Equal(metav1.ConditionUnknown), "SCIMAPIAvailableCondition should be set to Unknown on Organization")
				readyCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set on Organization")
				g.Expect(readyCondition.IsTrue()).To(BeTrue(), "ReadyCondition should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")

			By("updating Organization with SCIM Config without the secret")
			Eventually(func(g Gomega) { // In 'Eventually' block to avoid flaky tests.
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				testOrg.Spec.Authentication = &greenhousev1alpha1.Authentication{
					SCIMConfig: &greenhousev1alpha1.SCIMConfig{
						BaseURL:  groupsServer.URL + "/scim",
						AuthType: scim.Basic,
						BasicAuthUser: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "basicAuthUser",
							},
						},
						BasicAuthPw: greenhousev1alpha1.ValueFromSource{
							Secret: &greenhousev1alpha1.SecretKeyReference{
								Name: "test-secret",
								Key:  "basicAuthPw",
							},
						},
					},
				}
				err = setup.Update(test.Ctx, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error updating the Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")

			By("checking Organization status")
			Eventually(func(g Gomega) {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				scimAPIAvailableCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
				g.Expect(scimAPIAvailableCondition).ToNot(BeNil(), "SCIMAPIAvailableCondition should be set on Organization")
				g.Expect(scimAPIAvailableCondition.IsFalse()).To(BeTrue(), "SCIMAPIAvailableCondition should be False on Organization")
				readyCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set on Organization")
				g.Expect(readyCondition.IsFalse()).To(BeTrue(), "ReadyCondition should be False on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")

			By("creating secret for SCIM Config")
			createSecretForSCIMConfig(testOrgName)

			By("setting labels on Organization to trigger reconciliation")
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				testOrg.Labels = map[string]string{"test": "label"}
				return setup.Update(test.Ctx, testOrg)
			})
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the Organization")

			By("checking Organization status")
			Eventually(func(g Gomega) {
				var testOrg = new(greenhousev1alpha1.Organization)
				err := setup.Get(test.Ctx, types.NamespacedName{Name: testOrgName}, testOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization")
				scimAPIAvailableCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.SCIMAPIAvailableCondition)
				g.Expect(scimAPIAvailableCondition).ToNot(BeNil(), "SCIMAPIAvailableCondition should be set on Organization")
				g.Expect(scimAPIAvailableCondition.IsTrue()).To(BeTrue(), "SCIMAPIAvailableCondition should be True on Organization")
				readyCondition := testOrg.Status.GetConditionByType(greenhousev1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set on Organization")
				g.Expect(readyCondition.IsTrue()).To(BeTrue(), "ReadyCondition should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")
		})
	})
})

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
			"bearerToken":   []byte("100b8cad7cf2a56f6df78f171f97a1ec"),
		},
	}
	err := test.K8sClient.Create(test.Ctx, &testSecret)
	Expect(err).ToNot(HaveOccurred(), "there must be no error creating a secret")
}
