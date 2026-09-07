// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"encoding/json"

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

func PluginPresetSelectorReference(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	By("creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoClusterPluginDefinition(env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.IsReadyTrue()).To(BeTrue())
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
	sourceAPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-src-a",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
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
		test.WithPresetPluginSpec(sourceAPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourceAPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("creating second source PluginPreset with expression")
	sourceBExpressionStr := `"endpoint-b.${global.greenhouse.clusterName}.example.com"`
	sourceBPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-src-b",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
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
		test.WithPresetPluginSpec(sourceBPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, sourceBPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("waiting for both source Plugins to be ready")
	Eventually(func(g Gomega) {
		sourceAPlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: sourceAPreset.Name + "-" + remoteClusterName, Namespace: env.TestNamespace}, sourceAPlugin)).To(Succeed())
		g.Expect(sourceAPlugin.Status.IsReadyTrue()).To(BeTrue())

		sourceBPlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: sourceBPreset.Name + "-" + remoteClusterName, Namespace: env.TestNamespace}, sourceBPlugin)).To(Succeed())
		g.Expect(sourceBPlugin.Status.IsReadyTrue()).To(BeTrue())
	}).Should(Succeed())

	By("creating consumer PluginPreset that references sources by selector")
	consumerPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "selector-consumer",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name: optionUIMessage,
				ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
					Ref: &greenhousev1alpha1.ExternalValueSource{
						Kind: greenhousev1alpha1.PluginPresetKind,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								selectorTestLabel: "true",
							},
						},
						Expression: `spec.optionValues.filter(v, v.name == 'ui.message')[0].value`,
					},
				},
			},
		},
	}

	consumerPreset := test.NewPluginPreset("selector-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-selector-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying consumer Plugin has collected values from both sources")
	expectedConsumerPluginName := consumerPreset.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: consumerPreset.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1))

		consumerPlugin := &pluginList.Items[0]
		g.Expect(consumerPlugin.Name).To(Equal(expectedConsumerPluginName))

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil())

				var endpoints []any
				err := json.Unmarshal(ov.Value.Raw, &endpoints)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(endpoints).To(HaveLen(2))
				g.Expect(endpoints).To(ContainElement("endpoint-a." + remoteClusterName + ".example.com"))
				g.Expect(endpoints).To(ContainElement("endpoint-b." + remoteClusterName + ".example.com"))
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "Consumer should have collected values from both sources")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)
	test.EventuallyDeleted(ctx, adminClient, sourceBPreset)
	test.EventuallyDeleted(ctx, adminClient, sourceAPreset)
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
