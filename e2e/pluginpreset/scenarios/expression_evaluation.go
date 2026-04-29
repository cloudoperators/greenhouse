// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"encoding/json"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// PluginPresetExpressionEvaluation tests that CEL expressions in PluginPreset.spec.plugin.optionValues
// are evaluated during PluginPreset reconciliation and the resulting Plugin contains only resolved values.
func PluginPresetExpressionEvaluation(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	By("creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoClusterPluginDefinition(env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.IsReadyTrue()).To(BeTrue(), "the plugin definition should be ready")
	}).Should(Succeed())

	By("adding metadata labels to remote cluster")
	remoteCluster := &greenhousev1alpha1.Cluster{}
	err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
	Expect(err).ToNot(HaveOccurred())
	if remoteCluster.Labels == nil {
		remoteCluster.Labels = make(map[string]string)
	}
	remoteCluster.Labels["app"] = "test-expr-cluster"
	remoteCluster.Labels["metadata.greenhouse.sap/region"] = "eu-de-1"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("creating PluginPreset with CEL expressions")
	expressionHostname := `"podinfo-${global.greenhouse.clusterName}.example.com"`
	expressionRegion := `"service.${global.greenhouse.metadata.region}.example.com"`

	pluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "expr-test",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:       optionIngressHostname,
				Expression: &expressionHostname,
			},
			{
				Name:       optionUIMessage,
				Expression: &expressionRegion,
			},
		},
	}

	testPluginPreset := test.NewPluginPreset("expr-eval-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(pluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-expr-cluster"},
		}),
	)
	err = adminClient.Create(ctx, testPluginPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	expectedPluginName := testPluginPreset.Name + "-" + remoteClusterName

	By("checking the Plugin is created with resolved expression values")
	plugin := &greenhousev1alpha1.Plugin{}
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: testPluginPreset.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1), "there should be exactly one Plugin created by the PluginPreset")
		plugin = &pluginList.Items[0]
		g.Expect(plugin.Name).To(Equal(expectedPluginName))

		// Verify expressions are resolved - no expression fields should remain
		for _, ov := range plugin.Spec.OptionValues {
			g.Expect(ov.Expression).To(BeNil(), "Plugin should not contain any expression fields - option: "+ov.Name)
		}

		// Verify hostname expression was resolved with clusterName
		var hostnameFound bool
		for _, ov := range plugin.Spec.OptionValues {
			if ov.Name == optionIngressHostname {
				hostnameFound = true
				g.Expect(ov.Value).ToNot(BeNil(), "ingress.hostname value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"podinfo-`+remoteClusterName+`.example.com"`),
					"ingress.hostname should be resolved with cluster name")
			}
		}
		g.Expect(hostnameFound).To(BeTrue(), "ingress.hostname option should exist in Plugin")

		// Verify region expression was resolved with metadata
		var regionFound bool
		for _, ov := range plugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				regionFound = true
				g.Expect(ov.Value).ToNot(BeNil(), "ui.message value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"service.eu-de-1.example.com"`),
					"ui.message should be resolved with cluster metadata region")
			}
		}
		g.Expect(regionFound).To(BeTrue(), "ui.message option should exist in Plugin")
	}).Should(Succeed(), "Plugin should be created with resolved expression values")

	By("checking the Plugin is ready")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKeyFromObject(plugin), plugin)
		g.Expect(err).ToNot(HaveOccurred())

		ready := plugin.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		g.Expect(ready).ToNot(BeNil(), "Plugin Ready condition must be set")
		g.Expect(ready.Status).To(Equal(metav1.ConditionTrue), "Plugin should be Ready")
	}).Should(Succeed(), "Plugin should eventually be ready")

	By("verifying resolved values in HelmRelease")
	hr := &helmv2.HelmRelease{}
	hr.SetName(plugin.Name)
	hr.SetNamespace(plugin.Namespace)
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(hr), hr)
		g.Expect(err).NotTo(HaveOccurred(), "HelmRelease should exist")

		var valuesMap map[string]any
		err = json.Unmarshal(hr.Spec.Values.Raw, &valuesMap)
		g.Expect(err).NotTo(HaveOccurred(), "HelmRelease values should be valid JSON")

		// Check ingress.hostname is resolved in HelmRelease values
		ingress, ok := valuesMap["ingress"].(map[string]any)
		g.Expect(ok).To(BeTrue(), "ingress should exist in HelmRelease values")
		g.Expect(ingress["hostname"]).To(Equal("podinfo-"+remoteClusterName+".example.com"),
			"ingress.hostname in HelmRelease should match resolved expression")

		// Check ui.message is resolved in HelmRelease values
		ui, ok := valuesMap["ui"].(map[string]any)
		g.Expect(ok).To(BeTrue(), "ui should exist in HelmRelease values")
		g.Expect(ui["message"]).To(Equal("service.eu-de-1.example.com"),
			"ui.message in HelmRelease should match resolved expression")
	}).Should(Succeed(), "HelmRelease should contain resolved expression values")

	By("verifying PluginPreset status is healthy")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(testPluginPreset.Status.TotalPlugins).To(Equal(1))
		g.Expect(testPluginPreset.Status.PluginStatuses).To(HaveLen(1))
		g.Expect(testPluginPreset.Status.PluginStatuses[0].PluginName).To(Equal(expectedPluginName))
	}).Should(Succeed(), "PluginPreset status should be healthy")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, testPluginPreset)

	By("verifying HelmRelease is deleted")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(hr), hr)
		g.Expect(err).To(HaveOccurred(), "HelmRelease should be deleted")
	}).Should(Succeed())

	By("deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
