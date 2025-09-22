// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"slices"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	PluginDefinitionName           = "my-test-plugin"
	UIPluginDefinitionName         = "my-test-ui-plugin"
	PluginDefinitionVersion        = "1.0.0"
	PluginDefinitionVersionUpdated = "1.1.0"
	PluginDefinitionChartVersion   = "1.0.0"

	PluginOptionRequired     = "myRequiredOption"
	PluginOptionOptional     = "myOptionalOption"
	PluginOptionDefault      = "myDefaultOption"
	PluginOptionDefaultValue = "myDefaultValue"

	HelmRepo  = "https://my.dummy.io"
	HelmChart = "./../../test/fixtures/myChart"
)

func mockPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()

	clusterDef := test.NewClusterPluginDefinition(test.Ctx, PluginDefinitionName,
		test.WithVersion(PluginDefinitionVersion),
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       HelmChart,
			Repository: HelmRepo,
			Version:    PluginDefinitionChartVersion,
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionRequired,
			Description: "This is my required test plugin option",
			Required:    true,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionOptional,
			Description: "This is my optional test plugin option",
			Required:    false,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionDefault,
			Description: "This is my default test plugin option",
			Required:    false,
			Default:     test.MustReturnJSONFor(PluginOptionDefaultValue),
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
	)
	pluginDef := &greenhousev1alpha1.PluginDefinition{}
	pluginDef.SetName(clusterDef.Name)
	pluginDef.Spec = clusterDef.Spec
	return pluginDef
}

func mockUIPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	clusterDef := test.NewClusterPluginDefinition(test.Ctx, UIPluginDefinitionName, test.AppendPluginOption(
		greenhousev1alpha1.PluginOption{
			Name:    "test-plugin-definition-option-1",
			Type:    "int",
			Default: &apiextensionsv1.JSON{Raw: []byte("1")}},
	),
		test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
			Name:    "test-ui-app",
			Version: "0.0.1",
		}),
		test.WithoutHelmChart(),
	)
	pluginDefinition := &greenhousev1alpha1.PluginDefinition{}
	pluginDefinition.SetName(clusterDef.Name)
	pluginDefinition.Spec = clusterDef.Spec
	return pluginDefinition
}

func listEvents(involvedObjectName string) *corev1.EventList {
	GinkgoHelper()
	events := &corev1.EventList{}
	err := test.K8sClient.List(test.Ctx, events, cl.InNamespace(corev1.NamespaceDefault), cl.MatchingFields{"involvedObject.name": involvedObjectName})
	Expect(err).ToNot(HaveOccurred(), "there should be no error listing events")
	return events
}

var _ = Describe("PluginDefinition controller", func() {
	var (
		remoteEnvTest *envtest.Environment
	)
	BeforeEach(func() {
		_, _, remoteEnvTest, _ = test.StartControlPlane("6885", false, false)
	})
	AfterEach(func() {
		Expect(remoteEnvTest.Stop()).To(Succeed(), "there should be no error stopping the remote environment")
	})
	Context("When creating or updating a PluginDefinition", Ordered, func() {
		It("should successfully create or update a ClusterPluginDefinition from PluginDefinition", func() {
			By("creating a PluginDefinition")
			pluginDef := mockPluginDefinition()
			err := test.K8sClient.Create(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginDefinition")

			clusterDef := new(greenhousev1alpha1.ClusterPluginDefinition)
			clusterDef.SetName(pluginDef.Name)

			By("checking if the PluginDefinition is Ready and ClusterPluginDefinition is created")
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
				readyCondition := pluginDef.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "the PluginDefinition should have a Ready condition")
				err = test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the ClusterPluginDefinition")
				g.Expect(pluginDef.Spec).To(Equal(clusterDef.Spec), "the PluginDefinition Spec should match the ClusterPluginDefinition Spec")
				return nil
			}).Should(Succeed(), "the PluginDefinition should be created successfully")

			By("checking if the Created event is present for the ClusterPluginDefinition")
			events := listEvents(clusterDef.Name)
			Expect(events.Items).ToNot(BeEmpty(), "there should be at least one event for the ClusterPluginDefinition")
			createdEvent := slices.ContainsFunc(events.Items, func(event corev1.Event) bool {
				return event.Reason == "Created" && event.InvolvedObject.Name == clusterDef.Name
			})
			Expect(createdEvent).To(BeTrue(), "there should be a Created event for the ClusterPluginDefinition")

			By("updating the PluginDefinition")
			pluginDef.Spec.Version = PluginDefinitionVersionUpdated
			err = test.K8sClient.Update(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error updating the PluginDefinition")

			By("checking if the ClusterPluginDefinition is updated")
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the ClusterPluginDefinition")
				g.Expect(clusterDef.Spec.Version).To(Equal(pluginDef.Spec.Version), "the ClusterPluginDefinition version should be updated")
				return nil
			}).Should(Succeed(), "the ClusterPluginDefinition should be updated successfully")

			By("checking if the Updated event is present for the ClusterPluginDefinition")
			events = listEvents(clusterDef.Name)
			Expect(events.Items).ToNot(BeEmpty(), "there should be at least one event for the ClusterPluginDefinition")
			updatedEvent := slices.ContainsFunc(events.Items, func(event corev1.Event) bool {
				return event.Reason == "Updated" && event.InvolvedObject.Name == clusterDef.Name
			})
			Expect(updatedEvent).To(BeTrue(), "there should be an Updated event for the ClusterPluginDefinition")

			By("checking if flux HelmRepository is created")
			repositoryURL := flux.ChartURLToName(HelmRepo)
			repository := &sourcev1.HelmRepository{}
			repository.SetName(repositoryURL)
			repository.SetNamespace(test.TestGreenhouseNamespace)
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(repository), repository)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmRepository")
				g.Expect(repository.Spec.URL).To(Equal(HelmRepo), "the HelmRepository URL should match the PluginDefinition repository URL")
				return nil
			}).Should(Succeed(), "the HelmRepository should be created successfully")
		})
		It("should successfully create a ClusterPluginDefinition for a UI PluginDefinition", func() {
			By("creating a PluginDefinition")
			pluginDef := mockUIPluginDefinition()
			err := test.K8sClient.Create(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginDefinition")

			clusterDef := new(greenhousev1alpha1.ClusterPluginDefinition)
			clusterDef.SetName(pluginDef.Name)

			By("checking if the PluginDefinition is Ready and ClusterPluginDefinition is created")
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
				readyCondition := pluginDef.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "the PluginDefinition should have a Ready condition")
				err = test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the ClusterPluginDefinition")
				g.Expect(pluginDef.Spec).To(Equal(clusterDef.Spec), "the PluginDefinition Spec should match the ClusterPluginDefinition Spec")
				return nil
			}).Should(Succeed(), "the PluginDefinition should be created successfully")

			By("checking if the Skipped event is present for the ClusterPluginDefinition")
			events := listEvents(clusterDef.Name)
			Expect(events.Items).ToNot(BeEmpty(), "there should be at least one event for the ClusterPluginDefinition")
			skippedEvent := slices.ContainsFunc(events.Items, func(event corev1.Event) bool {
				return event.Reason == "Skipped" && event.InvolvedObject.Name == clusterDef.Name
			})
			Expect(skippedEvent).To(BeTrue(), "there should be a Skipped event for the ClusterPluginDefinition")
		})
	})
})
