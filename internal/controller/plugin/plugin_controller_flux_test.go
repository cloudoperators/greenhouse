// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var (
	remoteKubeConfig []byte
	remoteEnvTest    *envtest.Environment
	remoteK8sClient  client.Client
)

var (
	testPluginTeam = test.NewTeam(test.Ctx, "test-remote-cluster-team", test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

	testCluster = test.NewCluster(test.Ctx, "test-flux-cluster", test.TestNamespace,
		test.WithAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testPluginTeam.Name))

	testClusterK8sSecret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flux-cluster",
			Namespace: test.TestNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: testPluginTeam.Name},
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
	}

	testPlugin = test.NewPlugin(test.Ctx, "test-flux-plugindefinition", test.TestNamespace,
		test.WithCluster("test-cluster"),
		test.WithPluginDefinition("test-flux-plugindefinition"),
		test.WithReleaseName("release-test-flux"),
		test.WithReleaseNamespace(test.TestNamespace),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testPluginTeam.Name),
		test.WithPluginOptionValue("flatOption", test.MustReturnJSONFor("flatValue"), nil),
		test.WithPluginOptionValue("nested.option", test.MustReturnJSONFor("nestedValue"), nil),
		test.WithPluginOptionValue("nested.secretOption", nil, &greenhousev1alpha1.ValueFromSource{
			Secret: &greenhousev1alpha1.SecretKeyReference{
				Name: "test-cluster",
				Key:  greenhouseapis.GreenHouseKubeConfigKey,
			},
		}),
	)
	testPluginDefinition = test.NewClusterPluginDefinition(
		test.Ctx,
		"test-flux-plugindefinition",
		test.AppendPluginOption(
			greenhousev1alpha1.PluginOption{
				Name:    "flatOptionDefault",
				Type:    greenhousev1alpha1.PluginOptionTypeString,
				Default: test.MustReturnJSONFor("flatDefault"),
			}),
		test.AppendPluginOption(
			greenhousev1alpha1.PluginOption{
				Name:    "nested.optionDefault",
				Type:    greenhousev1alpha1.PluginOptionTypeString,
				Default: test.MustReturnJSONFor("nestedDefault"),
			},
		),
	)
)

var _ = Describe("Flux Plugin Controller", Ordered, func() {
	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginDefinition")

		By("bootstrapping remote cluster")
		_, remoteK8sClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)

		By("creating a Team")
		Expect(test.K8sClient.Create(test.Ctx, testPluginTeam)).Should(Succeed(), "there should be no error creating the Team")

		By("creating a cluster")
		Expect(test.K8sClient.Create(test.Ctx, testCluster)).Should(Succeed(), "there should be no error creating the cluster resource")

		By("creating a secret with a valid kubeconfig for a remote cluster")
		testClusterK8sSecret.Data = map[string][]byte{
			greenhouseapis.KubeConfigKey: remoteKubeConfig,
		}
		Expect(test.K8sClient.Create(test.Ctx, &testClusterK8sSecret)).Should(Succeed())
	})

	AfterAll(func() {
		By("stopping the test environment")
		err := remoteEnvTest.Stop()
		Expect(err).
			NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
	})

	It("should compute the HelmRelease values for a Plugin", func() {
		expected := map[string]any{
			"flatOption":        "flatValue",
			"flatOptionDefault": "flatDefault",
			"nested": map[string]any{
				"option":        "nestedValue",
				"optionDefault": "nestedDefault",
			},
		}

		// compute the expected global.greenhouse values
		greenhouseValues, err := helm.GetGreenhouseValues(test.Ctx, test.K8sClient, *testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the greenhouse values")
		greenhouseValueMap, err := helm.ConvertFlatValuesToHelmValues(greenhouseValues)
		Expect(err).ToNot(HaveOccurred(), "there should be no error converting the greenhouse values to Helm values")
		expected["global"] = greenhouseValueMap["global"]
		expectedRaw, err := json.Marshal(expected)
		Expect(err).ToNot(HaveOccurred(), "the expected HelmRelease values should be valid JSON")

		By("computing the Values for a Plugin")
		actual, err := addValuesToHelmRelease(test.Ctx, test.K8sClient, testPlugin)
		Expect(err).ToNot(HaveOccurred(), "there should be no error computing the HelmRelease values for the Plugin")

		By("checking the computed Values")
		Expect(actual).To(Equal(expectedRaw), "the computed HelmRelease values should match the expected values")
	})
})
