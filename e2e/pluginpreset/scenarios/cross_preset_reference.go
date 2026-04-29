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

// PluginPresetCrossPresetReference tests that one PluginPreset can reference
// option values from another PluginPreset that uses CEL expressions.
// The source PluginPreset has an expression that gets resolved,
// and the consumer PluginPreset references the resolved value.
func PluginPresetCrossPresetReference(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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
	remoteCluster.Labels["app"] = "test-ref-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("creating source PluginPreset with CEL expression")
	sourceExpressionStr := `"generated-${global.greenhouse.clusterName}"`
	sourcePluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "ref-source",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:       optionUIMessage,
				Expression: &sourceExpressionStr,
			},
		},
	}

	sourcePreset := test.NewPluginPreset("ref-source-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(sourcePluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-ref-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourcePreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	expectedSourcePluginName := sourcePreset.Name + "-" + remoteClusterName

	By("verifying source Plugin is created with resolved expression")
	sourcePlugin := &greenhousev1alpha1.Plugin{}
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: sourcePreset.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1))
		sourcePlugin = &pluginList.Items[0]
		g.Expect(sourcePlugin.Name).To(Equal(expectedSourcePluginName))

		// Verify expression is resolved
		var found bool
		for _, ov := range sourcePlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.Expression).To(BeNil(), "Source expression should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Source value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"generated-`+remoteClusterName+`"`),
					"Source expression should resolve with cluster name")
			}
		}
		g.Expect(found).To(BeTrue(), "ui.message should exist in source Plugin")
	}).Should(Succeed(), "Source Plugin should have resolved expression")

	By("waiting for source Plugin to be ready")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKeyFromObject(sourcePlugin), sourcePlugin)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(sourcePlugin.Status.IsReadyTrue()).To(BeTrue(), "source Plugin should be ready")
	}).Should(Succeed())

	By("creating consumer PluginPreset that references source PluginPreset")
	consumerPluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "ref-consumer",
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
						Kind:       greenhousev1alpha1.PluginPresetKind,
						Name:       sourcePreset.Name,
						Expression: `${spec.optionValues.filter(v, v.name == optionUIMessage)[0].value}`,
					},
				},
			},
		},
	}

	consumerPreset := test.NewPluginPreset("ref-consumer-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-ref-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	expectedConsumerPluginName := consumerPreset.Name + "-" + remoteClusterName

	By("verifying consumer Plugin is created with resolved reference value")
	consumerPlugin := &greenhousev1alpha1.Plugin{}
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerPreset.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1))
		consumerPlugin = &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedConsumerPluginName))

		// Verify reference is resolved - no valueFrom should remain
		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved and removed from consumer Plugin")
				g.Expect(ov.Expression).To(BeNil(), "Expression should not exist in consumer Plugin")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set in consumer Plugin")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"generated-`+remoteClusterName+`"`),
					"Consumer value should match source's resolved expression value")
			}
		}
		g.Expect(found).To(BeTrue(), "ui.message should exist in consumer Plugin")
	}).Should(Succeed(), "Consumer Plugin should have resolved reference value")

	By("verifying both Plugins have matching values")
	Eventually(func(g Gomega) {
		// Get fresh copies
		err := adminClient.Get(ctx, client.ObjectKeyFromObject(sourcePlugin), sourcePlugin)
		g.Expect(err).ToNot(HaveOccurred())
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(consumerPlugin), consumerPlugin)
		g.Expect(err).ToNot(HaveOccurred())

		var sourceVal, consumerVal string
		for _, ov := range sourcePlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				sourceVal = string(ov.Value.Raw)
			}
		}
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				consumerVal = string(ov.Value.Raw)
			}
		}

		g.Expect(sourceVal).ToNot(BeEmpty(), "Source value should not be empty")
		g.Expect(consumerVal).ToNot(BeEmpty(), "Consumer value should not be empty")
		g.Expect(sourceVal).To(Equal(consumerVal),
			"Source and consumer Plugin ui.message values should match")
	}).Should(Succeed(), "Both Plugins should have matching resolved values")

	By("verifying consumer HelmRelease has resolved values")
	consumerHR := &helmv2.HelmRelease{}
	consumerHR.SetName(consumerPlugin.Name)
	consumerHR.SetNamespace(consumerPlugin.Namespace)
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(consumerHR), consumerHR)
		g.Expect(err).NotTo(HaveOccurred(), "Consumer HelmRelease should exist")

		var valuesMap map[string]any
		err = json.Unmarshal(consumerHR.Spec.Values.Raw, &valuesMap)
		g.Expect(err).NotTo(HaveOccurred())

		ui, ok := valuesMap["ui"].(map[string]any)
		g.Expect(ok).To(BeTrue(), "ui should exist in HelmRelease values")
		g.Expect(ui["message"]).To(Equal("generated-"+remoteClusterName),
			"ui.message in consumer HelmRelease should match resolved reference value")
	}).Should(Succeed(), "Consumer HelmRelease should contain resolved reference values")

	By("cleaning up consumer PluginPreset")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)

	By("cleaning up source PluginPreset")
	test.EventuallyDeleted(ctx, adminClient, sourcePreset)

	By("verifying all HelmReleases are deleted")
	Eventually(func(g Gomega) {
		sourceHR := &helmv2.HelmRelease{}
		sourceHR.SetName(expectedSourcePluginName)
		sourceHR.SetNamespace(env.TestNamespace)
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourceHR), sourceHR)
		g.Expect(err).To(HaveOccurred(), "Source HelmRelease should be deleted")

		err = adminClient.Get(ctx, client.ObjectKeyFromObject(consumerHR), consumerHR)
		g.Expect(err).To(HaveOccurred(), "Consumer HelmRelease should be deleted")
	}).Should(Succeed(), "All HelmReleases should be deleted")

	By("deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
