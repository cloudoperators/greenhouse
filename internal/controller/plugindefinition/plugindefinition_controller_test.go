// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	PluginDefinitionName         = "my-test-plugin"
	UIPluginDefinitionName       = "my-test-ui-plugin"
	PluginDefinitionVersion      = "1.0.0"
	PluginDefinitionChartVersion = "1.0.0"

	PluginOptionRequired     = "myRequiredOption"
	PluginOptionOptional     = "myOptionalOption"
	PluginOptionDefault      = "myDefaultOption"
	PluginOptionDefaultValue = "myDefaultValue"

	HelmRepo  = "https://my.dummy.io"
	HelmChart = "./../../test/fixtures/myChart"
)

func mockClusterPluginDefinition() *greenhousev1alpha1.ClusterPluginDefinition {
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
	return clusterDef
}
func mockPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()

	pluginDef := test.NewPluginDefinition(test.Ctx, PluginDefinitionName, test.TestNamespace,
		test.WithPluginDefinitionVersion(PluginDefinitionVersion),
		test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       HelmChart,
			Repository: HelmRepo,
			Version:    PluginDefinitionChartVersion,
		}),
		test.AppendPluginDefinitionPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionRequired,
			Description: "This is my required test plugin option",
			Required:    true,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
		test.AppendPluginDefinitionPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionOptional,
			Description: "This is my optional test plugin option",
			Required:    false,
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
		test.AppendPluginDefinitionPluginOption(greenhousev1alpha1.PluginOption{
			Name:        PluginOptionDefault,
			Description: "This is my default test plugin option",
			Required:    false,
			Default:     test.MustReturnJSONFor(PluginOptionDefaultValue),
			Type:        greenhousev1alpha1.PluginOptionTypeString,
		}),
	)
	return pluginDef
}

func mockUIPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	pluginDef := test.NewPluginDefinition(test.Ctx, UIPluginDefinitionName, test.TestNamespace,
		test.AppendPluginDefinitionPluginOption(
			greenhousev1alpha1.PluginOption{
				Name:    "test-plugin-definition-option-1",
				Type:    "int",
				Default: &apiextensionsv1.JSON{Raw: []byte("1")}},
		),
		test.WithPluginDefinitionUIApplication(&greenhousev1alpha1.UIApplicationReference{
			Name:    "test-ui-app",
			Version: "0.0.1",
		}),
		test.WithoutPluginDefinitionHelmChart(),
	)
	return pluginDef
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

	Context("When creating a PluginDefinition", Ordered, func() {
		It("should successfully create or update a HelmRepository from PluginDefinition", func() {
			By("creating a PluginDefinition")
			pluginDef := mockPluginDefinition()
			err := test.K8sClient.Create(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginDefinition")

			By("checking if the PluginDefinition is Ready")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
				readyCondition := pluginDef.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "the PluginDefinition should have a Ready condition")
			}).Should(Succeed(), "the PluginDefinition should be ready")

			By("checking if flux HelmRepository is created")
			repositoryURL := flux.ChartURLToName(HelmRepo)
			repository := &sourcev1.HelmRepository{}
			repository.SetName(repositoryURL)
			repository.SetNamespace(pluginDef.GetNamespace())
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(repository), repository)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmRepository")
				g.Expect(repository.Spec.URL).To(Equal(HelmRepo), "the HelmRepository URL should match the PluginDefinition repository URL")
				return nil
			}).Should(Succeed(), "the HelmRepository should be created successfully")
		})
		It("should successfully create a HelmRepository for a UI PluginDefinition", func() {
			By("creating a PluginDefinition")
			pluginDef := mockUIPluginDefinition()
			err := test.K8sClient.Create(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginDefinition")

			By("checking if the PluginDefinition is Ready")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
				readyCondition := pluginDef.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "the PluginDefinition should have a Ready condition")
			}).Should(Succeed(), "the PluginDefinition should be ready")

			By("checking if flux HelmRepository is created")
			repositoryURL := flux.ChartURLToName(HelmRepo)
			repository := &sourcev1.HelmRepository{}
			repository.SetName(repositoryURL)
			repository.SetNamespace(pluginDef.GetNamespace())
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(repository), repository)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmRepository")
				g.Expect(repository.Spec.URL).To(Equal(HelmRepo), "the HelmRepository URL should match the PluginDefinition repository URL")
				return nil
			}).Should(Succeed(), "the HelmRepository should be created successfully")
		})
	})

	Context("When creating a ClusterPluginDefinition", Ordered, func() {
		It("should successfully create a HelmRepository from ClusterPluginDefinition", func() {
			By("creating a ClusterPluginDefinition")
			clusterDef := mockClusterPluginDefinition()
			err := test.K8sClient.Create(test.Ctx, clusterDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the ClusterPluginDefinition")

			By("checking if the ClusterPluginDefinition is Ready")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the PluginDefinition")
				readyCondition := clusterDef.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
				g.Expect(readyCondition).ToNot(BeNil(), "the PluginDefinition should have a Ready condition")
			}).Should(Succeed(), "the ClusterPluginDefinition should be created successfully")

			By("checking if flux HelmRepository is created")
			repositoryURL := flux.ChartURLToName(HelmRepo)
			repository := &sourcev1.HelmRepository{}
			repository.SetName(repositoryURL)
			repository.SetNamespace(flux.HelmRepositoryDefaultNamespace)
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(repository), repository)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmRepository")
				g.Expect(repository.Spec.URL).To(Equal(HelmRepo), "the HelmRepository URL should match the ClusterPluginDefinition repository URL")
				return nil
			}).Should(Succeed(), "the HelmRepository should be created successfully")
		})
	})
})
