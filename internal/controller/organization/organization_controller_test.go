// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/dex"
	dexapi "github.com/cloudoperators/greenhouse/internal/dex/api"
	"github.com/cloudoperators/greenhouse/internal/scim"
	"github.com/cloudoperators/greenhouse/internal/test"
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

	When("reconciling an organization", Ordered, func() {
		It("should create a namespace for new organization", func() {
			testOrgName := "test-org-1"
			setup.CreateOrganization(test.Ctx, testOrgName)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testOrgName}})
		})

		It("should create admin team for organization", func() {
			testOrgName := "test-org-2"
			testOrg := setup.CreateOrganization(test.Ctx, testOrgName, test.WithMappedAdminIDPGroup(validIdpGroupName))
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
				test.WithMappedAdminIDPGroup(validIdpGroupName),
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.Authentication = &greenhousev1alpha1.Authentication{
						SCIMConfig: &greenhousev1alpha1.SCIMConfig{
							BaseURL:  groupsServer.URL + "/scim",
							AuthType: scim.Basic,
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
				test.WithMappedAdminIDPGroup(validIdpGroupName),
				func(o *greenhousev1alpha1.Organization) {
					o.Spec.Authentication = &greenhousev1alpha1.Authentication{
						SCIMConfig: &greenhousev1alpha1.SCIMConfig{
							BaseURL:  groupsServer.URL + "/scim",
							AuthType: scim.BearerToken,
							BearerToken: &greenhousev1alpha1.ValueFromSource{
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
				readyCondition := testOrg.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
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
				readyCondition := testOrg.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
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
				readyCondition := testOrg.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set on Organization")
				g.Expect(readyCondition.IsTrue()).To(BeTrue(), "ReadyCondition should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")
		})

		It("should create dex resources if oidc is enabled", func() {
			team := setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

			defaultOrg := setup.CreateDefaultOrgWithOIDCSecret(test.Ctx, team.Name)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: defaultOrg.Name}})
			By("By check if the default organization is READY with oidc config")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: defaultOrg.Name}, defaultOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the organization")
				oidcCondition := defaultOrg.Status.GetConditionByType(greenhousev1alpha1.OrganizationOICDConfigured)
				g.Expect(oidcCondition).ToNot(BeNil(), "OrganizationOICDConfigured should be set on Organization")
				g.Expect(oidcCondition.IsTrue()).To(BeTrue(), "OrganizationOICDConfigured should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")

			By("creating a test organization for OIDC")
			oidcOrg := setup.CreateOrganization(test.Ctx, "test-oidc-org", test.WithMappedAdminIDPGroup(validIdpGroupName))
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: oidcOrg.Name}})

			By("creating a secret for OIDC config")
			oidcOrgSecret := setup.CreateOrgOIDCSecret(test.Ctx, oidcOrg.Name, team.Name)
			By("updating the organization with OIDC config")
			oidcOrg = setup.UpdateOrganization(test.Ctx,
				oidcOrg.Name,
				test.WithOIDCConfig(test.OIDCIssuer, oidcOrgSecret.Name, test.OIDCClientIDKey, test.OIDCClientSecretKey),
				test.WithAdditionalRedirects("https://example.com/app", "http://localhost:33768/auth/callback"),
			)
			defaultOrg = setup.UpdateOrganization(test.Ctx, defaultOrg.Name, test.WithAdditionalRedirects("https://foo.bar/app"))

			By("checking Organization status")
			checkOrganizationReadyStatus(defaultOrg.Name)
			checkOrganizationReadyStatus(oidcOrg.Name)

			if DexStorageType == dex.K8s {
				By("checking dex connector resource")
				connectors := &dexapi.ConnectorList{}
				oAuthClients := &dexapi.OAuth2ClientList{}

				err := setup.List(test.Ctx, connectors)
				Expect(err).ToNot(HaveOccurred(), "there should be no error listing dex connectors")
				Expect(len(connectors.Items)).To(BeNumerically(">", 1), "there should be at least one dex connector")
				err = setup.List(test.Ctx, oAuthClients)
				Expect(err).ToNot(HaveOccurred(), "there should be no error listing dex oauth clients")
				Expect(len(oAuthClients.Items)).To(BeNumerically(">", 1), "there should be at least one dex oauth client")

				filteredOrgConnector := slices.DeleteFunc(connectors.Items, func(c dexapi.Connector) bool {
					return c.ID != oidcOrg.Name
				})
				Expect(filteredOrgConnector).To(HaveLen(1), "there should be one dex connector after filtering")
				Expect(filteredOrgConnector[0].ID).To(Equal(oidcOrg.Name), "the connector ID should be equal to organization name")

				By("checking dex oauth client resource")
				filteredOAuthClients := slices.DeleteFunc(oAuthClients.Items, func(c dexapi.OAuth2Client) bool {
					return c.ID != oidcOrg.Name && c.ID != defaultOrg.Name
				})
				Expect(filteredOAuthClients).To(HaveLen(2), "there should be two dex oauth clients")
				for _, orgClient := range filteredOAuthClients {
					switch orgClient.ID {
					case oidcOrg.Name:
						Expect(orgClient.ID).To(Equal(oidcOrg.Name), "the oauth client ID should be equal to organization name")
						Expect(orgClient.RedirectURIs).To(HaveLen(5), "the oauth client redirect URIs should have the default 3 elements + 2 additionalRedirects")
						Expect(orgClient.RedirectURIs).To(ContainElements("https://example.com/app", "http://localhost:33768/auth/callback"), "the oauth client redirect URIs should be equal to organization redirect URIs")
					case defaultOrg.Name:
						Expect(orgClient.ID).To(Equal(defaultOrg.Name), "the oauth client ID should be equal to organization name")
						Expect(orgClient.RedirectURIs).To(ContainElements("https://test-oidc-org.dashboard."), "the greenhouse client should contain the org's dashboard redirect uri")
						Expect(len(orgClient.RedirectURIs)).To(BeNumerically(">=", 5), "the oauth client redirect URIs should have 4 elements (default 3 + 1 org + 1 additional)")
					default:
						Fail("unexpected oauth client ID")
					}
				}
				By("deleting the organizations")
				test.EventuallyDeleted(test.Ctx, test.K8sClient, &greenhousev1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: oidcOrg.Name}})
				By("checking if the dex resources are deleted")
				err = setup.List(test.Ctx, oAuthClients)
				Expect(err).ToNot(HaveOccurred(), "there should be no error listing dex oauth clients")
				Expect(oAuthClients.Items).ToNot(ContainElement(
					MatchFields(IgnoreExtras, Fields{"ID": Equal(oidcOrg.Name)}),
				), "the oauth client list should not contain the deleted organization")
			}
			test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		})
	})
})

func checkOrganizationReadyStatus(orgName string) {
	By("checking Organization status")
	Eventually(func(g Gomega) {
		org := &greenhousev1alpha1.Organization{}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: orgName}, org)
		g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the Organization "+orgName)
		readyCondition := org.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(readyCondition).ToNot(BeNil(), "ReadyCondition should be set on Organization "+orgName)
		g.Expect(readyCondition.IsTrue()).To(BeTrue(), "ReadyCondition should be True on Organization "+orgName)
		oidcCondition := org.Status.GetConditionByType(greenhousev1alpha1.OrganizationOICDConfigured)
		g.Expect(oidcCondition).ToNot(BeNil(), "OrganizationOICDConfigured should be set on Organization "+orgName)
		g.Expect(oidcCondition.IsTrue()).To(BeTrue(), "OrganizationOICDConfigured should be True on Organization "+orgName)
	}).Should(Succeed(), "Organization should have set correct status condition")
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
			"bearerToken":   []byte("100b8cad7cf2a56f6df78f171f97a1ec"),
		},
	}
	err := test.K8sClient.Create(test.Ctx, &testSecret)
	Expect(err).ToNot(HaveOccurred(), "there must be no error creating a secret")
}
