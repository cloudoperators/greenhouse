// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Validate Team Creation", func() {

	teamStub := *test.NewTeam(test.Ctx, "", "test-org", test.WithMappedIDPGroup("IDP_GROUP_NAME_MATCHING_TEAM"))

	BeforeEach(func() {
		pluginDefinition := test.NewClusterPluginDefinition(test.Ctx, "test-plugindefinition-1")
		err := test.K8sClient.Create(test.Ctx, pluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a pluginDefinition")

		pluginDefinition2 := test.NewClusterPluginDefinition(test.Ctx, "test-plugindefinition-2")
		err = test.K8sClient.Create(test.Ctx, pluginDefinition2)
		Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a pluginDefinition")
	})

	It("should correctly validate a team upon creation", func() {
		teamNoGreenhouseLabels := teamStub
		teamNoGreenhouseLabels.SetName("non-greenhouse-labels")

		teamGreenhouseLabels := teamStub
		teamGreenhouseLabels.SetName("greenhouse-labels")

		teamFaultyGreenhouseLabels := teamStub
		teamFaultyGreenhouseLabels.SetName("faulty-greenhouse-labels")

		teamValidJoinURL := teamStub
		teamValidJoinURL.SetName("valid-joinurl")

		teamInvalidJoinURL := teamStub
		teamInvalidJoinURL.SetName("invalid-joinurl")

		By("correctly allowing creation of a team with non-greenhouse labels", func() {
			teamNoGreenhouseLabels.SetLabels(map[string]string{
				"some-key": "some-value",
			})
			err := test.K8sClient.Create(test.Ctx, &teamNoGreenhouseLabels)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team with non-greenhouse labels")
		})

		By("correctly allowing creation of a team with greenhouse labels that use whitelabeled labels and/or existing pluginDefinition names", func() {
			teamGreenhouseLabels.SetLabels(map[string]string{
				"greenhouse.sap/test-plugindefinition-1": "true",
				"greenhouse.sap/support-group":           "true",
			})
			err := test.K8sClient.Create(test.Ctx, &teamGreenhouseLabels)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team with greenhouse labels that use existing pluginDefinition names")
		})

		By("correctly denying creation of a team with greenhouse labels that use non-existing pluginDefinition names", func() {
			teamFaultyGreenhouseLabels.SetLabels(map[string]string{
				"greenhouse.sap/test-plugindefinition-3": "true",
			})
			err := test.K8sClient.Create(test.Ctx, &teamFaultyGreenhouseLabels)
			Expect(err).To(HaveOccurred(), "There should be an error when creating a team with greenhouse labels that use non-existing pluginDefinition names")
			Expect(err.Error()).To(ContainSubstring("Only pluginDefinition names as greenhouse labels allowed."))
		})

		By("correctly allowing update of a team with non-greenhouse labels", func() {
			teamNoGreenhouseLabels.SetLabels(map[string]string{
				"some-key":       "some-value",
				"some-other-key": "some-other-value",
			})
			err := test.K8sClient.Update(test.Ctx, &teamNoGreenhouseLabels)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when updating a team with non-greenhouse labels")
		})

		By("correctly allowing update of a team with greenhouse labels that use whitelisted labels and/or existing pluginDefinition names", func() {
			teamGreenhouseLabels.SetLabels(map[string]string{
				"greenhouse.sap/test-plugindefinition-1": "true",
				"greenhouse.sap/test-plugindefinition-2": "true",
				"greenhouse.sap/support-group":           "true",
			})
			err := test.K8sClient.Update(test.Ctx, &teamGreenhouseLabels)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when updating a team with greenhouse labels that use existing pluginDefinition names")
		})

		By("correctly denying update of a team with greenhouse labels that use non-existing pluginDefinition names", func() {
			teamGreenhouseLabels.SetLabels(map[string]string{
				"greenhouse.sap/test-plugindefinition-3": "true",
			})
			err := test.K8sClient.Update(test.Ctx, &teamGreenhouseLabels)
			Expect(err).To(HaveOccurred(), "There should be an error when updating a team with greenhouse labels that use non-existing pluginDefinition names")
			Expect(err.Error()).To(ContainSubstring("Only pluginDefinition names as greenhouse labels allowed."))
		})

		By("correctly allowing create of a team with valid JoinURL", func() {
			teamValidJoinURL.Spec.JoinURL = "https://example.com/resource"
			err := test.K8sClient.Create(test.Ctx, &teamValidJoinURL)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when creating a team with valid JoinURL")
		})
		By("correctly denying create of a team with invalid JoinURL", func() {
			teamInvalidJoinURL.Spec.JoinURL = "testvalue"
			err := test.K8sClient.Create(test.Ctx, &teamInvalidJoinURL)
			Expect(err).To(HaveOccurred(), "There should be an error when creating a team with invalid JoinURL")
			Expect(err.Error()).To(ContainSubstring("JoinURL must be a valid 'http:' or 'https:' URL, like 'https://example.com'."))
		})
		By("correctly allowing update of a team with valid JoinURL", func() {
			teamValidJoinURL.Spec.JoinURL = "https://1.1.1.1:80"
			err := test.K8sClient.Update(test.Ctx, &teamValidJoinURL)
			Expect(err).ToNot(HaveOccurred(), "There should be no error when updating a team with valid JoinURL")
		})
		By("correctly denying update of a team with invalid JoinURL", func() {
			teamValidJoinURL.Spec.JoinURL = "/example/1"
			err := test.K8sClient.Update(test.Ctx, &teamValidJoinURL)
			Expect(err).To(HaveOccurred(), "There should be an error when updating a team with invalid JoinURL")
			Expect(err.Error()).To(ContainSubstring("JoinURL must be a valid 'http:' or 'https:' URL, like 'https://example.com'."))
		})
	})
})
