// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin_integration

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcecontroller "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
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

	It("should create HelmReleases and resolve values from external references", func() {
		By("creating alerts plugin with a label")
		alerts = test.NewPlugin(test.Ctx, "alerts", testNamespace,
			test.WithClusterPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(""),
			test.WithReleaseName("release-alerts"),
			test.WithPluginLabel("test-label", "test-value"),
			test.WithPluginOptionValue("trackedOption", test.MustReturnJSONFor("trackedValue")),
		)
		Expect(test.K8sClient.Create(test.Ctx, alerts)).To(Succeed(), "failed to create alerts plugin")

		By("creating kube-monitoring plugin that resolves value from alerts")
		kubeMonitoring = test.NewPlugin(test.Ctx, "kube-monitoring", testNamespace,
			test.WithClusterPluginDefinition(testPluginDefinition.Name),
			test.WithCluster(""),
			test.WithReleaseName("release-kube-monitoring"),
			test.WithPluginOptionValueFromRef("resolvedOption", &greenhousev1alpha1.ExternalValueSource{
				Name:       "alerts",
				Expression: "object.spec.optionValues[0].value",
			}),
		)
		Expect(test.K8sClient.Create(test.Ctx, kubeMonitoring)).To(Succeed(), "failed to create kube-monitoring plugin")

		By("verifying HelmRelease was created for alerts")
		alertsRelease := &helmv2.HelmRelease{}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "alerts", Namespace: testNamespace}, alertsRelease)
			g.Expect(err).ToNot(HaveOccurred(), "HelmRelease for alerts should exist")
		}).Should(Succeed())

		By("verifying HelmRelease was created for kube-monitoring")
		kubeMonitoringRelease := &helmv2.HelmRelease{}
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "kube-monitoring", Namespace: testNamespace}, kubeMonitoringRelease)
			g.Expect(err).ToNot(HaveOccurred(), "HelmRelease for kube-monitoring should exist")
		}).Should(Succeed())

		By("verifying kube-monitoring resolved the value from alerts")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "kube-monitoring", Namespace: testNamespace}, kubeMonitoringRelease)
			g.Expect(err).ToNot(HaveOccurred(), "HelmRelease for kube-monitoring should exist")

			// Check that the HelmRelease contains the resolved value from alerts in its inline values
			g.Expect(kubeMonitoringRelease.Spec.Values).ToNot(BeNil(), "HelmRelease should have inline values")
			valuesRaw := string(kubeMonitoringRelease.Spec.Values.Raw)
			g.Expect(valuesRaw).To(ContainSubstring("trackedValue"), "resolved value should contain 'trackedValue' from alerts plugin")
		}).Should(Succeed())
	})

	It("should track and untrack plugin dependencies", func() {
		By("verifying tracking annotation was set on alerts")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(alerts), alerts)
			g.Expect(err).ToNot(HaveOccurred())
			annotations := alerts.GetAnnotations()
			g.Expect(annotations).To(HaveKey(greenhouseapis.AnnotationKeyPluginTackingID))
			g.Expect(annotations[greenhouseapis.AnnotationKeyPluginTackingID]).To(ContainSubstring("Plugin/kube-monitoring"))
		}).Should(Succeed(), "tracking annotation should be set on alerts")

		By("verifying trackedObjects in kube-monitoring status")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(kubeMonitoring), kubeMonitoring)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(kubeMonitoring.Status.TrackedObjects).To(ContainElement("Plugin/alerts"))
		}).Should(Succeed(), "trackedObjects should contain alerts")

		By("updating kube-monitoring to remove valueFrom and use a plain value")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(kubeMonitoring), kubeMonitoring)
			g.Expect(err).ToNot(HaveOccurred())

			// Remove the valueFrom option and replace with plain value
			kubeMonitoring.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "resolvedOption",
					Value: test.MustReturnJSONFor("plainValue"),
				},
			}
			err = test.K8sClient.Update(test.Ctx, kubeMonitoring)
			g.Expect(err).ToNot(HaveOccurred())
		}).Should(Succeed(), "should update kube-monitoring spec")

		By("verifying trackedObjects in kube-monitoring status is empty")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(kubeMonitoring), kubeMonitoring)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(kubeMonitoring.Status.TrackedObjects).To(BeEmpty())
		}).Should(Succeed(), "trackedObjects should be empty after removing valueFrom")

		By("verifying tracking annotation was removed from alerts")
		Eventually(func(g Gomega) {
			err := test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(alerts), alerts)
			g.Expect(err).ToNot(HaveOccurred())
			annotations := alerts.GetAnnotations()
			if trackingID, ok := annotations[greenhouseapis.AnnotationKeyPluginTackingID]; ok {
				g.Expect(trackingID).ToNot(ContainSubstring("Plugin/kube-monitoring"))
			}
		}).Should(Succeed(), "tracking annotation should be removed from alerts")
	})
})
