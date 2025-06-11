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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
	testStartTime = time.Now().UTC()
})

var _ = AfterSuite(func() {
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)
	env.GenerateControllerLogs(ctx, testStartTime)
})

var _ = Describe("Plugin E2E", Ordered, func() {
	It("should onboard remote cluster", func() {
		By("onboarding remote cluster")
		shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace)
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
		pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Creating the plugin")
		// Creating plugin with release name
		testPlugin := fixtures.PreparePlugin("test-nginx-plugin-1", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(remoteClusterName),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
			test.WithReleaseName("test-nginx-plugin-1"),
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
		pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Prepare the plugin")
		// Creating plugin with release name
		testPlugin := fixtures.PreparePlugin("test-nginx-plugin-2", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
			test.WithReleaseName("test-nginx-plugin-2"),
		)

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
				Name:      "test-nginx-plugin-preset",
				Namespace: env.TestNamespace,
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
		Expect(err).ToNot(HaveOccurred())

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

		By("Update plugin preset with cluster overview")
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred())
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
		Expect(err).ToNot(HaveOccurred())

		By("Check the replicas in deployment")
		Eventually(func(g Gomega) {
			deploymentList := &appsv1.DeploymentList{}
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(env.TestNamespace))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(1))
			g.Expect(deploymentList.Items[0].Spec.Replicas).To(PointTo(Equal(int32(2))))
		}).Should(Succeed())

		By("Update plugin preset with cluster option override")
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
		os.Setenv("HELM_RELEASE_TIMEOUT", "5")

		By("Preparing the plugin")
		// Creating plugin with release name
		plugin := fixtures.PreparePlugin("test-cert-manager-plugin", env.TestNamespace,
			test.WithPluginDefinition(pluginDefinition.Name),
			test.WithCluster(remoteClusterName),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithReleaseName("test-cert-manager-plugin"),
		)

		By("Installing release manually on the remote cluster")
		_, err = helm.ExportInstallHelmRelease(ctx, adminClient, env.RemoteRestClientGetter, pluginDefinition, plugin, false)
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
		os.Setenv("HELM_RELEASE_TIMEOUT", "300")

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
})
