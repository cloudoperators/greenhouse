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

func PluginPresetExpressionEvaluation(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
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
	remoteCluster.Labels["app"] = "test-expr-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("creating PluginPreset with CEL expressions")
	expressionHostname := `"podinfo-${global.greenhouse.clusterName}.example.com"`
	expressionOrg := `"${global.greenhouse.organizationName}-service"`

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
				Name:       optionUIMessage,
				Expression: &expressionHostname,
			},
			{
				Name:       optionUIBackend,
				Expression: &expressionOrg,
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

	By("checking Plugin is created with resolved expression values")
	expectedPluginName := testPluginPreset.Name + "-" + remoteClusterName
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.MatchingLabels{greenhouseapis.LabelKeyPluginPreset: testPluginPreset.Name})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginList.Items).To(HaveLen(1))

		plugin := &pluginList.Items[0]
		g.Expect(plugin.Name).To(Equal(expectedPluginName))

		// Verify no expression fields remain
		for _, ov := range plugin.Spec.OptionValues {
			g.Expect(ov.Expression).To(BeNil(), "Plugin should not contain expression fields - option: "+ov.Name)
		}

		// Verify hostname resolved
		var hostnameFound bool
		for _, ov := range plugin.Spec.OptionValues {
			if ov.Name == optionUIMessage {
				hostnameFound = true
				g.Expect(ov.Value).ToNot(BeNil())
				g.Expect(string(ov.Value.Raw)).To(Equal(`"podinfo-` + remoteClusterName + `.example.com"`))
			}
		}
		g.Expect(hostnameFound).To(BeTrue())

		// Verify org expression resolved
		var orgFound bool
		for _, ov := range plugin.Spec.OptionValues {
			if ov.Name == optionUIBackend {
				orgFound = true
				g.Expect(ov.Value).ToNot(BeNil())
				g.Expect(string(ov.Value.Raw)).To(Equal(`"` + env.TestNamespace + `-service"`))
			}
		}
		g.Expect(orgFound).To(BeTrue())
	}).Should(Succeed(), "Plugin should be created with resolved expression values")

	By("cleaning up")
	test.EventuallyDeleted(ctx, adminClient, testPluginPreset)
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}
