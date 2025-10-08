// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Test common GetPluginDefinitionSpec", func() {
	var setup *test.TestSetup
	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "plugindefinitionspec")
	})

	Context("GetPluginDefinitionSpec for Plugin", func() {
		var testPlugin *greenhousev1alpha1.Plugin
		var testTeam *greenhousev1alpha1.Team

		BeforeEach(func() {
			testTeam = setup.CreateTeam(test.Ctx, "test-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

			testPlugin = test.NewPlugin(test.Ctx, "test-plugindefinition", setup.Namespace(),
				test.WithCluster("test-cluster"),
				test.WithReleaseName("release-test"),
				test.WithReleaseNamespace(test.TestNamespace),
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			)
		})
		AfterEach(func() {
			test.EventuallyDeleted(test.Ctx, setup.Client, testPlugin)
			test.EventuallyDeleted(test.Ctx, setup.Client, testTeam)
		})

		When("ClusterPluginDefinition exists", func() {
			var clusterPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
			BeforeEach(func() {
				By("creating ClusterPluginDefinition")
				clusterPluginDefinition = setup.CreateClusterPluginDefinition(test.Ctx, "cluster-pd")
			})
			AfterEach(func() {
				test.EventuallyDeleted(test.Ctx, setup.Client, clusterPluginDefinition)
			})

			It("should return the correct Spec when ClusterPluginDefinition is referenced by Plugin", func() {
				By("creating Plugin with ClusterPluginDefinition reference")
				test.WithClusterPluginDefinition(clusterPluginDefinition.Name)(testPlugin)
				Expect(setup.Create(test.Ctx, testPlugin)).To(Succeed(), "failed to create test Plugin")

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).ToNot(HaveOccurred(), "expected no error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).ToNot(BeNil(), "returned PluginDefinitionSpec cannot be nil")
				Expect(*pluginDefinitionSpec).To(Equal(clusterPluginDefinition.Spec), "GetPluginDefinitionSpec should return the correct Spec")
			})

			It("should return an error when the PluginDefinitionKind is not specified in Plugin", func() {
				By("creating Plugin with ClusterPluginDefinition reference")
				testPlugin.Spec.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
					Name: clusterPluginDefinition.Name,
					Kind: "",
				}

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).To(HaveOccurred(), "expected error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).To(BeNil(), "returned PluginDefinitionSpec should be nil")
				Expect(err.Error()).To(ContainSubstring("PluginDefinitionRef.Kind has not been set"), "error should contain the correct message")
			})
		})

		When("PluginDefinition exists", func() {
			var pluginDefinition *greenhousev1alpha1.PluginDefinition
			BeforeEach(func() {
				By("creating PluginDefinition")
				pluginDefinition = test.NewPluginDefinition(test.Ctx, "namespaced-pd", setup.Namespace())
				Expect(setup.Create(test.Ctx, pluginDefinition)).To(Succeed(), "failed to create PluginDefinition")
			})
			AfterEach(func() {
				test.EventuallyDeleted(test.Ctx, setup.Client, pluginDefinition)
			})

			It("should return the correct Spec when PluginDefinition is referenced by Plugin", func() {
				By("creating Plugin with PluginDefinition reference")
				test.WithPluginDefinition(pluginDefinition.Name)(testPlugin)
				Expect(setup.Create(test.Ctx, testPlugin)).To(Succeed(), "failed to create test Plugin")

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).ToNot(HaveOccurred(), "expected no error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).ToNot(BeNil(), "returned PluginDefinitionSpec cannot be nil")
				Expect(*pluginDefinitionSpec).To(Equal(pluginDefinition.Spec), "GetPluginDefinitionSpec should return the correct Spec")
			})
		})

		When("PluginDefinition does not exist", func() {
			It("should return an error when PluginDefinitionKind is not supported", func() {
				By("creating Plugin with incorrect PluginDefinition reference")
				testPlugin.Spec.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
					Name: "non-existing-pd",
					Kind: "NotSupportedKind",
				}

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).To(HaveOccurred(), "expected an error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).To(BeNil(), "returned PluginDefinitionSpec should be nil")
				Expect(err.Error()).To(ContainSubstring("unsupported PluginDefinition kind: NotSupportedKind"), "error should contain the correct message")
			})

			It("should return an error when non-existing ClusterPluginDefinition is referenced", func() {
				By("creating Plugin with ClusterPluginDefinition reference")
				test.WithClusterPluginDefinition("non-existing-pd")(testPlugin)
				Expect(setup.Create(test.Ctx, testPlugin)).To(Succeed(), "failed to create test Plugin")

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).To(HaveOccurred(), "expected an error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).To(BeNil(), "returned PluginDefinitionSpec should be nil")
				Expect(err.Error()).To(ContainSubstring("ClusterPluginDefinition non-existing-pd does not exist"), "error should contain the correct message")
			})

			It("should return an error when non-existing PluginDefinition is referenced", func() {
				By("creating Plugin with PluginDefinition reference")
				test.WithPluginDefinition("non-existing-pd")(testPlugin)
				Expect(setup.Create(test.Ctx, testPlugin)).To(Succeed(), "failed to create test Plugin")

				By("checking GetPluginDefinitionSpec func outcome")
				pluginDefinitionSpec, err := common.GetPluginDefinitionSpec(test.Ctx, setup.Client,
					testPlugin.Spec.PluginDefinitionRef,
					testPlugin.GetNamespace(),
				)
				Expect(err).To(HaveOccurred(), "expected an error getting PluginDefinitionSpec for Plugin")
				Expect(pluginDefinitionSpec).To(BeNil(), "returned PluginDefinitionSpec should be nil")
				Expect(err.Error()).To(ContainSubstring("PluginDefinition non-existing-pd does not exist in namespace "+testPlugin.GetNamespace()), "error should contain the correct message")
			})
		})
	})
})
