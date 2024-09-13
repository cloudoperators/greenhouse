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
	Context("without webhook", func() {
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

			testPluginDefinition := fixtures.NginxPluginDefinition
			testPluginDefinition.ObjectMeta.Namespace = setup.Namespace() // namespace override

			testPlugin := &greenhousev1alpha1.Plugin{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Plugin",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nginx-plugin",
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "nginx-18.1.7",
					ReleaseNamespace: setup.Namespace(),
					ClusterName:      secret.Name,
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "replicaCount",
							Value: &apiextensionsv1.JSON{Raw: []byte("1")},
						},
					},
				},
				Status: greenhousev1alpha1.PluginStatus{},
			}

			pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
			pluginList := &greenhousev1alpha1.PluginList{}
			deploymentList := &appsv1.DeploymentList{}
			ctx := test.Ctx

			// Creating plugin definition
			err := test.K8sClient.Create(ctx, testPluginDefinition)
			Expect(err).NotTo(HaveOccurred())
			err = test.K8sClient.List(ctx, pluginDefinitionList)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

			// Creating plugin
			err = test.K8sClient.Create(ctx, testPlugin)
			Expect(err).NotTo(HaveOccurred())
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
		// TODO: This test must not rely on index value, but on OptionValue.Name
		// A helper method to get and set an OptionValue by name should be introduced.
		testPlugin.Spec.OptionValues[10].Value.Raw = []byte("2")
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

	Context("with webhooks", func() {
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

			testPluginDefinition := fixtures.TestHookPluginDefinition
			testPluginDefinition.ObjectMeta.Namespace = setup.Namespace() // namespace override

			testPlugin := &greenhousev1alpha1.Plugin{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Plugin",
					APIVersion: greenhousev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hook-plugin",
					Namespace: setup.Namespace(),
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: testPluginDefinition.Name,
					ReleaseNamespace: setup.Namespace(),
					ClusterName:      secret.Name,
					OptionValues:     []greenhousev1alpha1.PluginOptionValue{},
				},
				Status: greenhousev1alpha1.PluginStatus{},
			}

			pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
			pluginList := &greenhousev1alpha1.PluginList{}
			podList := &v1.PodList{}
			ctx := test.Ctx

			// Creating plugin definition
			err := test.K8sClient.Create(ctx, testPluginDefinition)
			Expect(err).NotTo(HaveOccurred())
			err = test.K8sClient.List(ctx, pluginDefinitionList)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

			// Creating plugin
			err = test.K8sClient.Create(ctx, testPlugin)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				err = test.K8sClient.List(ctx, pluginList)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus).ToNot(BeNil())
				g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
			}).Should(Succeed())

			// Checking pod
			err = remoteClient.List(ctx, podList, client.InNamespace(setup.Namespace()))
			Expect(err).NotTo(HaveOccurred())
			SetDefaultEventuallyTimeout(60 * time.Second)
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
		})
	})
})
