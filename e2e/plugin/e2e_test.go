// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build pluginE2E

package plugin

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

const remoteClusterName = "remote-plugin-cluster"

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
		Expect(err).ToNot(HaveOccurred())

		By("Checking the plugin definition is ready")
		pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
		err = adminClient.List(ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		By("Creating the plugin")
		// Creating plugin
		testPlugin := fixtures.PreparePlugin("test-nginx-plugin", env.TestNamespace,
			test.WithPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(remoteClusterName),
			test.WithReleaseNamespace(env.TestNamespace),
			test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil))
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
})
