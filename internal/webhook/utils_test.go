// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package webhook_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/webhook"
)

var _ = Describe("Test webhook utils", func() {
	Context("Test ValidateLabelOwnedBy", Ordered, func() {
		const otherNamespace = "other-namespace"
		var (
			setup                   *test.TestSetup
			testSupportGroupTeam    *greenhousev1alpha1.Team
			testOtherNamespaceTeam  *greenhousev1alpha1.Team
			testNonSupportGroupTeam *greenhousev1alpha1.Team
		)
		BeforeAll(func() {
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, "webhookutils")
			testSupportGroupTeam = setup.CreateTeam(test.Ctx, "test-support-group-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			var namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otherNamespace}}
			Expect(setup.Create(test.Ctx, namespace)).To(Succeed(), "there should be no error creating a namespace")
			testOtherNamespaceTeam = test.NewTeam(test.Ctx, "test-other-namespace-team", otherNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			Expect(setup.Create(test.Ctx, testOtherNamespaceTeam)).To(Succeed(), "there should be no error creating the other Team")
			testNonSupportGroupTeam = setup.CreateTeam(test.Ctx, "test-non-support-group-team")
		})
		AfterAll(func() {
			test.EventuallyDeleted(test.Ctx, setup.Client, testOtherNamespaceTeam)
			test.EventuallyDeleted(test.Ctx, setup.Client, testSupportGroupTeam)
			test.EventuallyDeleted(test.Ctx, setup.Client, testNonSupportGroupTeam)
		})
		When("owner label is set to an existing support-group Team in the same namespace", func() {
			It("should return no warning for Cluster", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-correct-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(BeEmpty(), "warning message should be empty")
			})
			It("should return no warning for Secret", func() {
				secret := setup.CreateSecret(test.Ctx, "test-correct-secret",
					test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, secret)
				Expect(warning).To(BeEmpty(), "warning message should be empty")
			})
			It("should return no warning for Plugin", func() {
				plugin := setup.CreatePlugin(test.Ctx, "test-correct-plugin",
					test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, plugin)
				Expect(warning).To(BeEmpty(), "warning message should be empty")
			})
			It("should return no warning for PluginPreset", func() {
				plugin := setup.CreatePluginPreset(test.Ctx, "test-correct-preset",
					test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, plugin)
				Expect(warning).To(BeEmpty(), "warning message should be empty")
			})
			It("should return no warning for TeamRoleBinding", func() {
				teamRoleBinding := setup.CreateTeamRoleBinding(test.Ctx, "test-correct-trb",
					test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, teamRoleBinding)
				Expect(warning).To(BeEmpty(), "warning message should be empty")
			})
		})
		When("owner label is set to an existing Team in other namespace", func() {
			It("should return a proper warning message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-other-team-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testOtherNamespaceTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(ContainSubstring(fmt.Sprintf("team %s does not exist in the resource namespace", testOtherNamespaceTeam.Name)), "warning should have a correct message")
			})
		})
		When("owner label is set to an existing non-support-group Team", func() {
			It("should return a proper warning message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-non-support-group-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testNonSupportGroupTeam.Name))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(ContainSubstring(fmt.Sprintf("owner team %s should be a support group", testNonSupportGroupTeam.Name)), "warning should have a correct message")
			})
		})
		When("owner label is set to a non-existing Team", func() {
			It("should return a proper warning message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-invalid-team-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, "not-existing-team"))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(ContainSubstring(fmt.Sprintf("team %s does not exist in the resource namespace", "not-existing-team")), "warning should have a correct message")
			})
		})
		When("owner label value is missing", func() {
			It("should return a proper warning message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-missing-owner-value-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, ""))
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(ContainSubstring(fmt.Sprintf("label %s value is required", greenhouseapis.LabelKeyOwnedBy)), "warning should have a correct message")
			})
		})
		When("owner label is missing", func() {
			It("should return a proper warning message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-missing-owner-label-cluster")
				warning := webhook.ValidateLabelOwnedBy(test.Ctx, test.K8sClient, cluster)
				Expect(warning).To(ContainSubstring(fmt.Sprintf("label %s is required", greenhouseapis.LabelKeyOwnedBy)), "warning should have a correct message")
			})
		})
	})
})
