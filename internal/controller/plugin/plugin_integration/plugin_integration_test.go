// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_integration

import (
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const testNamespace = "greenhouse"

var (
	testPluginDefinition *greenhousev1alpha1.ClusterPluginDefinition
	alerts               *greenhousev1alpha1.Plugin
	kubeMonitoring       *greenhousev1alpha1.Plugin
)

var _ = Describe("Plugin Integration", Ordered, func() {
	BeforeAll(func() {
		By("creating greenhouse organization")
		org := test.NewOrganization(test.Ctx, testNamespace)
		err := test.K8sClient.Create(test.Ctx, org)
		Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the organization")
		ns := &corev1.Namespace{}
		ns.SetName(testNamespace)
		err = test.K8sClient.Create(test.Ctx, ns)
		Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the namespace")

		testPluginDefinition = test.NewClusterPluginDefinition(
			test.Ctx,
			"test-tracking-plugindefinition",
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:       "dummy",
				Repository: "oci://greenhouse/helm-charts",
				Version:    "1.0.0",
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, testPluginDefinition)).Should(Succeed(), "there should be no error creating the pluginDefinition")

		By("mocking HelmChart Ready condition for testPluginDefinition")
		Eventually(func(g Gomega) {
			helmChart := &sourcecontroller.HelmChart{}
			helmChart.SetName(testPluginDefinition.FluxHelmChartResourceName())
			helmChart.SetNamespace(flux.HelmRepositoryDefaultNamespace)
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(helmChart), helmChart)
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmChart")

			newHelmChart := &sourcecontroller.HelmChart{}
			*newHelmChart = *helmChart
			helmChartReadyCondition := metav1.Condition{
				Type:               fluxmeta.ReadyCondition,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Succeeded",
				Message:            "Helm chart is ready",
			}
			newHelmChart.Status.Conditions = []metav1.Condition{helmChartReadyCondition}
			g.Expect(test.K8sClient.Status().Patch(test.Ctx, newHelmChart, client.MergeFrom(helmChart))).To(Succeed(), "there should be no error patching HelmChart status")
		}).Should(Succeed(), "HelmChart should be mocked as ready")

		By("waiting for ClusterPluginDefinition to have a Ready condition")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the ClusterPluginDefinition")
			readyCondition := testPluginDefinition.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
			g.Expect(readyCondition).ToNot(BeNil(), "the ClusterPluginDefinition should have a Ready condition")
			g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "ClusterPluginDefinition should be Ready")
		}).Should(Succeed(), "the ClusterPluginDefinition should become Ready")
	})

	AfterAll(func() {
		By("cleaning up test plugins")
		test.EventuallyDeleted(test.Ctx, test.K8sClient, kubeMonitoring)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, alerts)
		test.EventuallyDeleted(test.Ctx, test.K8sClient, testPluginDefinition)
	})

})
