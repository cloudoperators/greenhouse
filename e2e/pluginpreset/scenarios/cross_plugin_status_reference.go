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

// PluginPresetCrossPluginStatusReference covers an option merging its own value with a Plugin's status.
func PluginPresetCrossPluginStatusReference(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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
	remoteCluster.Labels["app"] = "test-plugin-status-ref-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	selectorLabel := "ref-status-group"
	selectorValue := "plugin-status-test"

	By("creating the source Plugin")
	sourcePlugin := &greenhousev1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-ref-plugin",
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
			ReleaseName:      "status-ref-plugin",
			ReleaseNamespace: env.TestNamespace,
			OptionValues: []greenhousev1alpha1.PluginOptionValue{
				{Name: optionReplicaCount, Value: test.MustReturnJSONFor("1")},
			},
		},
	}
	err = adminClient.Create(ctx, sourcePlugin)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("waiting for the source Plugin to be ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(sourcePlugin), sourcePlugin)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sourcePlugin.Status.IsReadyTrue()).To(BeTrue())
	}).Should(Succeed())

	By("creating a consumer PluginPreset reading the source Plugin's status next to a value of its own")
	consumerPluginSpec := greenhousev1alpha1.PluginPresetPluginSpec{
		PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
			Name: testPluginDefinition.Name,
		},
		ReleaseName:      "ref-plugin-status-consumer",
		ReleaseNamespace: env.TestNamespace,
		OptionValues: []greenhousev1alpha1.PluginPresetPluginOptionValue{
			{
				Name:  optionReplicaCount,
				Value: test.MustReturnJSONFor("1"),
			},
			{
				Name:  optionUIMessage,
				Value: test.MustReturnJSONFor([]string{"static-entry"}),
				ValueFrom: &greenhousev1alpha1.PluginPresetPluginValueFromSource{
					Ref: &greenhousev1alpha1.ExternalValueSource{
						Kind: greenhousev1alpha1.PluginKind,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{selectorLabel: selectorValue},
						},
						Expression: `${status.statusConditions.conditions.filter(c, c.type == 'Ready')[0].status}`,
					},
				},
			},
		},
	}
	consumerPreset := test.NewPluginPreset("ref-plugin-status-consumer-preset", env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPresetPluginSpec(consumerPluginSpec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "test-plugin-status-ref-cluster"},
		}),
	)
	err = adminClient.Create(ctx, consumerPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("verifying the consumer Plugin holds its own value first and the resolved status after it")
	expectedConsumerPluginName := consumerPreset.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		consumerPlugin := &greenhousev1alpha1.Plugin{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: expectedConsumerPluginName, Namespace: env.TestNamespace}, consumerPlugin)
		g.Expect(err).NotTo(HaveOccurred())

		var found bool
		for _, ov := range consumerPlugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				found = true
				g.Expect(ov.ValueFrom).To(BeNil(), "ValueFrom should be resolved")
				g.Expect(ov.Value).ToNot(BeNil())

				var merged []any
				g.Expect(json.Unmarshal(ov.Value.Raw, &merged)).To(Succeed())
				g.Expect(merged).To(Equal([]any{"static-entry", "True"}))
			}
		}
		g.Expect(found).To(BeTrue())
	}).Should(Succeed(), "Consumer Plugin should hold the static value followed by the referenced Plugin's Ready status")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, consumerPreset)
	test.EventuallyDeleted(ctx, adminClient, sourcePlugin)
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
