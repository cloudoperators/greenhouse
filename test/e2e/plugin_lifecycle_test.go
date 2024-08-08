// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
	"github.com/cloudoperators/greenhouse/test/e2e/fixtures"
)

var _ = Describe("PluginLifecycle", Ordered, func() {
	It("should deploy the plugin", func() {
		testNamespace := &v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: test.TestNamespace,
			},
			Spec: v1.NamespaceSpec{},
		}

		testPluginDefinition := fixtures.NginxPluginDefinition

		testPlugin := &greenhousev1alpha1.Plugin{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Plugin",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nginx-plugin",
				Namespace: test.TestNamespace,
			},
			Spec: greenhousev1alpha1.PluginSpec{
				PluginDefinition: "nginx",
				ReleaseNamespace: test.TestNamespace,
			},
		}

		pluginDefinitionList := &greenhousev1alpha1.PluginDefinitionList{}
		pluginList := &greenhousev1alpha1.PluginList{}
		podList := &v1.PodList{}

		err := remoteClient.Create(test.Ctx, testNamespace)
		Expect(err).NotTo(HaveOccurred())

		err = remoteClient.Create(test.Ctx, testPluginDefinition)
		Expect(err).NotTo(HaveOccurred())

		err = remoteClient.List(test.Ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(1))

		err = remoteClient.Create(test.Ctx, testPlugin)
		Expect(err).NotTo(HaveOccurred())

		err = remoteClient.List(test.Ctx, pluginList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginList.Items)).To(BeEquivalentTo(1))

		err = remoteClient.Delete(test.Ctx, testPlugin)
		Expect(err).NotTo(HaveOccurred())

		err = remoteClient.List(test.Ctx, pluginList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginList.Items)).To(BeEquivalentTo(0))

		err = remoteClient.Delete(test.Ctx, testPluginDefinition)
		Expect(err).NotTo(HaveOccurred())

		err = remoteClient.List(test.Ctx, pluginDefinitionList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pluginDefinitionList.Items)).To(BeEquivalentTo(0))

		err = remoteClient.List(test.Ctx, podList, client.InNamespace(test.TestNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(podList.Items)).To(BeEquivalentTo(0))
	})
})
