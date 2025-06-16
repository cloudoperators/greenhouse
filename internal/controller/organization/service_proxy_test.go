// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Organization ServiceProxyReconciler", Ordered, func() {
	var setup *test.TestSetup
	serviceProxyPluginDefinition := test.NewPluginDefinition(test.Ctx, "service-proxy")

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "org-rbac-test")
	})

	When("plugin definition for service proxy is missing", func() {
		It("should log about missing plugin definition and create plugin when it's added", func() {
			By("ensuring service-proxy plugin definition does not exist")
			var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: ""}, pluginDefinition)
			Expect(err).To(HaveOccurred(), "there should be an error getting the service-proxy plugin definition")

			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating an organization")
			org := setup.CreateOrganization(test.Ctx, "test-serviceproxy-org1")

			By("ensuring ServiceProxyController logged about missing plugin definition")
			Eventually(func() []byte {
				return tee.Contents()
			}).Should(ContainSubstring("plugin definition for service-proxy not found"),
				"ServiceProxyController should log about missing plugin definition")

			By("creating service-proxy plugin definition")
			err = test.K8sClient.Create(test.Ctx, serviceProxyPluginDefinition)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the service-proxy plugin definition")

			By("ensuring a service-proxy plugin has been created for organization")
			var plugin = new(greenhousev1alpha1.Plugin)
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: org.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
		})
	})

	When("plugin definition for service proxy is present", func() {
		It("should create service-proxy plugin for organization", func() {
			By("getting service-proxy plugin definition")
			var pluginDefinition = new(greenhousev1alpha1.ClusterPluginDefinition)
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: ""}, pluginDefinition)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the service-proxy plugin definition")

			By("creating an organization")
			org := setup.CreateOrganization(test.Ctx, "test-serviceproxy-org2")

			By("ensuring a service-proxy plugin has been created for organization")
			Eventually(func(g Gomega) {
				var plugin = new(greenhousev1alpha1.Plugin)
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: org.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
		})
	})

	When("organization is annotated with the oauth2proxy preview annotation", Ordered, func() {
		It("should create default organization", func() {
			defaultOrg := setup.CreateDefaultOrgWithOIDCSecret(test.Ctx)
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: defaultOrg.Name}})
			By("By check if the default organization is READY with oidc config")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: defaultOrg.Name}, defaultOrg)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the organization")
				oidcCondition := defaultOrg.Status.GetConditionByType(greenhousev1alpha1.OrganizationOICDConfigured)
				g.Expect(oidcCondition).ToNot(BeNil(), "OrganizationOICDConfigured should be set on Organization")
				g.Expect(oidcCondition.IsTrue()).To(BeTrue(), "OrganizationOICDConfigured should be True on Organization")
			}).Should(Succeed(), "Organization should have set correct status condition")
		})
		It("should enable the oauth-proxy feature for the organization", func() {
			By("creating an organization with the oauth preview annotation & oauth config")
			org, _ := setup.CreateOrganizationWithOIDCConfig(test.Ctx, setup.Namespace())
			test.EventuallyCreated(test.Ctx, test.K8sClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org.Name}})
			By("annotating the organization")
			org = setup.UpdateOrganization(test.Ctx, org.Name, test.WithOrgAnnotations(map[string]string{oauthPreviewAnnotation: "true"}))
			By("ensuring a service-proxy plugin has been created for organization")
			Eventually(func(g Gomega) {
				var plugin = new(greenhousev1alpha1.Plugin)
				err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: org.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
				var secret = new(corev1.Secret)
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: org.Name + technicalSecretSuffix, Namespace: org.Name}, secret)
				g.Expect(err).ToNot(HaveOccurred(), "org technical secret should have been created by controller")
				g.Expect(secret.Data).To(HaveKeyWithValue(cookieSecretKey, Not(BeEmpty())), "org technical secret should contain a non-empty cookie secret")
				g.Expect(plugin.Spec.OptionValues).To(ContainElement(greenhousev1alpha1.PluginOptionValue{Name: "oauth2proxy.enabled", Value: &apiextensionsv1.JSON{Raw: []byte("\"true\"")}}))
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
		})
	})
})
