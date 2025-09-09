// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build pluginE2E

package plugin

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmcontroller "github.com/fluxcd/helm-controller/api/v2"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	remoteClusterName         = "remote-plugin-cluster"
	preventDeletionAnnotation = "greenhouse.sap/prevent-deletion"
)

var (
	env           *shared.TestEnv
	ctx           context.Context
	adminClient   client.Client
	remoteClient  client.Client
	testStartTime time.Time
	team          *greenhousev1alpha1.Team
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = context.Background()
	env = shared.NewExecutionEnv()

	var err error
	adminClient, err = clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the admin client")
	remoteClient, err = clientutil.NewK8sClientFromRestClientGetter(env.RemoteRestClientGetter)
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the remote client")
	env = env.WithOrganization(ctx, adminClient, "./testdata/organization.yaml")
	team = test.NewTeam(ctx, "test-plugin-e2e-team", env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = adminClient.Create(ctx, team)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating a Team")
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	test.EventuallyDeleted(ctx, adminClient, team)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Plugin E2E", Ordered, func() {
	It("should onboard remote cluster", func() {
		By("onboarding remote cluster")
		shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, team.Name)
	})
	It("should have a cluster resource created", func() {
		By("verifying if the cluster resource is created")
		Eventually(func(g Gomega) {
			err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
			g.Expect(err).ToNot(HaveOccurred())
		}).Should(Succeed(), "cluster resource should be created")

		By("verifying the cluster status is ready")
		shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)
	})

	It("should deploy the plugin", func() {
		By("creating plugin definition")
		testPluginDefinition := fixtures.PrepareNginxPluginDefinition(env.TestNamespace)
		err := adminClient.Create(ctx, testPluginDefinition)
		Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

		By("Checking the plugin definition is ready")
		pluginDefinitionList := &greenhousev1alpha1.ClusterPluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Creating the plugin")
		// Creating plugin with release name
		testPlugin := fixtures.PreparePlugin("test-nginx-plugin-1", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(remoteClusterName),
			test.WithReleaseName("test-nginx-plugin"),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
			test.WithReleaseName("test-nginx-plugin-1"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
		)
		err = adminClient.Create(ctx, testPlugin)
		Expect(err).ToNot(HaveOccurred())

		By("Checking the plugin status is ready")
		pluginList := &greenhousev1alpha1.PluginList{}
		Eventually(func(g Gomega) {
			err = adminClient.List(ctx, pluginList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
			g.Expect(pluginList.Items[0].Status.HelmReleaseStatus).ToNot(BeNil())
			g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
		}).Should(Succeed())

		By("Checking deployment")
		deploymentList := &appsv1.DeploymentList{}
		Eventually(func(g Gomega) {
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(deploymentList.Items).ToNot(BeEmpty())
		}).Should(Succeed())

		By("Checking the name of deployment")
		nginxDeploymentExists := false
		for _, deployment := range deploymentList.Items {
			if strings.Contains(deployment.Name, "nginx") {
				nginxDeploymentExists = true
				Expect(deployment.Spec.Replicas).To(PointTo(Equal(int32(1))))
				break
			}
		}
		Expect(nginxDeploymentExists).To(BeTrue())

		By("Updating replicas")
		Eventually(func(g Gomega) {
			namespacedName := types.NamespacedName{Name: testPlugin.Name, Namespace: env.TestNamespace}
			err = adminClient.Get(ctx, namespacedName, testPlugin)
			g.Expect(err).NotTo(HaveOccurred())
			test.SetOptionValueForPlugin(testPlugin, "replicaCount", "2")
			err = adminClient.Update(ctx, testPlugin)
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())

		By("Check the diff status")
		Eventually(func(g Gomega) {
			err = adminClient.Get(ctx, client.ObjectKey{Name: testPlugin.Name, Namespace: env.TestNamespace}, testPlugin)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(testPlugin.Status.HelmReleaseStatus).ToNot(BeNil())
			g.Expect(len(testPlugin.Status.HelmReleaseStatus.Diff) > 0).To(BeTrue())
		}).Should(Succeed())

		By("Check replicas in deployment list")
		Eventually(func(g Gomega) {
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			for _, deployment := range deploymentList.Items {
				if strings.Contains(deployment.Name, "nginx") {
					g.Expect(deployment.Spec.Replicas).To(PointTo(Equal(int32(2))))
				}
			}
		}).Should(Succeed())

		By("Deleting plugin")
		test.EventuallyDeleted(ctx, adminClient, testPlugin)

		By("Check, is deployment deleted")
		Eventually(func(g Gomega) bool {
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(0))
			return true
		}).Should(BeTrue())

		By("Deleting plugin definition")
		test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
	})

	It("should deploy the plugin by the plugin preset", func() {
		By("Creating plugin definition")
		testPluginDefinition := fixtures.PrepareNginxPluginDefinition(env.TestNamespace)
		err := adminClient.Create(ctx, testPluginDefinition)
		Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

		By("Checking the plugin definition is ready")
		pluginDefinitionList := &greenhousev1alpha1.ClusterPluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Prepare the plugin")
		// Creating plugin with release name
		testPlugin := fixtures.PreparePlugin("test-nginx-plugin-2", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithReleaseName("test-nginx-plugin"),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
			test.WithReleaseName("test-nginx-plugin-2"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
		)

		By("Add labels to remote cluster")
		remoteCluster := &greenhousev1alpha1.Cluster{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
		Expect(err).ToNot(HaveOccurred())
		remoteCluster.Labels["app"] = "test-cluster"
		err = adminClient.Update(ctx, remoteCluster)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the plugin preset")
		testPluginPreset := test.NewPluginPreset("test-nginx-plugin-preset", env.TestNamespace,
			test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
			test.WithPluginPresetPluginSpec(testPlugin.Spec),
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-cluster",
				},
			}),
		)
		err = adminClient.Create(ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginPreset")

		By("Checking the plugin status is ready")
		Eventually(func(g Gomega) {
			pluginList := &greenhousev1alpha1.PluginList{}
			err = adminClient.List(ctx, pluginList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
		}).Should(Succeed())

		By("Check the replicas in deployment")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1))
			g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(1))))
		}).Should(Succeed())

		By("Updating plugin preset with cluster overview")
		Eventually(func(g Gomega) {
			err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the PluginPreset")
			testPluginPreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
				{
					ClusterName: remoteClusterName,
					Overrides: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "replicaCount",
							Value: &apiextensionsv1.JSON{Raw: []byte("2")},
						},
					},
				},
			}
			err = adminClient.Update(ctx, testPluginPreset)
			g.Expect(err).ToNot(HaveOccurred(), "failed to update the PluginPreset")
		}).Should(Succeed(), "there should be no error updating the PluginPreset")

		By("Check the replicas in deployment")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1))
			g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(2))))
		}).Should(Succeed())

		By("Updating plugin preset with cluster option override")
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred())
		testPluginPreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
			{
				ClusterName: remoteClusterName,
				Overrides: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "replicaCount",
						Value: &apiextensionsv1.JSON{Raw: []byte("3")},
					},
				},
			},
		}
		err = adminClient.Update(ctx, testPluginPreset)
		Expect(err).ToNot(HaveOccurred(), "there should be no error updating the plugin preset with override")

		By("Ensure the plugin preset is updated")
		Eventually(func(g Gomega) {
			err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
			Expect(err).ToNot(HaveOccurred())
			Expect(testPluginPreset.Spec.ClusterOptionOverrides).To(HaveLen(1))
			Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides).To(HaveLen(1))
			Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Name).To(BeEquivalentTo("replicaCount"))
			Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Value).To(BeEquivalentTo(&apiextensionsv1.JSON{Raw: []byte("3")}))
		}).Should(Succeed())

		By("Checking the plugin status is ready")
		Eventually(func(g Gomega) {
			pluginList := &greenhousev1alpha1.PluginList{}
			err = adminClient.List(ctx, pluginList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
			g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
		}).Should(Succeed())

		By("Check the replicas in deployment")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1))
			g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(3))))
		}).Should(Succeed())

		By("Deleting the plugin preset")
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred())
		// Remove prevent-deletion annotation before deleting plugin preset.
		_, _ = clientutil.Patch(ctx, adminClient, testPluginPreset, func() error {
			delete(testPluginPreset.Annotations, preventDeletionAnnotation)
			return nil
		})
		test.EventuallyDeleted(ctx, adminClient, testPluginPreset)

		By("Check that the deployment is deleted")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err := remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error listing deployments")
			g.Expect(deploymentList.Items).To(BeEmpty(), "there should be no deployments")
		}).Should(Succeed(), "deployments list should be empty")

		By("Deleting the plugin definition")
		test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
	})

	It("should rollback the first helm release on failed deployment", func() {
		By("Creating a plugin definition with cert-manager helm chart")
		pluginDefinition := fixtures.PrepareCertManagerPluginDefinition(env.TestNamespace)
		err := adminClient.Create(ctx, pluginDefinition)
		Expect(client.IgnoreAlreadyExists(err)).To(Succeed(), "there should be no error creating the plugin definition")

		By("Setting the HELM_RELEASE_TIMEOUT to 5 seconds")
		Expect(os.Setenv("HELM_RELEASE_TIMEOUT", "5")).To(Succeed(), "there should be no error setting HELM_RELEASE_TIMEOUT env")

		By("Preparing the plugin")
		// Creating plugin with release name
		plugin := fixtures.PreparePlugin("test-cert-manager-plugin", env.TestNamespace,
			test.WithPluginDefinition(pluginDefinition.Name),
			test.WithCluster(remoteClusterName),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithReleaseName("test-cert-manager-plugin"),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name),
		)

		By("Installing release manually on the remote cluster")
		_, err = helm.ExportInstallHelmRelease(ctx, adminClient, env.RemoteRestClientGetter, pluginDefinition.Spec, plugin, false)
		Expect(err).To(HaveOccurred(), "there should be an error installing the helm chart")

		By("Creating helm config")
		cfg, err := helm.ExportNewHelmAction(env.RemoteRestClientGetter, plugin.Spec.ReleaseNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating helm config")

		By("Checking if the release has status set to failed")
		Eventually(func(g Gomega) {
			getAction := action.NewGet(cfg)
			helmRelease, err := getAction.Run(plugin.Name)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
			g.Expect(helmRelease.Info.Status).To(Equal(release.StatusFailed), "helm release status should be set to failed")
			g.Expect(helmRelease.Version).To(Equal(1), "helm release version should equal 1")
		}).Should(Succeed(), "helm release should be set to failed")

		By("Setting the HELM_RELEASE_TIMEOUT to 5 minutes")
		Expect(os.Setenv("HELM_RELEASE_TIMEOUT", "300")).To(Succeed(), "there should be no error setting HELM_RELEASE_TIMEOUT env")

		By("Creating the plugin")
		Expect(adminClient.Create(ctx, plugin)).To(Succeed(), "there should be no error creating the plugin")

		By("Checking if the rollback took place")
		Eventually(func(g Gomega) {
			namespacedName := types.NamespacedName{Name: plugin.Name, Namespace: env.TestNamespace}
			err := adminClient.Get(ctx, namespacedName, plugin)
			g.Expect(err).NotTo(HaveOccurred(), "there should be no error getting the plugin")
			g.Expect(plugin.Status.HelmReleaseStatus).ToNot(BeNil(), "plugin HelmReleaseStatus should not be nil")
			g.Expect(plugin.Status.HelmReleaseStatus.Status).To(Equal("deployed"), "helm release status should be set to deployed")
		}).Should(Succeed(), "plugin should show HelmReleaseStatus as deployed")

		By("Checking if the release has status set to deployed")
		Eventually(func(g Gomega) {
			getAction := action.NewGet(cfg)
			helmRelease, err := getAction.Run(plugin.Name)
			g.Expect(err).ShouldNot(HaveOccurred(), "there should be no error getting the helm release")
			g.Expect(helmRelease.Info.Status).To(Equal(release.StatusDeployed), "helm release status should be set back to deployed")
			g.Expect(helmRelease.Version).To(Equal(3), "helm release version should change to 3")
		}).Should(Succeed(), "helm release status should be set back to deployed")

		By("Checking the helm release history for rollback")
		historyAction := action.NewHistory(cfg)
		releases, err := historyAction.Run(plugin.Name)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting helm release history")
		Expect(releases).To(HaveLen(3), "there should be exactly 3 entries in helm release history")
		secondReleaseInfo := releases[1].Info
		Expect(secondReleaseInfo.Description).To(Equal("Rollback to 1"), "release history should show Rollback to 1 in description")
		Expect(secondReleaseInfo.Status).To(Equal(release.StatusSuperseded), "release history should show Superseded status")
		thirdReleaseInfo := releases[2].Info
		Expect(thirdReleaseInfo.Status).To(Equal(release.StatusDeployed), "the last release entry should have status set to deployed")

		By("Deleting the plugin")
		test.EventuallyDeleted(ctx, adminClient, plugin)

		By("Check that the deployment is deleted")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err := remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error listing deployments")
			g.Expect(deploymentList.Items).To(BeEmpty(), "there should be no deployments")
		}).Should(Succeed(), "deployments list should be empty")

		By("Deleting the plugin definition")
		test.EventuallyDeleted(ctx, adminClient, pluginDefinition)
	})

	It("should deploy the plugin with flux", func() {
		By("Creating plugin definition")
		testPluginDefinition := fixtures.PreparePodInfoPluginDefinition(env.TestNamespace)
		err := adminClient.Create(ctx, testPluginDefinition)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			By("Deleting the plugin definition")
			test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
		})

		By("Checking the plugin definition is ready")
		pluginDefinitionList := &greenhousev1alpha1.ClusterPluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Checking the helmrepository is ready")
		Eventually(func(g Gomega) {
			helmRepositoryList := &sourcecontroller.HelmRepositoryList{}
			err = adminClient.List(ctx, helmRepositoryList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(helmRepositoryList.Items)).To(BeEquivalentTo(1))
			g.Expect(helmRepositoryList.Items[0].Spec.URL).To(Equal("oci://ghcr.io/stefanprodan/charts"), "the helm repository URL should match the expected value")
		})

		By("Prepare the plugin")
		testPlugin := fixtures.PreparePlugin("test-podinfo-plugin", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithReleaseName("test-podinfo-plugin"),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil))

		By("Add labels to remote cluster")
		remoteCluster := &greenhousev1alpha1.Cluster{}
		err = adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
		Expect(err).ToNot(HaveOccurred())
		remoteCluster.Labels = map[string]string{
			"app": "test-cluster",
		}
		err = adminClient.Update(ctx, remoteCluster)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the plugin preset")
		testPluginPreset := &greenhousev1alpha1.PluginPreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-podinfo-plugin-preset",
				Namespace: env.TestNamespace,
				Labels: map[string]string{
					"greenhouse.sap/deployment-tool": "flux",
				},
				Annotations: map[string]string{
					"greenhouse.sap/propagate-labels": "greenhouse.sap/deployment-tool",
				},
			},
			Spec: greenhousev1alpha1.PluginPresetSpec{
				Plugin: testPlugin.Spec,
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-cluster",
					},
				},
			},
		}
		err = adminClient.Create(ctx, testPluginPreset)
		Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

		DeferCleanup(func() {
			By("Deleting the plugin preset")
			test.EventuallyDeleted(ctx, adminClient, testPluginPreset)
		})

		By("Checking the plugin has been created")
		Eventually(func(g Gomega) {
			err = adminClient.Get(ctx,
				types.NamespacedName{
					Name:      testPluginPreset.Name + "-" + remoteClusterName,
					Namespace: env.TestNamespace,
				}, testPlugin)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(testPlugin.Labels).To(HaveKeyWithValue("greenhouse.sap/deployment-tool", "flux"), "plugin should have the greenhouse.sap/deployment-tool label set to flux")
		}).Should(Succeed())

		By("Checking the helmRelease is created")
		Eventually(func(g Gomega) {
			helmReleaseList := &helmcontroller.HelmReleaseList{}
			err = adminClient.List(ctx, helmReleaseList)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(helmReleaseList.Items).To(HaveLen(1))
		}).Should(Succeed())

		By("Checking the deployment is created on the remote cluster")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1), "there should be exactly one deployment")
			g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(1))), "the deployment should have 1 replica")
		}).Should(Succeed())

		By("Checking the HelmRelease Ready condition is True")
		Eventually(func(g Gomega) {
			helmRelease := &helmcontroller.HelmRelease{}
			err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPlugin), helmRelease)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the HelmRelease")
			releaseReady := meta.FindStatusCondition(helmRelease.Status.Conditions, fluxmeta.ReadyCondition)
			g.Expect(releaseReady).ToNot(BeNil(), "HelmRelease Ready condition must be set")
			g.Expect(helmRelease.Status.ObservedGeneration).To(BeNumerically(">=", helmRelease.Generation), "HelmRelease status must be current")
			g.Expect(releaseReady.Status).To(Equal(metav1.ConditionTrue), "HelmRelease Ready condition must be true")
			g.Expect(releaseReady.Reason).To(Equal("InstallSucceeded"), "HelmRelease Ready condition should have the correct Reason")
		}).Should(Succeed())

		By("ensuring Plugin status has been updated")
		Eventually(func(g Gomega) {
			err := adminClient.Get(ctx, client.ObjectKeyFromObject(testPlugin), testPlugin)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get the Plugin")

			clusterAccess := testPlugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
			g.Expect(clusterAccess).ToNot(BeNil(), "Plugin clusterAccess condition must be set")
			g.Expect(clusterAccess.Status).To(Equal(metav1.ConditionTrue), "Plugin clusterAccess condition must be true")

			reconcileFailed := testPlugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition)
			g.Expect(reconcileFailed).ToNot(BeNil(), "Plugin reconcileFailed condition must be set")
			g.Expect(reconcileFailed.Status).To(Equal(metav1.ConditionFalse), "Plugin reconcileFailed condition must be false")

			ready := testPlugin.Status.StatusConditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(ready).ToNot(BeNil(), "Plugin Ready condition must be set")
			g.Expect(ready.Status).To(Equal(metav1.ConditionTrue), "Plugin Ready condition must be true")

			statusUpToDate := testPlugin.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.StatusUpToDateCondition)
			g.Expect(statusUpToDate).ToNot(BeNil(), "Plugin StatusUpToDate condition must be set")
			g.Expect(statusUpToDate.Status).To(Equal(metav1.ConditionTrue), "Plugin statusUpToDate condition must be true")

			g.Expect(testPlugin.Status.ExposedServices).To(BeEmpty(), "exposed services in plugin status should be empty")
		}).Should(Succeed())
	})
})
