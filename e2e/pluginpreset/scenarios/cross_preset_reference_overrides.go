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

// PluginPresetCrossPresetReferenceWithOverrides tests that when a consumer PluginPreset
// references a source PluginPreset, the source's ClusterOptionOverrides are applied
// before evaluating expressions. This ensures the consumer gets the per-cluster
// overridden value, not the default.
func PluginPresetCrossPresetReferenceWithOverrides(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {

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
	remoteCluster.Labels["app"] = "test-override-ref-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("creating source PluginPreset with default expression and cluster override")
	defaultExpressionStr := `"default-${global.greenhouse.clusterName}.example.com"`
	overrideExpressionStr := `"overridden-${global.greenhouse.clusterName}.special.com"`

	sourcePluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "override-ref-source",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:       optionServiceHostname,
				Expression: &defaultExpressionStr,
			},
		},
	}

	sourcePreset := test.NewPluginPreset("override-ref-source", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPresetPluginSpec(sourcePluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-override-ref-cluster"},
		}),
	)
	sourcePreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
		{
			ClusterName: remoteClusterName,
			Overrides: []greenhousev1alpha1.PluginPresetPluginOptionValue{
				{
					Name:       optionServiceHostname,
					Expression: &overrideExpressionStr,
				},
			},
		},
	}
	err = adminClient.Create(ctx, sourcePreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying source Plugin has the overridden value (not default)")
	expectedSourcePluginName := sourcePreset.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		sourcePlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedSourcePluginName, Namespace: env.TestNamespace}, sourcePlugin)
		g.Expect(err).NotTo(HaveOccurred())

		var found bool
		for _, ov := range sourcePlugin.Spec.OptionValues {
			if ov.Name == optionServiceHostname {
				found = true
				g.Expect(ov.Value).ToNot(BeNil())
				g.Expect(string(ov.Value.Raw)).To(Equal(`"overridden-`+remoteClusterName+`.special.com"`),
					"Source Plugin should have the overridden value, not the default")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "Source Plugin should have overridden expression resolved")

	By("waiting for source Plugin to be ready")
	Eventually(func(g Gomega) {
		sourcePlugin := &greenhousev1alpha1.Plugin{}
		g.Expect(adminClient.Get(ctx, client.ObjectKey{Name: expectedSourcePluginName, Namespace: env.TestNamespace}, sourcePlugin)).To(Succeed())
		g.Expect(sourcePlugin.Status.IsReadyTrue()).To(BeTrue())
	}).Should(Succeed())

	By("creating consumer PluginPreset that references source")
	consumerPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "override-ref-consumer",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name: "upstream.url",
				ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
					Ref: &greenhousev1alpha1.ExternalValueSource{
						Kind:       greenhousev1alpha1.PluginPresetKind,
						Name:       sourcePreset.Name,
						Expression: `"https://" + spec.optionValues.filter(v, v.name == 'service.hostname')[0].value`,
					},
				},
			},
		},
	}

	consumerPreset := test.NewPluginPreset("override-ref-consumer", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-override-ref-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying consumer Plugin gets the overridden value from source (not default)")
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
			if ov.Name == "upstream.url" {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil())
				g.Expect(string(ov.Value.Raw)).To(Equal(`"https://overridden-`+remoteClusterName+`.special.com"`),
					"Consumer should receive the overridden value from source, not the default")
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "Consumer Plugin should have resolved reference with source's overridden value")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)
	test.EventuallyDeleted(ctx, adminClient, sourcePreset)
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
