// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var _ = Describe("Organization ServiceProxyReconciler", func() {
	var setup *test.TestSetup

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "org-serviceproxy-test")
	})

	When("plugin definition for service proxy is missing", func() {
		It("should log about missing plugin definition", func() {
			By("ensuring service-proxy plugin definition does not exist")
			var serviceProxyPluginDefinition = new(greenhousev1alpha1.PluginDefinition)
			err := setup.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: ""}, serviceProxyPluginDefinition)
			Expect(err).To(HaveOccurred(), "there should be an error getting the service-proxy plugin definition")

			By("teeing logger to a custom writer")
			tee := gbytes.NewBuffer()
			GinkgoWriter.TeeTo(tee)
			defer GinkgoWriter.ClearTeeWriters()

			By("creating an organization")
			setup.CreateOrganization(test.Ctx, "test-serviceproxy-org1")

			By("ensuring ServiceProxyController logged about missing plugin definition")
			Eventually(func() []byte {
				return tee.Contents()
			}).Should(ContainSubstring("plugin definition for service-proxy not found"),
				"ServiceProxyController should log about missing plugin definition")
		})
	})

	When("plugin definition for service proxy is present", func() {
		It("should create service-proxy plugin for organization", func() {
			By("creating service-proxy plugin definition")
			var serviceProxyPluginDefinition = &greenhousev1alpha1.PluginDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PluginDefinition",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-proxy",
					Namespace: corev1.NamespaceDefault,
				},
				Spec: greenhousev1alpha1.PluginDefinitionSpec{
					Description: "Testplugin",
					Version:     "1.0.0",
					HelmChart: &greenhousev1alpha1.HelmChartReference{
						Name:       "./../../test/fixtures/myChart",
						Repository: "dummy",
						Version:    "1.0.0",
					},
				},
			}
			err := setup.Create(test.Ctx, serviceProxyPluginDefinition)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the service-proxy plugin definition")

			By("creating an organization")
			org := setup.CreateOrganization(test.Ctx, "test-serviceproxy-org2")

			By("ensuring a service-proxy plugin has been created for organization")
			Eventually(func(g Gomega) {
				var plugin = new(greenhousev1alpha1.Plugin)
				err = setup.Get(test.Ctx, types.NamespacedName{Name: "service-proxy", Namespace: org.Name}, plugin)
				g.Expect(err).ToNot(HaveOccurred(), "service-proxy plugin should have been created by controller")
			}).Should(Succeed(), "service-proxy plugin should have been created for organization")
		})
	})
})
