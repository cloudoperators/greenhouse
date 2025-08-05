// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Organization ServiceProxyReconciler", Ordered, func() {
	var setup *test.TestSetup
	var org *greenhousev1alpha1.Organization
	serviceProxyPluginDefinition := test.NewClusterPluginDefinition(test.Ctx, "service-proxy")

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "org-rbac-test")
	})

	Context("plugin definition for service proxy is missing", Ordered, func() {
		It("should skip creating service-proxy", func() {
			By("ensuring service-proxy plugin definition does not exist")
			var pluginDefinition = new(greenhousev1alpha1.ClusterPluginDefinition)
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: ""}, pluginDefinition)
			Expect(err).To(HaveOccurred(), "there should be an error getting the service-proxy plugin definition")

			By("creating an organization")
			org = setup.CreateDefaultOrgWithOIDCSecret(test.Ctx, "test-serviceproxy-org1")

			By("ensuring service-proxy plugin is not created for organization")
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: org.Name}, org)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the organization")
				serviceProxyCondition := org.Status.GetConditionByType(greenhousev1alpha1.ServiceProxyProvisioned)
				g.Expect(serviceProxyCondition).ToNot(BeNil(), "ServiceProxyProvisioned condition should not be nil on Organization")
				g.Expect(serviceProxyCondition.IsFalse()).To(BeTrue(), "ServiceProxyProvisioned condition should be False on Organization")
				g.Expect(serviceProxyCondition.Reason).To(Equal(greenhousev1alpha1.ServiceProxyNotFound), "ServiceProxyProvisioned condition reason should be ServiceProxyNotFound")
			}).Should(Succeed(), "service-proxy plugin should not be created for organization")
		})
	})

	Context("plugin definition for service proxy is present", Ordered, func() {
		It("should create service-proxy plugin definition", func() {
			err := test.K8sClient.Create(test.Ctx, serviceProxyPluginDefinition)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the service-proxy plugin definition")

			By("ensuring a service-proxy plugin has been created for organization")
			var plugin = new(greenhousev1alpha1.Plugin)
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: org.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
		})
		It("should create default organization", func() {
			team := setup.CreateTeam(test.Ctx, "test-team1", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			defaultOrg := setup.CreateDefaultOrgWithOIDCSecret(test.Ctx, team.Name)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: defaultOrg.Name}})
			By("By check if the default organization is READY with oidc config")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: defaultOrg.Name}, defaultOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the organization")
				oidcCondition := defaultOrg.Status.GetConditionByType(greenhousev1alpha1.OrganizationOICDConfigured)
				g.Expect(oidcCondition.IsTrue()).To(BeTrue(), "OrganizationOICDConfigured should be True on Organization")
				// Check if the service-proxy plugin is created and the technical secret is created
				var plugin = new(greenhousev1alpha1.Plugin)
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: defaultOrg.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
				var secret = new(corev1.Secret)
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: defaultOrg.Name + technicalSecretSuffix, Namespace: defaultOrg.Name}, secret)
				g.Expect(err).ToNot(HaveOccurred(), "org technical secret should have been created by controller")
				g.Expect(secret.Data).To(HaveKeyWithValue(cookieSecretKey, Not(BeEmpty())), "org technical secret should contain a non-empty cookie secret")
				g.Expect(plugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "oauth2proxy.enabled", Value: &apiextensionsv1.JSON{Raw: []byte("\"true\"")}}))
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, team)
		})
	})
})
