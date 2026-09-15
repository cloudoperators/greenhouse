// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugindefinition

import (
	"context"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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

			By("checking if flux HelmChart is created")
			helmChart := &sourcev1.HelmChart{}
			helmChart.SetName(pluginDef.FluxHelmChartResourceName())
			helmChart.SetNamespace(pluginDef.GetNamespace())
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(helmChart), helmChart)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmChart")
				g.Expect(helmChart.Spec.Chart).To(Equal(HelmChart), "the HelmChart chart should match")
				g.Expect(helmChart.Spec.Version).To(Equal(PluginDefinitionChartVersion), "the HelmChart version should match")
				return nil
			}).Should(Succeed(), "the HelmChart should be created successfully")

			By("checking if HelmChartReadyCondition is set on PluginDefinition")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred())
				helmChartCondition := pluginDef.Status.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
				g.Expect(helmChartCondition).ToNot(BeNil(), "the PluginDefinition should have a HelmChartReady condition")
				// Without Flux source-controller, HelmChart has no Ready condition yet.
				g.Expect(string(helmChartCondition.Status)).To(Equal(string(metav1.ConditionUnknown)))
			}).Should(Succeed())
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

	Context("When updating a PluginDefinition's HelmChart version", Ordered, func() {
		It("should delete orphaned HelmChart resources when the chart version is bumped", func() {
			By("creating a PluginDefinition with chart version 1.0.0")
			pluginDef := test.NewPluginDefinition(test.Ctx, "orphan-test-plugin", test.TestNamespace,
				test.WithPluginDefinitionVersion(PluginDefinitionVersion),
				test.WithPluginDefinitionHelmChart(&greenhousev1alpha1.HelmChartReference{
					Name:       HelmChart,
					Repository: HelmRepo,
					Version:    "1.0.0",
				}),
			)
			err := test.K8sClient.Create(test.Ctx, pluginDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the PluginDefinition")

			By("waiting for the initial HelmChart (v1.0.0) to be created")
			// Use FluxHelmChartResourceName() rather than hard-coding the name+version
			// convention so that the test stays aligned with the production logic.
			initialHelmChart := &sourcev1.HelmChart{}
			initialHelmChart.SetName(pluginDef.FluxHelmChartResourceName())
			initialHelmChart.SetNamespace(pluginDef.GetNamespace())
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(initialHelmChart), initialHelmChart)
				g.Expect(err).ToNot(HaveOccurred(), "initial HelmChart v1.0.0 should be created")
			}).Should(Succeed(), "the initial HelmChart should be created")

			By("updating the PluginDefinition to chart version 2.0.0")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(pluginDef), pluginDef)
				g.Expect(err).ToNot(HaveOccurred())
				pluginDef.Spec.HelmChart.Version = "2.0.0"
				err = test.K8sClient.Update(test.Ctx, pluginDef)
				g.Expect(err).ToNot(HaveOccurred(), "should be able to update the PluginDefinition version")
			}).Should(Succeed())

			By("verifying the new HelmChart (v2.0.0) is created")
			// Derive the new expected name via FluxHelmChartResourceName() after the version was bumped.
			newHelmChart := &sourcev1.HelmChart{}
			newHelmChart.SetName(pluginDef.FluxHelmChartResourceName())
			newHelmChart.SetNamespace(pluginDef.GetNamespace())
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(newHelmChart), newHelmChart)
				g.Expect(err).ToNot(HaveOccurred(), "new HelmChart v2.0.0 should be created")
			}).Should(Succeed(), "the new HelmChart v2.0.0 should be created")

			By("verifying the orphaned HelmChart (v1.0.0) is deleted")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(initialHelmChart), initialHelmChart)
				g.Expect(err).To(HaveOccurred(), "orphaned HelmChart v1.0.0 should be deleted")
				g.Expect(cl.IgnoreNotFound(err)).To(Succeed(), "the error should be a NotFound error")
			}).Should(Succeed(), "the orphaned HelmChart v1.0.0 should be deleted after the version bump")
		})
	})

	Context("When updating a ClusterPluginDefinition's HelmChart version", Ordered, func() {
		It("should delete orphaned HelmChart resources when the chart version is bumped", func() {
			By("creating a ClusterPluginDefinition with chart version 1.0.0")
			clusterDef := test.NewClusterPluginDefinition(test.Ctx, "orphan-cluster-test-plugin",
				test.WithVersion(PluginDefinitionVersion),
				test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
					Name:       HelmChart,
					Repository: HelmRepo,
					Version:    "1.0.0",
				}),
			)
			err := test.K8sClient.Create(test.Ctx, clusterDef)
			Expect(err).ToNot(HaveOccurred(), "there should be no error creating the ClusterPluginDefinition")

			By("waiting for the initial HelmChart (v1.0.0) to be created")
			initialHelmChart := &sourcev1.HelmChart{}
			initialHelmChart.SetName(clusterDef.FluxHelmChartResourceName())
			initialHelmChart.SetNamespace(flux.HelmRepositoryDefaultNamespace)
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(initialHelmChart), initialHelmChart)
				g.Expect(err).ToNot(HaveOccurred(), "initial HelmChart v1.0.0 should be created")
			}).Should(Succeed(), "the initial ClusterPluginDefinition HelmChart should be created")

			By("updating the ClusterPluginDefinition to chart version 2.0.0")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred())
				clusterDef.Spec.HelmChart.Version = "2.0.0"
				err = test.K8sClient.Update(test.Ctx, clusterDef)
				g.Expect(err).ToNot(HaveOccurred(), "should be able to update the ClusterPluginDefinition version")
			}).Should(Succeed())

			By("verifying the new HelmChart (v2.0.0) is created")
			newHelmChart := &sourcev1.HelmChart{}
			newHelmChart.SetName(clusterDef.FluxHelmChartResourceName())
			newHelmChart.SetNamespace(flux.HelmRepositoryDefaultNamespace)
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(newHelmChart), newHelmChart)
				g.Expect(err).ToNot(HaveOccurred(), "new HelmChart v2.0.0 should be created")
			}).Should(Succeed(), "the new ClusterPluginDefinition HelmChart v2.0.0 should be created")

			By("verifying the orphaned HelmChart (v1.0.0) is deleted")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(initialHelmChart), initialHelmChart)
				g.Expect(err).To(HaveOccurred(), "orphaned HelmChart v1.0.0 should be deleted")
				g.Expect(cl.IgnoreNotFound(err)).To(Succeed(), "the error should be a NotFound error")
			}).Should(Succeed(), "the orphaned ClusterPluginDefinition HelmChart v1.0.0 should be deleted after the version bump")
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

			By("checking if flux HelmChart is created")
			helmChart := &sourcev1.HelmChart{}
			helmChart.SetName(clusterDef.FluxHelmChartResourceName())
			helmChart.SetNamespace(flux.HelmRepositoryDefaultNamespace)
			Eventually(func(g Gomega) error {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(helmChart), helmChart)
				g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the HelmChart")
				g.Expect(helmChart.Spec.Chart).To(Equal(HelmChart), "the HelmChart chart should match")
				g.Expect(helmChart.Spec.Version).To(Equal(PluginDefinitionChartVersion), "the HelmChart version should match")
				return nil
			}).Should(Succeed(), "the HelmChart should be created successfully")

			By("checking if HelmChartReadyCondition is set on ClusterPluginDefinition")
			Eventually(func(g Gomega) {
				err := test.K8sClient.Get(test.Ctx, cl.ObjectKeyFromObject(clusterDef), clusterDef)
				g.Expect(err).ToNot(HaveOccurred())
				helmChartCondition := clusterDef.Status.GetConditionByType(greenhousev1alpha1.HelmChartReadyCondition)
				g.Expect(helmChartCondition).ToNot(BeNil(), "the ClusterPluginDefinition should have a HelmChartReady condition")
				// Without Flux source-controller, HelmChart has no Ready condition yet.
				g.Expect(string(helmChartCondition.Status)).To(Equal(string(metav1.ConditionUnknown)))
			}).Should(Succeed())
		})
	})
})

const (
	testRegistry  = "keppel.eu-de-1.cloud.sap"
	testChartName = "ccloud-ghcr-io-mirror/cloudoperators/greenhouse-extensions/charts/audit-logs"
	testVersion   = "0.0.21"
)

func replicatedPluginDefinition() *greenhousev1alpha1.PluginDefinition {
	return &greenhousev1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "audit-logs-compute", Namespace: "sci"},
		Status: greenhousev1alpha1.PluginDefinitionStatus{
			LastSyncedArtifact: &greenhousev1alpha1.LastSyncedArtifact{
				Registry:          testRegistry,
				ChartName:         testChartName,
				Version:           testVersion,
				ReplicationStatus: greenhousev1alpha1.ReplicationStatusReplicated,
			},
		},
	}
}

type noopRecorder struct{}

func (noopRecorder) Eventf(_, _ runtime.Object, _, _, _, _ string, _ ...any) {}

func replicationTestScheme() *runtime.Scheme {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(greenhousev1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(sourcev1.AddToScheme(scheme)).To(Succeed())
	return scheme
}

var _ = Describe("chart replication", func() {
	Context("shouldSkipChartReplication", func() {
		DescribeTable("deciding whether to skip an already-recorded chart",
			func(mutate func(*greenhousev1alpha1.PluginDefinition), expectSkip bool) {
				pluginDef := replicatedPluginDefinition()
				mutate(pluginDef)
				Expect(shouldSkipChartReplication(pluginDef, testRegistry, testChartName, testVersion)).To(Equal(expectSkip))
			},
			Entry("skips when the recorded artifact matches the desired chart",
				func(*greenhousev1alpha1.PluginDefinition) {}, true),
			Entry("replicates when nothing was recorded yet",
				func(pd *greenhousev1alpha1.PluginDefinition) { pd.Status.LastSyncedArtifact = nil }, false),
			Entry("replicates when the recorded version differs",
				func(pd *greenhousev1alpha1.PluginDefinition) { pd.Status.LastSyncedArtifact.Version = "0.0.20" }, false),
			Entry("replicates when a reconcile is requested via annotation",
				func(pd *greenhousev1alpha1.PluginDefinition) {
					pd.SetAnnotations(map[string]string{lifecycle.ReconcileAnnotation: "2026-08-25T13:34:13Z"})
				}, false),
			Entry("skips again once the annotation is removed",
				func(pd *greenhousev1alpha1.PluginDefinition) {
					pd.SetAnnotations(map[string]string{"unrelated": "value"})
				}, true),
		)
	})

	Context("createUpdateHelmChart", func() {
		DescribeTable("propagating the reconcile request to the HelmChart",
			func(annotations map[string]string, expectValue string) {
				pluginDef := replicatedPluginDefinition()
				pluginDef.SetAnnotations(annotations)
				pluginDef.Spec.HelmChart = &greenhousev1alpha1.HelmChartReference{
					Name:       "audit-logs",
					Repository: "oci://" + testRegistry + "/ccloud-ghcr-io-mirror/cloudoperators/greenhouse-extensions/charts",
					Version:    testVersion,
				}

				h := &helmer{
					k8sClient:     fake.NewClientBuilder().WithScheme(replicationTestScheme()).WithObjects(pluginDef).Build(),
					recorder:      noopRecorder{},
					pluginDef:     pluginDef,
					namespaceName: pluginDef.Namespace,
				}

				helmChart, err := h.createUpdateHelmChart(context.Background(), &sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{Name: "keppel-repo", Namespace: pluginDef.Namespace},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(helmChart.GetAnnotations()[fluxmeta.ReconcileRequestAnnotation]).To(Equal(expectValue))
			},
			Entry("propagates the reconcile request value to the HelmChart",
				map[string]string{lifecycle.ReconcileAnnotation: "2026-08-25T13:34:13Z"}, "2026-08-25T13:34:13Z"),
			Entry("leaves the HelmChart untouched when no reconcile was requested",
				nil, ""),
		)
	})
})
