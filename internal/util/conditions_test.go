// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/internal/util"
)

var _ = Describe("Test common controllers utils", func() {
	Context("Test ComputeOwnerLabelCondition", Ordered, func() {
		const otherNamespace = "other-namespace"
		var (
			setup                   *test.TestSetup
			testSupportGroupTeam    *greenhousev1alpha1.Team
			testOtherNamespaceTeam  *greenhousev1alpha1.Team
			testNonSupportGroupTeam *greenhousev1alpha1.Team
		)
		BeforeAll(func() {
			setup = test.NewTestSetup(test.Ctx, test.K8sClient, "ownerlabel")
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
			It("should return True OwnerLabelSet condition for Cluster", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-correct-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionTrue), "OwnerLabelSet condition should be set to True")
			})
			It("should return True OwnerLabelSet condition for Secret", func() {
				secret := setup.CreateSecret(test.Ctx, "test-correct-secret",
					test.WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, secret)
				Expect(condition.Status).To(Equal(metav1.ConditionTrue), "OwnerLabelSet condition should be set to True")
			})
			It("should return True OwnerLabelSet condition for Plugin", func() {
				plugin := setup.CreatePlugin(test.Ctx, "test-correct-plugin",
					test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, plugin)
				Expect(condition.Status).To(Equal(metav1.ConditionTrue), "OwnerLabelSet condition should be set to True")
			})
			It("should return True OwnerLabelSet condition for PluginPreset", func() {
				plugin := setup.CreatePluginPreset(test.Ctx, "test-correct-preset",
					test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, plugin)
				Expect(condition.Status).To(Equal(metav1.ConditionTrue), "OwnerLabelSet condition should be set to True")
			})
			It("should return True OwnerLabelSet condition for TeamRoleBinding", func() {
				teamRoleBinding := setup.CreateTeamRoleBinding(test.Ctx, "test-correct-trb",
					test.WithTeamRoleBindingLabel(greenhouseapis.LabelKeyOwnedBy, testSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, teamRoleBinding)
				Expect(condition.Status).To(Equal(metav1.ConditionTrue), "OwnerLabelSet condition should be set to True")
			})
		})
		When("owner label is set to an existing Team in other namespace", func() {
			It("should return False OwnerLabelSet condition with proper reason and message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-other-team-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testOtherNamespaceTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionFalse), "OwnerLabelSet condition should be set to False")
				Expect(condition.Reason).To(Equal(greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason), "OwnerLabelSet condition should have a correct reason")
				Expect(condition.Message).To(Equal(fmt.Sprintf("team %s does not exist in resource namespace", testOtherNamespaceTeam.Name)), "OwnerLabelSet condition should have a correct message")
			})
		})
		When("owner label is set to an existing non-support-group Team", func() {
			It("should return False OwnerLabelSet condition with proper reason and message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-non-support-group-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testNonSupportGroupTeam.Name))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionFalse), "OwnerLabelSet condition should be set to False")
				Expect(condition.Reason).To(Equal(greenhousemetav1alpha1.OwnerLabelSetToNonSupportGroupTeamReason), "OwnerLabelSet condition should have a correct reason")
				Expect(condition.Message).To(Equal(fmt.Sprintf("owner team %s should be a support group", testNonSupportGroupTeam.Name)), "OwnerLabelSet condition should have a correct message")
			})
		})
		When("owner label is set to a non-existing Team", func() {
			It("should return False OwnerLabelSet condition with proper reason and message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-invalid-team-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, "not-existing-team"))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionFalse), "OwnerLabelSet condition should be set to False")
				Expect(condition.Reason).To(Equal(greenhousemetav1alpha1.OwnerLabelSetToNotExistingTeamReason), "OwnerLabelSet condition should have a correct reason")
				Expect(condition.Message).To(Equal(fmt.Sprintf("team %s does not exist in resource namespace", "not-existing-team")), "OwnerLabelSet condition should have a correct message")
			})
		})
		When("owner label value is missing", func() {
			It("should return False OwnerLabelSet condition with proper reason and message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-missing-owner-value-cluster",
					test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, ""))
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionFalse), "OwnerLabelSet condition should be set to False")
				Expect(condition.Reason).To(Equal(greenhousemetav1alpha1.OwnerLabelMissingReason), "OwnerLabelSet condition should have a correct reason")
				Expect(condition.Message).To(Equal(fmt.Sprintf("Label %s is missing", greenhouseapis.LabelKeyOwnedBy)), "OwnerLabelSet condition should have a correct message")
			})
		})
		When("owner label is missing", func() {
			It("should return False OwnerLabelSet condition with proper reason and message", func() {
				cluster := setup.CreateCluster(test.Ctx, "test-missing-owner-label-cluster")
				condition := util.ComputeOwnerLabelCondition(test.Ctx, test.K8sClient, cluster)
				Expect(condition.Status).To(Equal(metav1.ConditionFalse), "OwnerLabelSet condition should be set to False")
				Expect(condition.Reason).To(Equal(greenhousemetav1alpha1.OwnerLabelMissingReason), "OwnerLabelSet condition should have a correct reason")
				Expect(condition.Message).To(Equal(fmt.Sprintf("Label %s is missing", greenhouseapis.LabelKeyOwnedBy)), "OwnerLabelSet condition should have a correct message")
			})
		})
	})
})
