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
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// PluginPresetSelectorReference tests that a PluginPreset can reference
// multiple other PluginPresets using a label selector.
// Each source PluginPreset has a CEL expression that gets resolved,
// and the consumer collects all resolved values.
func PluginPresetSelectorReference(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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

	By("adding labels to remote cluster")
	remoteCluster := &greenhousev1alpha1.Cluster{}
	err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
	Expect(err).ToNot(HaveOccurred())
	if remoteCluster.Labels == nil {
		remoteCluster.Labels = make(map[string]string)
	}
	remoteCluster.Labels["app"] = "test-selector-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("creating first source PluginPreset with expression")
	sourceAExpressionStr := `"endpoint-a.${global.greenhouse.clusterName}.example.com"`
	sourceAPluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-src-a",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:       optionUIMessage,
				Expression: &sourceAExpressionStr,
			},
		},
	}

	sourceAPreset := test.NewPluginPreset("selector-source-a", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetLabel(selectorTestLabel, "true"),
		test.WithPluginPresetPluginSpec(sourceAPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourceAPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("creating second source PluginPreset with expression")
	sourceBExpressionStr := `"endpoint-b.${global.greenhouse.clusterName}.example.com"`
	sourceBPluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-src-b",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:       optionUIMessage,
				Expression: &sourceBExpressionStr,
			},
		},
	}

	sourceBPreset := test.NewPluginPreset("selector-source-b", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetLabel(selectorTestLabel, "true"),
		test.WithPluginPresetPluginSpec(sourceBPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourceBPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	expectedSourceAPluginName := sourceAPreset.Name + "-" + remoteClusterName
	expectedSourceBPluginName := sourceBPreset.Name + "-" + remoteClusterName

	By("verifying both source Plugins are created with resolved expressions")
	Eventually(func(g Gomega) {
		sourceAPlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedSourceAPluginName, Namespace: env.TestNamespace}, sourceAPlugin)
		g.Expect(err).NotTo(HaveOccurred(), "Source A Plugin should exist")
		var foundA bool
		for _, ov := range sourceAPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				foundA = true
				g.Expect(ov.Expression).To(BeNil(), "Source A expression should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Source A value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"endpoint-a.` + remoteClusterName + `.example.com"`))
			}
		}
		g.Expect(foundA).To(BeTrue(), "Source A should have ui.message")

		sourceBPlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedSourceBPluginName, Namespace: env.TestNamespace}, sourceBPlugin)
		g.Expect(err).NotTo(HaveOccurred(), "Source B Plugin should exist")
		var foundB bool
		for _, ov := range sourceBPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				foundB = true
				g.Expect(ov.Expression).To(BeNil(), "Source B expression should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Source B value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"endpoint-b.` + remoteClusterName + `.example.com"`))
			}
		}
		g.Expect(foundB).To(BeTrue(), "Source B should have ui.message")
	}).Should(Succeed(), "Both source Plugins should have resolved expressions")

	By("waiting for both source Plugins to be ready")
	Eventually(func(g Gomega) {
		sourceAPlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: expectedSourceAPluginName, Namespace: env.TestNamespace}, sourceAPlugin)).To(Succeed())
		g.Expect(sourceAPlugin.Status.IsReadyTrue()).To(BeTrue(), "Source A Plugin should be ready")

		sourceBPlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: expectedSourceBPluginName, Namespace: env.TestNamespace}, sourceBPlugin)).To(Succeed())
		g.Expect(sourceBPlugin.Status.IsReadyTrue()).To(BeTrue(), "Source B Plugin should be ready")
	}).Should(Succeed(), "Both source Plugins should be ready")

	By("creating consumer PluginPreset that references sources by selector")
	consumerPluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-consumer",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name: optionUIMessage,
				ValueFrom: &greenhousev1alpha1.PluginValueFromSource{
					Ref: &greenhousev1alpha1.ExternalValueSource{
						Kind: greenhousev1alpha1.PluginPresetKind,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								selectorTestLabel: "true",
							},
						},
						Expression: `spec.optionValues.filter(v, v.name == "ui.message")[0].value`,
					},
				},
			},
		},
	}

	consumerPreset := test.NewPluginPreset("selector-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	expectedConsumerPluginName := consumerPreset.Name + "-" + remoteClusterName

	By("verifying consumer Plugin has collected values from both sources via selector")
	Eventually(func(g Gomega) {
		consumerPlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedConsumerPluginName, Namespace: env.TestNamespace}, consumerPlugin)
		g.Expect(err).NotTo(HaveOccurred(), "Consumer Plugin should exist")

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Expression).To(BeNil(), "Expression should not exist")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")

				var endpoints []any
				err := json.Unmarshal(ov.Value.Raw, &endpoints)
				g.Expect(err).ToNot(HaveOccurred(), "Value should be valid JSON array")
				g.Expect(endpoints).To(HaveLen(2), "Should have values from both source PluginPresets")
				g.Expect(endpoints).To(ContainElement("endpoint-a."+remoteClusterName+".example.com"),
					"Should contain source A resolved value")
				g.Expect(endpoints).To(ContainElement("endpoint-b."+remoteClusterName+".example.com"),
					"Should contain source B resolved value")
			}
		}
		g.Expect(found).To(BeTrue(), "ui.message option should exist in consumer Plugin")
	}).Should(Succeed(), "Consumer Plugin should have collected values from both sources")

	By("verifying consumer Plugin is ready")
	Eventually(func(g Gomega) {
		consumerPlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: expectedConsumerPluginName, Namespace: env.TestNamespace}, consumerPlugin)).To(Succeed())
		g.Expect(consumerPlugin.Status.IsReadyTrue()).To(BeTrue(), "Consumer Plugin should be ready")
	}).Should(Succeed())

	By("verifying consumer HelmRelease has resolved values")
	consumerHR := &helmv2.HelmRelease{}
	consumerHR.SetName(expectedConsumerPluginName)
	consumerHR.SetNamespace(env.TestNamespace)
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(consumerHR), consumerHR)
		g.Expect(err).NotTo(HaveOccurred(), "Consumer HelmRelease should exist")

		var valuesMap map[string]any
		err = json.Unmarshal(consumerHR.Spec.Values.Raw, &valuesMap)
		g.Expect(err).NotTo(HaveOccurred())

		ui, ok := valuesMap["ui"].(map[string]any)
		g.Expect(ok).To(BeTrue(), "ui should exist in HelmRelease values")

		message := ui["message"]
		g.Expect(message).ToNot(BeNil(), "ui.message should exist in HelmRelease values")

		messageSlice, ok := message.([]any)
		g.Expect(ok).To(BeTrue(), "ui.message should be an array")
		g.Expect(messageSlice).To(HaveLen(2), "Should have 2 endpoints")
		g.Expect(messageSlice).To(ContainElement("endpoint-a." + remoteClusterName + ".example.com"))
		g.Expect(messageSlice).To(ContainElement("endpoint-b." + remoteClusterName + ".example.com"))
	}).Should(Succeed(), "Consumer HelmRelease should contain collected selector values")

	By("cleaning up consumer PluginPreset")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)

	By("cleaning up source PluginPresets")
	test.EventuallyDeleted(ctx, adminClient, sourceBPreset)
	test.EventuallyDeleted(ctx, adminClient, sourceAPreset)

	By("verifying all HelmReleases are deleted")
	Eventually(func(g Gomega) {
		sourceAHR := &helmv2.HelmRelease{}
		sourceAHR.SetName(expectedSourceAPluginName)
		sourceAHR.SetNamespace(env.TestNamespace)
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourceAHR), sourceAHR)
		g.Expect(err).To(HaveOccurred(), "Source A HelmRelease should be deleted")

		sourceBHR := &helmv2.HelmRelease{}
		sourceBHR.SetName(expectedSourceBPluginName)
		sourceBHR.SetNamespace(env.TestNamespace)
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourceBHR), sourceBHR)
		g.Expect(err).To(HaveOccurred(), "Source B HelmRelease should be deleted")

		err = adminClient.Get(ctx, client.ObjectKeyFromObject(consumerHR), consumerHR)
		g.Expect(err).To(HaveOccurred(), "Consumer HelmRelease should be deleted")
	}).Should(Succeed(), "All HelmReleases should be deleted")

	By("deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
