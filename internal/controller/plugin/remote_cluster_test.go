// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

// Test environment.
var (
	remoteKubeConfig []byte
	remoteEnvTest    *envtest.Environment
	remoteK8sClient  client.Client
)

// Test stimuli.
var (
	testTeam = test.NewTeam(test.Ctx, "test-remotecluster-team", test.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))

	testPlugin = test.NewPlugin(test.Ctx, "test-plugindefinition", test.TestNamespace,
		test.WithCluster("test-cluster"),
		test.WithClusterPluginDefinition("test-plugindefinition"),
		test.WithReleaseName("release-test"),
		test.WithReleaseNamespace(test.TestNamespace),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name))

	testPluginWithSR = test.NewPlugin(test.Ctx, "test-plugin-secretref", test.TestNamespace,
		test.WithCluster("test-cluster"),
		test.WithClusterPluginDefinition("test-plugindefinition"),
		test.WithReleaseName("release-with-secretref"),
		test.WithPluginOptionValueFrom("secretValue", &greenhousev1alpha1.PluginValueFromSource{
			Secret: &greenhousev1alpha1.SecretKeyReference{
				Name: "test-secret",
				Key:  "test-key",
			},
		}),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
	)

	testPluginWithCRDs = test.NewPlugin(test.Ctx, "test-plugin-crd", test.TestNamespace,
		test.WithCluster(testCluster.GetName()),
		test.WithClusterPluginDefinition("test-plugindefinition-crd"),
		test.WithReleaseName("plugindefinition-crd"),
		test.WithReleaseNamespace(test.TestNamespace),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
	)

	testPluginWithExposedService = test.NewPlugin(test.Ctx, "test-plugin-exposed", test.TestNamespace,
		test.WithCluster(testCluster.GetName()),
		test.WithClusterPluginDefinition("test-plugindefinition-exposed"),
		test.WithReleaseName("plugindefinition-exposed"),
		test.WithReleaseNamespace(test.TestNamespace),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
	)

	testSecret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: test.TestNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
		},
		Data: map[string][]byte{
			"test-key": []byte("secret-value"),
		},
	}

	testPluginDefinition = test.NewClusterPluginDefinition(
		test.Ctx,
		"test-plugindefinition",
	)

	testPluginWithHelmChartCRDs = test.NewClusterPluginDefinition(
		test.Ctx,
		"test-plugindefinition-crd",
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:    "./../../test/fixtures/myChartWithCRDs",
			Version: "1.0.0",
		}),
	)

	pluginDefinitionWithExposedService = test.NewClusterPluginDefinition(
		test.Ctx,
		"test-plugindefinition-exposed",
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:    "./../../test/fixtures/chartWithExposedService",
			Version: "1.3.0",
		}))

	testCluster = test.NewCluster(test.Ctx, "test-cluster", test.TestNamespace,
		test.WithAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
		test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name))

	testClusterK8sSecret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: test.TestNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: testTeam.Name},
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
	}
)

// checkReadyConditionComponentsUnderTest asserts that components of plugin's ReadyCondition are ready,
// except for WorkloadReady condition, which is not a subject under test.
// This is done because the cumulative Ready condition in tests will be false due to workload not being ready.
func checkReadyConditionComponentsUnderTest(g Gomega, plugin *greenhousev1alpha1.Plugin) {
	readyCondition := plugin.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	g.Expect(readyCondition).ToNot(BeNil(), "Ready condition should not be nil")
	clusterAccessReadyCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
	g.Expect(clusterAccessReadyCondition).ToNot(BeNil())
	g.Expect(clusterAccessReadyCondition.Status).To(Equal(metav1.ConditionTrue), "ClusterAccessReady condition should be true")
	helmReconcileFailedCondition := plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
	g.Expect(helmReconcileFailedCondition).ToNot(BeNil())
	g.Expect(helmReconcileFailedCondition.Status).To(Equal(metav1.ConditionFalse), "HelmReconcileFailed condition should be false")
}

var _ = Describe("HelmController reconciliation", Ordered, func() {
	BeforeAll(func() {
		err := test.K8sClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the pluginDefinition")

		By("bootstrapping remote cluster")
		bootstrapRemoteCluster()

		By("creating a Team")
		Expect(test.K8sClient.Create(test.Ctx, testTeam)).Should(Succeed(), "there should be no error creating the Team")

		By("creating a cluster")
		Expect(test.K8sClient.Create(test.Ctx, testCluster)).Should(Succeed(), "there should be no error creating the cluster resource")

		// kubeConfigController ensures the namespace within the remote cluster -- we have to create it
		By("creating the namespace on the cluster")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPlugin.Spec.ReleaseNamespace, clientutil.WithPersistentConfig())
		remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(remoteRestClientGetter)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		err = remoteClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testPlugin.Spec.ReleaseNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the namespace")

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

	It("should correctly handle the plugin on a referenced cluster", func() {
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPlugin.Spec.ReleaseNamespace, clientutil.WithPersistentConfig())

		By("creating a plugin referencing the cluster")
		testPlugin.Spec.ClusterName = "test-cluster"
		Expect(test.K8sClient.Create(test.Ctx, testPlugin)).Should(Succeed(), "there should be no error updating the plugin")

		By("checking the ClusterAccessReadyCondition on the plugin")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, testPlugin)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the plugin")
			checkReadyConditionComponentsUnderTest(g, testPlugin)
		}).Should(Succeed(), "the ClusterAccessReadyCondition should be true")

		By("checking the helm releases deployed to the remote cluster")
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPlugin.Spec.ReleaseNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
		listAction := action.NewList(helmConfig)

		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(ContainElement(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(testPlugin.Spec.ReleaseName)}))), "the helm release should be deployed to the remote cluster")

		By("updating the plugin")
		_, err = clientutil.CreateOrPatch(test.Ctx, test.K8sClient, testPlugin, func() error {
			// this value enables the template of another pod
			testPlugin.Spec.OptionValues = append(testPlugin.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{Name: "enabled", Value: test.MustReturnJSONFor("true")})
			return nil
		})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error updating the plugin")
		By("checking the resources deployed to the remote cluster")
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the k8s client")
		podID := types.NamespacedName{Name: "alpine-flag", Namespace: test.TestNamespace}
		pod := &corev1.Pod{}
		Eventually(func(g Gomega) bool {
			err := remoteK8sClient.Get(test.Ctx, podID, pod)
			if err != nil {
				g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error retrieving the pod")
				return false
			}
			return true
		}).Should(BeTrue(), "the pod should have been created on the remote cluster")

		By("deleting the plugin")
		Expect(test.K8sClient.Delete(test.Ctx, testPlugin)).Should(Succeed(), "there should be no error deleting the plugin")

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

	It("should correctly handle the plugin on a referenced cluster with a secret reference", func() {
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPlugin.Spec.ReleaseNamespace, clientutil.WithPersistentConfig())

		By("creating a secret holding the OptionValue referenced by the Plugin")
		Expect(test.K8sClient.Create(test.Ctx, &testSecret)).Should(Succeed())

		By("creating a plugin referencing the cluster")
		testPluginWithSR.Spec.ClusterName = "test-cluster"
		Expect(test.K8sClient.Create(test.Ctx, testPluginWithSR)).Should(Succeed(), "there should be no error updating the plugin")

		By("checking the helm releases deployed to the remote cluster")
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPluginWithSR.Namespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
		listAction := action.NewList(helmConfig)

		Eventually(func(g Gomega) []*release.Release {
			releases, err := listAction.Run()
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(ContainElement(
			gstruct.PointTo(
				gstruct.MatchFields(
					gstruct.IgnoreExtras, gstruct.Fields{
						"Name":   Equal(testPluginWithSR.Spec.ReleaseName),
						"Config": gstruct.MatchKeys(gstruct.IgnoreExtras, gstruct.Keys{"secretValue": Equal("secret-value")})}))), "the helm release should be deployed to the remote cluster")

		By("deleting the plugin")
		Expect(test.K8sClient.Delete(test.Ctx, testPluginWithSR)).Should(Succeed(), "there should be no error deleting the plugin")

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func(g Gomega) []*release.Release {
			releases, err := listAction.Run()
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

	It("should correctly handle the plugin on a referenced cluster with a different namespace", func() {
		testPluginInDifferentNamespace := test.NewPlugin(test.Ctx, "test-plugin-in-made-up-namespace", test.TestNamespace,
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeam.Name),
			test.WithCluster(testCluster.GetName()),
			test.WithClusterPluginDefinition(testPluginDefinition.GetName()),
			test.WithReleaseName("release-test-in-made-up-namespace"),
			test.WithReleaseNamespace("made-up-namespace"))

		Expect(testPluginInDifferentNamespace.GetNamespace()).
			Should(Equal(test.TestNamespace), "the namespace should be the test namespace")
		Expect(testPluginInDifferentNamespace.Spec.ReleaseNamespace).
			Should(Equal("made-up-namespace"), "the release namespace should be the made-up-namespace")

		By("creating a plugin referencing the cluster")
		Expect(test.K8sClient.Create(test.Ctx, testPluginInDifferentNamespace)).
			Should(Succeed(), "there should be no error creating the plugin")

		By("checking the helm releases deployed to the remote cluster in a different namespace")
		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(
			remoteKubeConfig, testPluginInDifferentNamespace.Spec.ReleaseNamespace, clientutil.WithPersistentConfig(),
		)
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPluginInDifferentNamespace.Spec.ReleaseNamespace)
		Expect(err).
			ShouldNot(HaveOccurred(), "there should be no error creating helm config")

		Eventually(func(g Gomega) string {
			release, err := action.NewGet(helmConfig).Run(testPluginInDifferentNamespace.GetReleaseName())
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return release.Namespace
		}).Should(
			Equal(testPluginInDifferentNamespace.Spec.ReleaseNamespace),
			"the helm release should be deployed to the remote cluster in a different namespace",
		)

		By("checking the pod template without explicit namespace is deployed to the releaseNamespace")
		podName := types.NamespacedName{Name: "alpine", Namespace: testPluginInDifferentNamespace.Spec.ReleaseNamespace}
		Eventually(func(g Gomega) {
			pod := &corev1.Pod{}
			err := remoteK8sClient.Get(test.Ctx, podName, pod)
			g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the pod")
		}).Should(
			Succeed(),
			"the pod template without explicit namespace should be deployed to the releaseNamespace",
		)

		By("deleting the plugin")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginInDifferentNamespace)

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func(g Gomega) []*release.Release {
			releases, err := action.NewList(helmConfig).Run()
			g.Expect(err).
				ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

	It("should re-create CRD if CRD was deleted", func() {
		By("creating plugin definition with CRDs")
		Expect(test.K8sClient.Create(test.Ctx, testPluginWithHelmChartCRDs)).To(Succeed(), "should create plugin definition")

		remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPluginWithCRDs.Spec.ReleaseNamespace, clientutil.WithPersistentConfig())

		By("creating test plugin referencing the cluster")
		testPluginWithCRDs.Spec.ClusterName = "test-cluster"
		Expect(test.K8sClient.Create(test.Ctx, testPluginWithCRDs)).
			Should(Succeed(), "there should be no error creating the plugin")

		By("checking the ClusterAccessReadyCondition on the plugin")
		Eventually(func(g Gomega) bool {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginWithCRDs.Name, Namespace: testPluginWithCRDs.Namespace}, testPluginWithCRDs)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the plugin")
			clusterAccessReadyCondition := testPluginWithCRDs.Status.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
			readyCondition := testPluginWithCRDs.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(clusterAccessReadyCondition).ToNot(BeNil(), "the ClusterAccessReadyCondition should not be nil")
			g.Expect(readyCondition).ToNot(BeNil(), "the ReadyCondition should not be nil")
			return true
		}).Should(BeTrue(), "the ClusterAccessReadyCondition should be false")

		By("checking the helm releases deployed to the remote cluster")
		helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPluginWithCRDs.Spec.ReleaseNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
		listAction := action.NewList(helmConfig)
		Eventually(func() []*release.Release {
			releases, err := listAction.Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(ContainElement(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(testPluginWithCRDs.Spec.ReleaseName)}))), "the helm release should be deployed to the remote cluster")

		By("checking if helm release exists")
		Eventually(func() bool {
			_, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, remoteRestClientGetter, testPluginWithCRDs)
			return err == nil
		}).Should(BeTrue(), "release for helm chart should already exist")

		teamCRDName := "teams.greenhouse.fixtures"
		teamCRDKey := types.NamespacedName{Name: teamCRDName, Namespace: ""}

		By("Getting Team CRD from remote cluster")
		var teamCRD = &apiextensionsv1.CustomResourceDefinition{}
		Eventually(func(g Gomega) {
			g.Expect(remoteK8sClient.Get(test.Ctx, teamCRDKey, teamCRD)).To(Succeed(), "there must be no error getting the Team CRD")
			g.Expect(teamCRD.Name).To(Equal(teamCRDName), "created Team CRD should have the correct name")
		}).ShouldNot(HaveOccurred(), "Team CRD should be created on remote cluster")

		By("deleting Team CRD from the remote cluster")
		Eventually(func() error {
			return remoteK8sClient.Delete(test.Ctx, teamCRD)
		}).Should(Succeed(), "there must be no error deleting Team CRD")

		By("setting label on plugin to trigger reconciliation")
		// Get up-to-date version of plugin.
		err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginWithCRDs.Name, Namespace: testPluginWithCRDs.Namespace}, testPluginWithCRDs)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting plugin")
		// Set a label on the plugin to trigger reconciliation.
		testPluginWithCRDs.Labels = map[string]string{"test": "label"}
		Expect(test.K8sClient.Update(test.Ctx, testPluginWithCRDs)).Should(Succeed(), "there should be no error updating the plugin")

		By("ensuring Team CRD was re-created in the remote cluster")
		Eventually(func(g Gomega) {
			var teamCRD = &apiextensionsv1.CustomResourceDefinition{}
			g.Expect(remoteK8sClient.Get(test.Ctx, teamCRDKey, teamCRD)).To(Succeed(), "there must be no error getting the Team CRD")
			g.Expect(teamCRD.Name).To(Equal(teamCRDName), "re-created Team CRD should have the correct name")
		}).Should(Succeed(), "Team CRD should be re-created")

		By("deleting the plugin")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginWithCRDs)

		By("checking the helm releases deployed to the remote cluster")
		Eventually(func() []*release.Release {
			releases, err := action.NewList(helmConfig).Run()
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
			return releases
		}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
	})

	When("reconciling status for plugin with exposed service", func() {
		It("should generate exposed service URL", func() {
			common.DNSDomain = "example.com"

			By("creating plugin definition with exposed service")
			Expect(test.K8sClient.Create(test.Ctx, pluginDefinitionWithExposedService)).To(Succeed(), "should create plugin definition")

			testPluginWithExposedService1 := testPluginWithExposedService.DeepCopy()

			remoteRestClientGetter := clientutil.NewRestClientGetterFromBytes(remoteKubeConfig, testPluginWithExposedService1.Spec.ReleaseNamespace, clientutil.WithPersistentConfig())

			By("creating test plugin referencing the cluster")
			testPluginWithExposedService1.Spec.ClusterName = "test-cluster"
			Expect(test.K8sClient.Create(test.Ctx, testPluginWithExposedService1)).
				Should(Succeed(), "there should be no error creating the plugin")

			By("checking the helm releases deployed to the remote cluster")
			helmConfig, err := helm.ExportNewHelmAction(remoteRestClientGetter, testPluginWithExposedService1.Spec.ReleaseNamespace)
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")
			listAction := action.NewList(helmConfig)
			Eventually(func() []*release.Release {
				releases, err := listAction.Run()
				Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
				return releases
			}).Should(ContainElement(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{"Name": Equal(testPluginWithExposedService1.Spec.ReleaseName)}))), "the helm release should be deployed to the remote cluster")

			By("checking plugin status")
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginWithExposedService1.Name, Namespace: testPluginWithExposedService1.Namespace}, testPluginWithExposedService1)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting plugin")
				statusUpToDateCondition := testPluginWithExposedService1.Status.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition)
				g.Expect(statusUpToDateCondition).ToNot(BeNil(), "StatusUpToDate condition should not be nil")
				g.Expect(statusUpToDateCondition.Status).To(Equal(metav1.ConditionTrue), "plugin status up to date condition should be set to true")
			}).Should(Succeed(), "plugin should have correct status")

			By("checking Plugin exposed services")
			Eventually(func(g Gomega) {
				g.Expect(testPluginWithExposedService1.Status.ExposedServices).ToNot(BeEmpty(), "exposed services in plugin status should not be empty")
				g.Expect(testPluginWithExposedService1.Status.ExposedServices).To(HaveLen(2), "there should be two exposed services (service + ingress)")

				serviceFound := false
				ingressFound := false
				var serviceURL, ingressURL string

				for url, svc := range testPluginWithExposedService1.Status.ExposedServices {
					if svc.Type == greenhousev1alpha1.ServiceTypeService {
						serviceFound = true
						serviceURL = url
						g.Expect(svc.Name).To(Equal("exposed-service"), "service should have correct name")
						g.Expect(svc.Port).To(Equal(int32(80)), "service should have port 80")
						g.Expect(svc.Namespace).To(Equal("test-org"), "service should have correct namespace")
					}
					if svc.Type == greenhousev1alpha1.ServiceTypeIngress {
						ingressFound = true
						ingressURL = url
						g.Expect(svc.Name).To(Equal("exposed-ingress"), "ingress should have correct name")
						g.Expect(svc.Namespace).To(Equal("test-org"), "ingress should have correct namespace")
					}
				}

				g.Expect(serviceFound).To(BeTrue(), "should find service type exposure")
				g.Expect(ingressFound).To(BeTrue(), "should find ingress type exposure")

				expectedServiceURL := common.URLForExposedServiceInPlugin("exposed-service", testPluginWithExposedService1)
				g.Expect(serviceURL).To(Equal(expectedServiceURL), "service URL should be generated correctly")

				g.Expect(ingressURL).To(Equal("https://api.test.example.com"), "ingress URL should match the specified host with HTTPS")
			}).Should(Succeed(), "plugin should have correct status")

			By("checking that the plugin has the exposed-services label")
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginWithExposedService1.Name, Namespace: testPluginWithExposedService1.Namespace}, testPluginWithExposedService1)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting plugin")
				g.Expect(testPluginWithExposedService1.GetLabels()).To(HaveKeyWithValue(greenhouseapis.LabelKeyPluginExposedServices, "true"),
					"Plugin with ExposedServices should have plugin-exposed-services label")
			}).Should(Succeed(), "plugin should have correct exposed services label")

			By("deleting the plugin")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginWithExposedService1)

			By("checking the helm releases deployed to the remote cluster")
			Eventually(func() []*release.Release {
				releases, err := action.NewList(helmConfig).Run()
				Expect(err).ShouldNot(HaveOccurred(), "there should be no error listing helm releases")
				return releases
			}).Should(BeEmpty(), "the helm release should be deleted from the remote cluster")
		})

		It("should set error on status when cluster name is missing", func() {
			testPluginWithExposedService2 := testPluginWithExposedService.DeepCopy()

			By("checking greenhouse namespace")
			var greenhouseNamespace = new(corev1.Namespace)
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: "greenhouse"}, greenhouseNamespace)
			if err != nil {
				By("creating central cluster greenhouse namespace")
				greenhouseNamespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "greenhouse"}}
				err = test.K8sClient.Create(test.Ctx, greenhouseNamespace)
				Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the greenhouse namespace")
			}

			By("creating a test Team in greenhouse namespace")
			testCentralTeam := test.NewTeam(test.Ctx, "test-central-team", "greenhouse", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
			Expect(test.K8sClient.Create(test.Ctx, testCentralTeam)).To(Succeed(), "there should be no error creating a test Team in the greenhouse namespace")
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testCentralTeam.Name)(testPluginWithExposedService2)

			By("creating test plugin without ClusterName")
			// Deploy plugin to central cluster.
			testPluginWithExposedService2.Namespace = "greenhouse"
			testPluginWithExposedService2.Spec.ClusterName = ""
			Expect(test.K8sClient.Create(test.Ctx, testPluginWithExposedService2)).
				Should(Succeed(), "there should be no error creating the plugin")

			By("checking plugin status")
			Eventually(func(g Gomega) {
				err = test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: testPluginWithExposedService2.Name, Namespace: testPluginWithExposedService2.Namespace}, testPluginWithExposedService2)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting plugin")
				statusUpToDateCondition := testPluginWithExposedService2.Status.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition)
				g.Expect(statusUpToDateCondition).ToNot(BeNil(), "status up to date condition should exist")
				g.Expect(statusUpToDateCondition.Status).To(Equal(metav1.ConditionFalse), "plugin status up to date condition should be set to false")
				g.Expect(statusUpToDateCondition.Message).To(ContainSubstring("plugin does not have ClusterName"), "plugin status up to date condition should have correct message")
				g.Expect(testPluginWithExposedService2.Status.ExposedServices).To(BeEmpty(), "exposed services in plugin status should be empty")
			}).Should(Succeed(), "plugin should have correct status")

			By("deleting the plugin")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginWithExposedService2)
			By("deleting the test team")
			test.EventuallyDeleted(test.Ctx, test.K8sClient, testCentralTeam)
		})
	})
})

func bootstrapRemoteCluster() {
	_, remoteK8sClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
}
