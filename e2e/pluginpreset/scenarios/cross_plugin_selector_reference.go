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

func PluginPresetCrossPluginSelectorReference(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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
	remoteCluster.Labels["app"] = "test-plugin-selector-ref-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	selectorLabel := "ref-group"
	selectorValue := "plugin-selector-test"

	By("creating first source Plugin")
	sourcePlugin1 := &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-ref-plugin-1",
			Namespace: env.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.LabelKeyOwnedBy: teamName,
				selectorLabel:                  selectorValue,
			},
		},
		Spec: greenhousev1alpha1.PluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: testPluginDefinition.Name,
			},
			ClusterName:      remoteClusterName,
			ReleaseName:      "selector-ref-plugin-1",
			ReleaseNamespace: env.TestNamespace,
			OptionValues: []greenhousev1alpha1.PluginOptionValue{
				{Name: optionReplicaCount, Value: test.MustReturnJSONFor("1")},
				{Name: optionUIMessage, Value: test.MustReturnJSONFor("value-from-plugin-1")},
			},
		},
	}
	err = adminClient.Create(ctx, sourcePlugin1)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("creating second source Plugin")
	sourcePlugin2 := &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-ref-plugin-2",
			Namespace: env.TestNamespace,
			Labels: map[string]string{
				greenhouseapis.LabelKeyOwnedBy: teamName,
				selectorLabel:                  selectorValue,
			},
		},
		Spec: greenhousev1alpha1.PluginSpec{
			PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
				Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
				Name: testPluginDefinition.Name,
			},
			ClusterName:      remoteClusterName,
			ReleaseName:      "selector-ref-plugin-2",
			ReleaseNamespace: env.TestNamespace,
			OptionValues: []greenhousev1alpha1.PluginOptionValue{
				{Name: optionReplicaCount, Value: test.MustReturnJSONFor("1")},
				{Name: optionUIMessage, Value: test.MustReturnJSONFor("value-from-plugin-2")},
			},
		},
	}
	err = adminClient.Create(ctx, sourcePlugin2)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("waiting for source Plugins to be ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourcePlugin1), sourcePlugin1)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sourcePlugin1.Status.IsReadyTrue()).To(BeTrue())
	}).Should(Succeed())

	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourcePlugin2), sourcePlugin2)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sourcePlugin2.Status.IsReadyTrue()).To(BeTrue())
	}).Should(Succeed())

	By("creating consumer PluginPreset that references Plugins by selector")
	consumerPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "ref-plugin-sel-consumer",
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
						Kind: greenhousev1alpha1.PluginKind,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{selectorLabel: selectorValue},
						},
						Expression: `${spec.optionValues.filter(v, v.name == 'ui.message')[0].value}`,
					},
				},
			},
		},
	}
	consumerPreset := test.NewPluginPreset("ref-plugin-sel-consumer-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-plugin-selector-ref-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying consumer Plugin has resolved reference values from both source Plugins")
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
				// The selector resolves multiple plugins, results are flattened into an array
				rawValue := string(ov.Value.Raw)
				g.Expect(rawValue).To(ContainSubstring("value-from-plugin-1"))
				g.Expect(rawValue).To(ContainSubstring("value-from-plugin-2"))
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "Consumer Plugin should have resolved reference values from both source Plugins")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)
	test.EventuallyDeleted(ctx, adminClient, sourcePlugin1)
	test.EventuallyDeleted(ctx, adminClient, sourcePlugin2)
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
