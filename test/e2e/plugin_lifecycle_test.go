// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
	"github.com/cloudoperators/greenhouse/test/e2e/fixtures"
)

var _ = Describe("PluginLifecycle", Ordered, func() {
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
						Name:  "containerSecurityContext.enabled",
						Value: &apiextensionsv1.JSON{Raw: []byte("false")},
					},
					{
						Name:  "podSecurityContext.enabled",
						Value: &apiextensionsv1.JSON{Raw: []byte("false")},
					},
					{
						Name:  "autoscaling.minReplicas",
						Value: &apiextensionsv1.JSON{Raw: []byte("\"2\"")},
					},
					{
						Name:  "autoscaling.maxReplicas",
						Value: &apiextensionsv1.JSON{Raw: []byte("\"4\"")},
					},
					{
						Name:  "autoscaling.enabled",
						Value: &apiextensionsv1.JSON{Raw: []byte("true")},
					},
				},
			},
		}

		pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
		pluginList := &greenhousev1alpha1.PluginList{}
		podList := &v1.PodList{}

		err := test.K8sClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).NotTo(HaveOccurred())

		err = test.K8sClient.List(test.Ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		err = test.K8sClient.Create(test.Ctx, testPlugin)
		Expect(err).NotTo(HaveOccurred())

		err = test.K8sClient.List(test.Ctx, pluginList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginList.Items)).To(BeEquivalentTo(1))

		Eventually(func(g Gomega) bool {
			err = test.K8sClient.List(test.Ctx, podList, client.InNamespace(setup.Namespace()))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(podList.Items) > 0).To(BeTrue())
			return true
		}).Should(BeTrue())

		err = test.K8sClient.Delete(test.Ctx, testPlugin)
		Expect(err).NotTo(HaveOccurred())

		err = test.K8sClient.List(test.Ctx, pluginList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginList.Items)).To(BeEquivalentTo(0))

		err = test.K8sClient.Delete(test.Ctx, testPluginDefinition)
		Expect(err).NotTo(HaveOccurred())

		err = test.K8sClient.List(test.Ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(0))

		Eventually(func(g Gomega) bool {
			err = remoteClient.List(test.Ctx, podList, client.InNamespace(setup.Namespace()))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(podList.Items)).To(BeEquivalentTo(0))
			return true
		}).Should(BeTrue())
	})
})
