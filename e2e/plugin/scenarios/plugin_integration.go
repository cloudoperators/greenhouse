// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"
	"encoding/json"
	"math/rand"
	"slices"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	multiRefPluginLabelKey     = "e2e.greenhouse.sap/multi-ref-plugin"
	selectorRefPluginA         = "selector-ref-plugin-a"
	selectorRefPluginB         = "selector-ref-plugin-b"
	selectorResolverPluginName = "selector-resolver-plugin"
	directResolverPluginName   = "direct-resolver-plugin"
	directReferencePluginName  = "direct-reference-plugin"
)

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func generateRandomEnvs(prefix string) (*apiextensionsv1.JSON, error) {
	envVars := []map[string]string{
		{
			"name":  prefix + "_LOG_LEVEL",
			"value": randomLogLevel(),
		},
		{
			"name":  prefix + "_REGION",
			"value": randomRegion(),
		},
		{
			"name":  prefix + "_SESSION_ID",
			"value": randomSessionID(),
		},
		{
			"name":  prefix + "_MODE",
			"value": randomMode(),
		},
	}

	jsonBytes, err := json.Marshal(envVars)
	if err != nil {
		return nil, err
	}

	return &apiextensionsv1.JSON{Raw: jsonBytes}, nil
}

// randomLogLevel returns a random zap log level
func randomLogLevel() string {
	logLevels := []string{
		"debug",
		"info",
		"warn",
		"error",
		"dpanic",
		"panic",
		"fatal",
	}
	return logLevels[rng.Intn(len(logLevels))]
}

// randomRegion returns a random AWS-like region
func randomRegion() string {
	regions := []string{
		"eu-central-1",
		"eu-west-1",
		"us-east-1",
		"us-west-2",
		"ap-southeast-1",
		"ap-northeast-1",
	}
	return regions[rng.Intn(len(regions))]
}

// randomSessionID generates a random session ID
func randomSessionID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// randomMode returns a random deployment mode
func randomMode() string {
	modes := []string{
		"staging",
		"production",
		"development",
		"testing",
	}
	return modes[rng.Intn(len(modes))]
}

// containsExpectedEnvs checks if all environment variables from rawExtraEnvs are present in the envVars slice
// The envVars may have additional environment variables that we don't care about
func containsExpectedEnvs(envVars []corev1.EnvVar, rawExtraEnvs interface{}) bool {
	// Parse rawExtraEnvs into a slice of maps
	rawEnvsBytes, err := json.Marshal(rawExtraEnvs)
	if err != nil {
		return false
	}
	var expectedEnvs []map[string]string
	if err := json.Unmarshal(rawEnvsBytes, &expectedEnvs); err != nil {
		return false
	}
	// Check that each expected env var is present in the container with the correct value
	for _, expectedEnv := range expectedEnvs {
		expectedName := expectedEnv["name"]
		expectedValue := expectedEnv["value"]
		if !slices.ContainsFunc(envVars, func(envVar corev1.EnvVar) bool {
			return envVar.Name == expectedName && envVar.Value == expectedValue
		}) {
			return false
		}
	}
	return true
}

func generatePlugin(name, namespace string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	plugin := &greenhousev1alpha1.Plugin{}
	plugin.SetName(name)
	plugin.SetNamespace(namespace)
	for _, o := range opts {
		o(plugin)
	}
	return plugin
}

func PluginIntegrationByDirectReference(ctx context.Context, adminClient client.Client, remoteClient client.Client, env *shared.TestEnv, remoteClusterName string) {
	By("creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoPluginDefinition("podinfo", env.TestNamespace, "6.9.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.IsReadyTrue()).To(BeTrue(), "the plugin definition should be ready")
	}).Should(Succeed())

	By("creating reference plugin")
	directRefEnvs, err := generateRandomEnvs("DIRECT")
	Expect(err).ToNot(HaveOccurred(), "there should be no error generating random envs for plugin A")
	pluginDirectRef := &greenhousev1alpha1.Plugin{}
	pluginDirectRef.SetName(directReferencePluginName)
	pluginDirectRef.SetNamespace(env.TestNamespace)
	_, err = controllerutil.CreateOrPatch(ctx, adminClient, pluginDirectRef, func() error {
		pluginDirectRef.Spec = generatePlugin(
			directReferencePluginName,
			env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
			test.WithPluginOptionValue("extraEnvs", directRefEnvs),
			test.WithReleaseName(directReferencePluginName+"-release"),
			test.WithReleaseNamespace(directReferencePluginName+"-namespace"),
			test.WithCluster(remoteClusterName),
		).Spec
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the reference plugin")

	By("checking the reference plugin is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(pluginDirectRef), pluginDirectRef)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pluginDirectRef.Status.IsReadyTrue()).To(BeTrue(), "the reference plugin should be ready")
	}).Should(Succeed(), "there should be no error in reference plugin readiness")

	By("creating resolver plugin")
	resolverPlugin := &greenhousev1alpha1.Plugin{}
	resolverPlugin.SetName("resolver-plugin")
	resolverPlugin.SetNamespace(env.TestNamespace)
	_, err = controllerutil.CreateOrPatch(ctx, adminClient, resolverPlugin, func() error {
		resolverPlugin.Spec = generatePlugin(
			directResolverPluginName,
			env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
			test.WithReleaseName(directResolverPluginName+"-release"),
			test.WithReleaseNamespace(directResolverPluginName+"-namespace"),
			test.WithCluster(remoteClusterName),
			test.WithPluginOptionValueFromRef("extraEnvs", &greenhousev1alpha1.ExternalValueSource{
				Name:       pluginDirectRef.Name,
				Expression: "object.spec.optionValues.filter(o, o.name == 'extraEnvs')[0].value",
			}),
		).Spec
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the resolver plugin")

	By("checking the resolver plugin is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(resolverPlugin), resolverPlugin)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the resolver plugin")
		g.Expect(resolverPlugin.Status.IsReadyTrue()).To(BeTrue(), "the resolver plugin should be ready")
	}).Should(Succeed(), "the resolver plugin should be ready")

	By("verifying tracking-id annotation is set on the referenced plugin")
	Eventually(func(g Gomega) {
		g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(pluginDirectRef), pluginDirectRef)).To(Succeed(), "there should be no error getting the reference plugin to check annotations")
		annotations := pluginDirectRef.GetAnnotations()
		g.Expect(annotations).NotTo(BeNil(), "there should be annotations on the plugin A")
		g.Expect(annotations[greenhouseapis.AnnotationKeyPluginTackingID]).To(Equal("Plugin/"+resolverPlugin.Name), "the tracking ID annotation on plugin A should match the resolver plugin name")
	}).Should(Succeed(), "the reference plugin should have the tracking ID annotation set by resolver plugin")

	By("verifying the resolver extraEnvs values in flux HelmRelease")
	var extraEnvsFromHR any
	hr := &helmv2.HelmRelease{}
	hr.SetName(resolverPlugin.Name)
	hr.SetNamespace(resolverPlugin.Namespace)
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(hr), hr)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the HelmRelease for the resolver plugin")

		var valuesMap map[string]any
		err = json.Unmarshal(hr.Spec.Values.Raw, &valuesMap)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error unmarshalling the HelmRelease values")

		extraEnvsFromHR = valuesMap["extraEnvs"]
		extraEnvsBytes, err := json.Marshal(extraEnvsFromHR)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error marshalling the raw extraEnvs from the HelmRelease values")
		resolverExtraEnvs := &apiextensionsv1.JSON{Raw: extraEnvsBytes}
		g.Expect(resolverExtraEnvs).To(Equal(directRefEnvs), "the extraEnvs in the HelmRelease should match the ones from plugin A")
	}).Should(Succeed(), "the HelmRelease for the resolver plugin should have the expected extraEnvs values")

	// TODO: can checking the remote deployment be skipped since we already verified the values in the HelmRelease?
	By("verifying the envs in the remote cluster deployment")
	deployment := &appsv1.Deployment{}
	deployment.SetName(resolverPlugin.Spec.ReleaseName + "-podinfo")
	deployment.SetNamespace(resolverPlugin.Spec.ReleaseNamespace)
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the deployment from the remote cluster")
		envVars := deployment.Spec.Template.Spec.Containers[0].Env
		// Verify that the expected envs from extraEnvsFromHR are present in the deployment
		g.Expect(containsExpectedEnvs(envVars, extraEnvsFromHR)).To(BeTrue(), "the deployment should contain all expected environment variables from rawExtraEnvs")
	}).Should(Succeed(), "the deployment should be present in the remote cluster with expected envs")
}

func PluginIntegrationBySelector(ctx context.Context, adminClient client.Client, remoteClient client.Client, env *shared.TestEnv, remoteClusterName string) {
	By("creating plugin definition")
	testPluginDefinition := fixtures.PreparePodInfoPluginDefinition("podinfo-latest", env.TestNamespace, "6.11.0")
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.IsReadyTrue()).To(BeTrue(), "the plugin definition should be ready")
	}).Should(Succeed())

	By("creating reference plugins")
	pluginAEnvs, err := generateRandomEnvs("A")
	Expect(err).ToNot(HaveOccurred(), "there should be no error generating random envs for plugin A")
	pluginA := &greenhousev1alpha1.Plugin{}
	pluginA.SetName(selectorRefPluginA)
	pluginA.SetNamespace(env.TestNamespace)
	_, err = controllerutil.CreateOrPatch(ctx, adminClient, pluginA, func() error {
		pluginA.SetLabels(map[string]string{
			multiRefPluginLabelKey: "true",
		})
		pluginA.Spec = generatePlugin(
			selectorRefPluginA,
			env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
			test.WithPluginOptionValue("extraEnvs", pluginAEnvs),
			test.WithReleaseName(selectorRefPluginA+"-release"),
			test.WithReleaseNamespace(selectorRefPluginA+"-namespace"),
			test.WithCluster(remoteClusterName),
		).Spec
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin "+selectorRefPluginA)
	pluginBEnvs, err := generateRandomEnvs("B")
	Expect(err).ToNot(HaveOccurred(), "there should be no error generating random envs for plugin B")
	pluginB := &greenhousev1alpha1.Plugin{}
	pluginB.SetName(selectorRefPluginB)
	pluginB.SetNamespace(env.TestNamespace)
	_, err = controllerutil.CreateOrPatch(ctx, adminClient, pluginB, func() error {
		pluginB.SetLabels(map[string]string{
			multiRefPluginLabelKey: "true",
		})
		pluginB.Spec = generatePlugin(
			selectorRefPluginB,
			env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
			test.WithPluginOptionValue("extraEnvs", pluginBEnvs),
			test.WithReleaseName(selectorRefPluginB+"-release"),
			test.WithReleaseNamespace(selectorRefPluginB+"-namespace"),
			test.WithCluster(remoteClusterName),
		).Spec
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the plugin "+selectorRefPluginB)

	By("creating resolver plugin with selector reference to the plugins")
	resolverPlugin := &greenhousev1alpha1.Plugin{}
	resolverPlugin.SetName(selectorResolverPluginName)
	resolverPlugin.SetNamespace(env.TestNamespace)
	_, err = controllerutil.CreateOrPatch(ctx, adminClient, resolverPlugin, func() error {
		resolverPlugin.Spec = generatePlugin(
			selectorResolverPluginName,
			env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
			test.WithReleaseName(selectorResolverPluginName+"-release"),
			test.WithReleaseNamespace(selectorResolverPluginName+"-namespace"),
			test.WithCluster(remoteClusterName),
			test.WithPluginOptionValueFromRef("extraEnvs", &greenhousev1alpha1.ExternalValueSource{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						multiRefPluginLabelKey: "true",
					},
				},
				Expression: "object.spec.optionValues.filter(o, o.name == 'extraEnvs')[0].value",
			}),
		).Spec
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the resolver plugin with selector reference")

	By("checking the resolver plugin is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(resolverPlugin), resolverPlugin)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the resolver plugin")
		g.Expect(resolverPlugin.Status.IsReadyTrue()).To(BeTrue(), "the resolver plugin should be ready")
	}).Should(Succeed(), "the resolver plugin should be ready")

	// TODO: check tracking ID annotation similar to how the controller does it by exporting the helper functions
	By("checking the reference plugins are ready and have the tracking ID annotation set")
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(ctx, pluginList, client.InNamespace(env.TestNamespace), client.MatchingLabels{multiRefPluginLabelKey: "true"})
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error listing plugins with the multi-ref label")
		g.Expect(pluginList.Items).To(HaveLen(2), "there should be 2 plugins with the multi-ref label")
		for i := range pluginList.Items {
			g.Expect(pluginList.Items[i].Status.IsReadyTrue()).To(BeTrue(), "each plugin with the multi-ref label should be ready")
			annotations := pluginList.Items[i].GetAnnotations()
			g.Expect(annotations).NotTo(BeNil(), "there should be annotations on each plugin with the multi-ref label")
			g.Expect(annotations[greenhouseapis.AnnotationKeyPluginTackingID]).To(Equal("Plugin/"+resolverPlugin.Name), "the tracking ID annotation should match the resolver plugin name")
		}
	}).Should(Succeed(), "all reference plugins should be ready")

	By("verifying the resolver extraEnvs values in flux HelmRelease")
	var extraEnvsFromHR any
	hr := &helmv2.HelmRelease{}
	hr.SetName(resolverPlugin.Name)
	hr.SetNamespace(resolverPlugin.Namespace)
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(hr), hr)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the HelmRelease for the resolver plugin")

		var valuesMap map[string]any
		err = json.Unmarshal(hr.Spec.Values.Raw, &valuesMap)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error unmarshalling the HelmRelease values")

		extraEnvsFromHR = valuesMap["extraEnvs"]

		// Since the resolver plugin references multiple plugins with a selector, the expected extraEnvs in the HelmRelease should be a combination of the extraEnvs from both plugins
		// We verify that all envs from both plugins are present, regardless of order (since selector results and helm value ordering are non-deterministic)
		var pluginAEnvsSlice, pluginBEnvsSlice []map[string]interface{}
		err = json.Unmarshal(pluginAEnvs.Raw, &pluginAEnvsSlice)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error unmarshalling plugin A envs")
		err = json.Unmarshal(pluginBEnvs.Raw, &pluginBEnvsSlice)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error unmarshalling plugin B envs")

		// Verify all envs from both plugins are present in the HelmRelease
		for _, expectedEnv := range pluginAEnvsSlice {
			g.Expect(extraEnvsFromHR).To(ContainElement(expectedEnv), "the HelmRelease should contain environment variable from plugin A: %v", expectedEnv)
		}
		for _, expectedEnv := range pluginBEnvsSlice {
			g.Expect(extraEnvsFromHR).To(ContainElement(expectedEnv), "the HelmRelease should contain environment variable from plugin B: %v", expectedEnv)
		}
	}).Should(Succeed(), "the HelmRelease for the resolver plugin should have the expected combined extraEnvs values")

	By("verifying the envs in the remote cluster deployment")
	deployment := &appsv1.Deployment{}
	deployment.SetName(resolverPlugin.Spec.ReleaseName + "-podinfo")
	deployment.SetNamespace(resolverPlugin.Spec.ReleaseNamespace)
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
		g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the deployment from the remote cluster")
		envVars := deployment.Spec.Template.Spec.Containers[0].Env

		// Verify that the expected envs from extraEnvsFromHR are present in the deployment
		g.Expect(containsExpectedEnvs(envVars, extraEnvsFromHR)).To(BeTrue(), "the deployment should contain all expected environment variables from the HelmRelease")
	}).Should(Succeed(), "the deployment should be present in the remote cluster with expected envs")
}
