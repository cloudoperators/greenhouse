// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
	"github.com/cloudoperators/greenhouse/test/e2e/fixtures"
)

var _ = Describe("PluginLifecycle", Ordered, func() {
	Context("without helm hook", func() {
		It("should deploy the plugin", func() {

			const clusterName = "test-cluster-a"
			setup := test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
			secret := setup.CreateSecret(test.Ctx, clusterName,
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}))
			cluster := &greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: setup.Namespace(),
				},
			}
			test.EventuallyCreated(test.Ctx, test.K8sClient, cluster)

			pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
			pluginList := &greenhousev1alpha1.PluginList{}
			deploymentList := &appsv1.DeploymentList{}
			ctx := test.Ctx

			// Creating plugin definition
			testPluginDefinition := fixtures.CreateNginxPluginDefinition(ctx, setup)
			err := test.K8sClient.List(ctx, pluginDefinitionList)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

			// Creating plugin
			testPlugin := setup.CreatePlugin(test.Ctx, "test-nginx-plugin",
				test.WithPluginDefinition(testPluginDefinition.Name),
				test.WithCluster(secret.Name),
				test.WithReleaseNamespace(setup.Namespace()),
				test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil))
			Eventually(func(g Gomega) bool {
				err = test.K8sClient.List(ctx, pluginList)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus).ToNot(BeNil())
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
				return true
			}).Should(BeTrue())

			// Checking deployment
			err = remoteClient.List(ctx, deploymentList, client.InNamespace(setup.Namespace()))
			Expect(err).NotTo(HaveOccurred())
			SetDefaultEventuallyTimeout(60 * time.Second)
			Eventually(func(g Gomega) bool {
				err = remoteClient.List(ctx, deploymentList, client.InNamespace(setup.Namespace()))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(deploymentList.Items).ToNot(BeEmpty())
				return true
			}).Should(BeTrue())

			// Checking the name of deployment
			nginxDeploymentExists := false
			for _, deployment := range deploymentList.Items {
				if strings.Contains(deployment.Name, "nginx") {
					nginxDeploymentExists = true
					Expect(deployment.Spec.Replicas).To(PointTo(Equal(int32(1))))
					break
				}
			}
			Expect(nginxDeploymentExists).To(BeTrue())

			// Updating replicas
			namespacedName := types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}
			err = test.K8sClient.Get(ctx, namespacedName, testPlugin)
			Expect(err).NotTo(HaveOccurred())
			testPlugin = &pluginList.Items[0]
			test.SetOptionValueForPlugin(testPlugin, "replicaCount", "2")
			err = test.K8sClient.Update(ctx, testPlugin)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) bool {
				err = remoteClient.List(ctx, deploymentList, client.InNamespace(setup.Namespace()))
				g.Expect(err).NotTo(HaveOccurred())
				for _, deployment := range deploymentList.Items {
					if strings.Contains(deployment.Name, "nginx") {
						g.Expect(deployment.Spec.Replicas).To(PointTo(Equal(int32(2))))
					}
				}
				return true
			}).Should(BeTrue())

			// Deleting plugin
			test.EventuallyDeleted(ctx, test.K8sClient, testPlugin)

			// Check, is deployment deleted
			Eventually(func(g Gomega) bool {
				err = remoteClient.List(ctx, deploymentList, client.InNamespace(setup.Namespace()))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(deploymentList.Items)).To(BeEquivalentTo(0))
				return true
			}).Should(BeTrue())

			// Deleting plugin definition
			test.EventuallyDeleted(ctx, test.K8sClient, testPluginDefinition)
		})
	})

	Context("with helm lifecycle hooks", func() {
		It("should deploy the plugin", func() {

			const clusterName = "test-cluster-b"
			setup := test.NewTestSetup(test.Ctx, test.K8sClient, "teamrbac")
			secret := setup.CreateSecret(test.Ctx, clusterName,
				test.WithSecretType(greenhouseapis.SecretTypeKubeConfig),
				test.WithSecretData(map[string][]byte{greenhouseapis.KubeConfigKey: remoteKubeConfig}))
			cluster := &greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: setup.Namespace(),
				},
			}
			test.EventuallyCreated(test.Ctx, test.K8sClient, cluster)

			pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
			pluginList := &greenhousev1alpha1.PluginList{}
			podList := &v1.PodList{}
			ctx := test.Ctx

			// Creating plugin definition
			testPluginDefinition := fixtures.CreateTestHookPluginDefinition(test.Ctx, setup)
			err := test.K8sClient.List(ctx, pluginDefinitionList)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

			// Creating plugin
			_ = setup.CreatePlugin(test.Ctx, "test-hook-plugin",
				test.WithPluginDefinition(testPluginDefinition.Name),
				test.WithCluster(secret.Name),
				test.WithReleaseNamespace(setup.Namespace()))

			// Check jobs
			jobList := &batchv1.JobList{}
			err = remoteClient.List(ctx, jobList, client.InNamespace(setup.Namespace()))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobList.Items)).To(BeEquivalentTo(0))

			// Check plugin list
			var plugin greenhousev1alpha1.Plugin
			Eventually(func(g Gomega) {
				err = test.K8sClient.List(ctx, pluginList)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus).ToNot(BeNil())
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
				plugin = pluginList.Items[0]
			}).Should(Succeed())

			// Checking pod
			err = remoteClient.List(ctx, podList, client.InNamespace(setup.Namespace()))
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				err = remoteClient.List(ctx, podList, client.InNamespace(setup.Namespace()))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(podList.Items).ToNot(BeEmpty())
			}).Should(Succeed())

			// Checking the name of pod
			podExists := false
			for _, pod := range podList.Items {
				if strings.Contains(pod.Name, "alpine") {
					podExists = true
					break
				}
			}
			Expect(podExists).To(BeTrue())

			// Update plugin value
			test.SetOptionValueForPlugin(&plugin, "hook_enabled", "true")
			err = test.K8sClient.Update(ctx, &plugin)
			Expect(err).NotTo(HaveOccurred())

			// Check jobs
			Eventually(func(g Gomega) {
				err = remoteClient.List(ctx, jobList, client.InNamespace(setup.Namespace()))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(jobList.Items)).To(BeEquivalentTo(1))
			}).Should(Succeed())

		})
	})
})
