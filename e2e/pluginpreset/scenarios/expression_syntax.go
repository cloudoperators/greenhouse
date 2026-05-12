// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

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

// PluginPresetExpressionSyntaxStyles tests that both the legacy 'object.*' syntax
// and the new simplified 'spec.*' syntax work in valueFrom.ref.expression.
// This ensures backward compatibility with existing expressions.
func PluginPresetExpressionSyntaxStyles(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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
	remoteCluster.Labels["app"] = "test-syntax-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	// === Source PluginPreset ===

	By("creating source PluginPreset with expression")
	sourceExpressionStr := `"syntax-test-${global.greenhouse.clusterName}"`
	sourcePluginSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "syntax-source",
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

	sourcePreset := test.NewPluginPreset("syntax-source-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(sourcePluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-syntax-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourcePreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("waiting for source Plugin to be ready")
	expectedSourcePluginName := sourcePreset.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		sourcePlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedSourcePluginName, Namespace: env.TestNamespace}, sourcePlugin)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sourcePlugin.Status.IsReadyTrue()).To(BeTrue(), "source Plugin should be ready")
	}).Should(Succeed())

	// === Test Style 1: New simplified syntax (spec.*) ===

	By("creating consumer with NEW simplified syntax: spec.optionValues...")
	consumerNewSyntaxSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "syntax-new",
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
						Name: sourcePreset.Name,
						// NEW style: spec.optionValues...
						Expression: `spec.optionValues.filter(v, v.name == "` + optionUIMessage + `")[0].value`,
					},
				},
			},
		},
	}

	consumerNewSyntax := test.NewPluginPreset("syntax-new-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerNewSyntaxSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-syntax-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerNewSyntax)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying NEW syntax consumer Plugin resolves correctly")
	expectedNewConsumerPluginName := consumerNewSyntax.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerNewSyntax.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1), "NEW syntax consumer should create one Plugin")

		consumerPlugin := &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedNewConsumerPluginName))

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"syntax-test-`+remoteClusterName+`"`),
					"NEW syntax should resolve correctly")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "NEW syntax consumer Plugin should have resolved value")

	// === Test Style 2: Legacy syntax (object.spec.*) ===

	By("creating consumer with LEGACY syntax: object.spec.optionValues...")
	consumerLegacySyntaxSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "syntax-legacy",
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
						Name: sourcePreset.Name,
						// LEGACY style: object.spec.optionValues...
						Expression: `object.spec.optionValues.filter(v, v.name == "` + optionUIMessage + `")[0].value`,
					},
				},
			},
		},
	}

	consumerLegacySyntax := test.NewPluginPreset("syntax-legacy-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerLegacySyntaxSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-syntax-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerLegacySyntax)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying LEGACY syntax consumer Plugin resolves correctly")
	expectedLegacyConsumerPluginName := consumerLegacySyntax.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerLegacySyntax.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1), "LEGACY syntax consumer should create one Plugin")

		consumerPlugin := &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedLegacyConsumerPluginName))

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"syntax-test-`+remoteClusterName+`"`),
					"LEGACY syntax should resolve correctly")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "LEGACY syntax consumer Plugin should have resolved value")

	// === Test Style 3: ${...} wrapper syntax ===

	By("creating consumer with WRAPPER syntax: ${spec.optionValues...}")
	consumerWrapperSyntaxSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "syntax-wrapper",
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
						Name: sourcePreset.Name,
						// WRAPPER style: ${spec.optionValues...}
						Expression: `${spec.optionValues.filter(v, v.name == "` + optionUIMessage + `")[0].value}`,
					},
				},
			},
		},
	}

	consumerWrapperSyntax := test.NewPluginPreset("syntax-wrapper-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerWrapperSyntaxSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-syntax-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerWrapperSyntax)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying WRAPPER syntax consumer Plugin resolves correctly")
	expectedWrapperConsumerPluginName := consumerWrapperSyntax.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerWrapperSyntax.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1), "WRAPPER syntax consumer should create one Plugin")

		consumerPlugin := &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedWrapperConsumerPluginName))

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"syntax-test-`+remoteClusterName+`"`),
					"WRAPPER syntax should resolve correctly")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "WRAPPER syntax consumer Plugin should have resolved value")

	// === Test Style 4: Legacy with ${...} wrapper ===

	By("creating consumer with LEGACY WRAPPER syntax: ${object.spec.optionValues...}")
	consumerLegacyWrapperSpec := greenhousev1alpha1.PluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "syntax-legacy-wrap",
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
						Name: sourcePreset.Name,
						// LEGACY WRAPPER style: ${object.spec.optionValues...}
						Expression: `${object.spec.optionValues.filter(v, v.name == "` + optionUIMessage + `")[0].value}`,
					},
				},
			},
		},
	}

	consumerLegacyWrapper := test.NewPluginPreset("syntax-legwrap-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(consumerLegacyWrapperSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-syntax-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerLegacyWrapper)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying LEGACY WRAPPER syntax consumer Plugin resolves correctly")
	expectedLegacyWrapperPluginName := consumerLegacyWrapper.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerLegacyWrapper.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1), "LEGACY WRAPPER syntax consumer should create one Plugin")

		consumerPlugin := &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedLegacyWrapperPluginName))

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil(), "Value should be set")
				g.Expect(string(ov.Value.Raw)).To(Equal(`"syntax-test-`+remoteClusterName+`"`),
					"LEGACY WRAPPER syntax should resolve correctly")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "LEGACY WRAPPER syntax consumer Plugin should have resolved value")

	By("cleaning up all consumer PluginPresets")
	test.EventuallyDeleted(ctx, adminClient, consumerNewSyntax)
	test.EventuallyDeleted(ctx, adminClient, consumerLegacySyntax)
	test.EventuallyDeleted(ctx, adminClient, consumerWrapperSyntax)
	test.EventuallyDeleted(ctx, adminClient, consumerLegacyWrapper)

	By("cleaning up source PluginPreset")
	test.EventuallyDeleted(ctx, adminClient, sourcePreset)

	By("deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
