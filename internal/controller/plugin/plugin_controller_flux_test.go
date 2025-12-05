// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var (
	remoteKubeConfig []byte
	remoteEnvTest    *envtest.Environment
	remoteK8sClient  client.Client
)

var (
	testPluginTeam = test.NewTeam(test.Ctx, "test-remote-cluster-team", test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

	testOrganization = test.NewOrganization(test.Ctx, test.TestNamespace)

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
		test.WithCluster("test-flux-cluster"),
		test.WithClusterPluginDefinition("test-flux-plugindefinition"),
		test.WithReleaseName("release-test-flux"),
		test.WithReleaseNamespace(test.TestNamespace),
		test.WithPluginLabel(greenhouseapis.GreenhouseHelmDeliveryToolLabel, greenhouseapis.GreenhouseHelmDeliveryToolFlux),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testPluginTeam.Name),
		test.WithPluginOptionValue("flatOption", test.MustReturnJSONFor("flatValue")),
		test.WithPluginOptionValue("nested.option", test.MustReturnJSONFor("nestedValue")),
		test.WithPluginOptionValueFrom("nested.secretOption", &greenhousev1alpha1.ValueFromSource{
			Secret: &greenhousev1alpha1.SecretKeyReference{
				Name: "test-cluster",
				Key:  greenhouseapis.GreenHouseKubeConfigKey,
			},
		}),
		test.WithPluginWaitFor([]greenhousev1alpha1.WaitForItem{
			{
				PluginRef: greenhousev1alpha1.PluginRef{
					Name: "", PluginPreset: "dependent-preset-1",
				},
			},
			{
				PluginRef: greenhousev1alpha1.PluginRef{
					Name: "dependent-plugin-1", PluginPreset: "",
				},
			},
		}),
	)

	testPluginDefinition = test.NewClusterPluginDefinition(
		test.Ctx,
		"test-flux-plugindefinition",
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "dummy",
			Repository: "oci://greenhouse/helm-charts",
			Version:    "1.0.0",
		}),
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

	uiPluginDefinition = test.NewClusterPluginDefinition(
		test.Ctx, "test-flux-ui-plugindefinition",
		test.WithVersion("1.0.0"),
		test.WithoutHelmChart(),
		test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
			Name:    "test-ui-app",
			Version: "0.0.1",
		}),
	)

	uiPlugin = test.NewPlugin(test.Ctx, "test-flux-ui-plugin", test.TestNamespace,
		test.WithClusterPluginDefinition(uiPluginDefinition.Name),
		test.WithReleaseName("release-test-flux"),
		test.WithPluginLabel(greenhouseapis.GreenhouseHelmDeliveryToolLabel, greenhouseapis.GreenhouseHelmDeliveryToolFlux),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testPluginTeam.Name),
	)
)

var _ = Describe("Flux Plugin Controller", Ordered, func() {
	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginDefinition")

		err = test.K8sClient.Create(test.Ctx, uiPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the UI pluginDefinition")

		By("bootstrapping remote cluster")
		_, remoteK8sClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)

		By("creating an Organization")
		Expect(test.K8sClient.Create(test.Ctx, testOrganization)).Should(Succeed(), "there should be no error creating the Organization")

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
		actual, err := addValuesToHelmRelease(test.Ctx, test.K8sClient, testPlugin, false)
		Expect(err).ToNot(HaveOccurred(), "there should be no error computing the HelmRelease values for the Plugin")

		By("checking the computed Values")
		Expect(actual).To(Equal(expectedRaw), "the computed HelmRelease values should match the expected values")
	})

	It("should create HelmRelease for Plugin", func() {
		By("ensuring HelmRepository has been created for ClusterPluginDefinition")
		helmRepository := &sourcecontroller.HelmRepository{}
		repoName := flux.ChartURLToName(testPluginDefinition.Spec.HelmChart.Repository)
		repoNamespace := flux.HelmRepositoryDefaultNamespace
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: repoName, Namespace: repoNamespace}, helmRepository)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRepository")
		}).Should(Succeed())

		By("creating test Plugin")
		Expect(test.K8sClient.Create(test.Ctx, testPlugin)).To(Succeed(), "failed to create Plugin")

		By("ensuring HelmRelease has been created")
		release := &helmv2.HelmRelease{}
		releaseKey := types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, releaseKey, release)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRelease")
		}).Should(Succeed())

		By("ensuring Plugin dependencies has been resolved and set on the HelmRelease")
		Expect(release.Spec.DependsOn).To(ContainElement(helmv2.DependencyReference{Name: "dependent-plugin-1"}),
			"Flux HelmRelease should have the dependency for global plugin's release set")
		Expect(release.Spec.DependsOn).To(ContainElement(helmv2.DependencyReference{Name: "dependent-preset-1-" + testPlugin.Spec.ClusterName}),
			"Flux HelmRelease should have the dependency for resolved plugin's release set")

		By("ensuring the Plugin Status is updated")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPlugin), testPlugin)
			g.Expect(err).ToNot(HaveOccurred())

			clusterAccessReadyCondition := testPlugin.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
			g.Expect(clusterAccessReadyCondition).ToNot(BeNil())
			g.Expect(clusterAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "ClusterAccessReady condition should be true")
			helmReconcileFailedCondition := testPlugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
			g.Expect(helmReconcileFailedCondition).ToNot(BeNil())
			g.Expect(helmReconcileFailedCondition.Status).To(Equal(metav1.ConditionUnknown), "HelmReconcileFailed condition should be unknown")
			readyCondition := testPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil())
			g.Expect(readyCondition.IsFalse()).To(BeTrue(), "Ready condition should be set to false")
			g.Expect(readyCondition.Message).To(ContainSubstring("Reconciling"))
			// The status won't change further, because Flux HelmController can't be registered here. See E2E tests.
		}).Should(Succeed())

		By("ensuring the Flux HelmRelease is suspended")
		test.MustSetAnnotation(test.Ctx, test.K8sClient, testPlugin, lifecycle.SuspendAnnotation, "true")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, releaseKey, release)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRelease")
			g.Expect(release.Spec.Suspend).To(BeTrue(), "HelmRelease should be suspended")
		}).Should(Succeed())

		By("ensuring the Flux HelmRelease is resumed")
		test.MustRemoveAnnotation(test.Ctx, test.K8sClient, testPlugin, lifecycle.SuspendAnnotation)

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, releaseKey, release)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRelease")
			g.Expect(release.Spec.Suspend).To(BeFalse(), "HelmRelease should not be suspended")
		}).Should(Succeed())

		By("ensuring the HelmRelease has the last reconcile annotation updated")
		test.MustSetAnnotation(test.Ctx, test.K8sClient, testPlugin, lifecycle.ReconcileAnnotation, "foobar")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, releaseKey, release)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRelease")
			g.Expect(release.GetAnnotations()).Should(HaveKeyWithValue(fluxmeta.ReconcileRequestAnnotation, "foobar"), "HelmRelease should have the reconcile annotation updated")

			err = test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPlugin), testPlugin)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get Plugin")
			g.Expect(testPlugin.Status.LastReconciledAt).To(Equal("foobar"), "Plugin status LastReconcile should be updated")
		}).Should(Succeed())

		test.MustRemoveAnnotation(test.Ctx, test.K8sClient, testPlugin, lifecycle.ReconcileAnnotation)

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, releaseKey, release)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get HelmRelease")
			g.Expect(release.GetAnnotations()).ShouldNot(HaveKey(fluxmeta.ReconcileRequestAnnotation), "HelmRelease should have the reconcile annotation removed")

			err = test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPlugin), testPlugin)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get Plugin")
			g.Expect(testPlugin.Status.LastReconciledAt).To(BeEmpty(), "Plugin status LastReconcile should be empty")
		}).Should(Succeed())
	})

	It("should reconcile a UI-only Plugin", func() {
		By("creating UI-only Plugin")
		Expect(test.K8sClient.Create(test.Ctx, uiPlugin)).To(Succeed(), "failed to create UI-only Plugin")

		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(uiPlugin), uiPlugin)
			g.Expect(err).ToNot(HaveOccurred())
			readyCondition := uiPlugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil())
			g.Expect(readyCondition.IsTrue()).To(BeTrue())
			g.Expect(uiPlugin.Status.UIApplication).To(Equal(uiPluginDefinition.Spec.UIApplication))
		}).Should(Succeed())

		By("ensuring HelmRelease has not been created")
		Eventually(func(g Gomega) {
			release := &helmv2.HelmRelease{}
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: uiPlugin.Name, Namespace: uiPlugin.Namespace}, release)
			g.Expect(err).To(HaveOccurred(), "there should be an error getting the HelmRelease")
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
		}).Should(Succeed())

		By("checking that the UI plugin has the ui-plugin label")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(uiPlugin), uiPlugin)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(uiPlugin.GetLabels()).To(HaveKeyWithValue(greenhouseapis.LabelKeyUIPlugin, "true"),
				"Plugin with UIApplication should have ui-plugin label")
			g.Expect(uiPlugin.GetLabels()).ToNot(HaveKey(greenhouseapis.LabelKeyPluginExposedServices),
				"Plugin without ExposedServices should not have plugin-exposed-services label")
		}).Should(Succeed())

		By("checking that the non-UI plugin does not have the ui-plugin label")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPlugin), testPlugin)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(testPlugin.GetLabels()).ToNot(HaveKey(greenhouseapis.LabelKeyUIPlugin),
				"Plugin without UIApplication should not have ui-plugin label")
			g.Expect(uiPlugin.GetLabels()).ToNot(HaveKey(greenhouseapis.LabelKeyPluginExposedServices),
				"Plugin without ExposedServices should not have plugin-exposed-services label")
		}).Should(Succeed())
	})
})
