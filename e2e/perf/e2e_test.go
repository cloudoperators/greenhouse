// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build perfE2E

package perf

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/perf/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	remoteClusterName         = "remote-plugin-cluster"
	preventDeletionAnnotation = "greenhouse.sap/prevent-deletion"
	testTeamIDPGroup          = "test-idp-group"
	testTeamName              = "test-perf-team"
	numTestRuns               = 10
)

var (
	env                  *shared.TestEnv
	ctx                  context.Context
	adminClient          client.Client
	remoteClient         client.Client
	testPluginDefinition *greenhousev1alpha1.PluginDefinition
	testTeam             *greenhousev1alpha1.Team
	testStartTime        time.Time
	experiment           *gmeasure.Experiment
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Performance E2E Suite")
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

	By("Re-runing the Manager Deployment Pod with all controllers except for plugin and kubeconfig")
	// Disable the plugin and kubeconfig controllers to not interfere with Plugin deletion. Only Plugin creation is tested here.
	desiredValue := "organizationController,teamController,teamRoleBindingController,pluginDefinition,bootStrap,clusterReconciler"
	shared.RerunManagerDeploymentPodWithDifferentControllers(ctx, adminClient, desiredValue)

	By("creating a Team")
	testTeam = test.NewTeam(ctx, testTeamName, env.TestNamespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
	err = client.IgnoreAlreadyExists(adminClient.Create(ctx, testTeam))
	Expect(err).ToNot(HaveOccurred(), "error creating the Team")

	By("onboarding remote cluster")
	shared.OnboardRemoteCluster(ctx, adminClient, env.RemoteKubeConfigBytes, remoteClusterName, env.TestNamespace, testTeamName)
	By("verifying if the cluster resource is created")
	Eventually(func(g Gomega) {
		err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, &greenhousev1alpha1.Cluster{})
		g.Expect(err).ToNot(HaveOccurred())
	}).Should(Succeed(), "cluster resource should be created")
	By("verifying the cluster status is ready")
	shared.ClusterIsReady(ctx, adminClient, remoteClusterName, env.TestNamespace)

	By("Creating a plugin definition with cert-manager helm chart")
	testPluginDefinition = fixtures.PrepareCertManagerPluginDefinition(env.TestNamespace)
	err = adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).To(Succeed(), "there should be no error creating the plugin definition")

	experiment = gmeasure.NewExperiment("Plugin Creation")
})

var _ = AfterSuite(func() {
	By("Checking if all Plugins have been deleted")
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		g.Expect(adminClient.List(ctx, pluginList)).To(Succeed(), "failed listing the Plugins")
		g.Expect(pluginList.Items).To(BeEmpty(), "there should be no Plugins left")
	}).Should(Succeed(), "All Plugins should have been deleted in the tests")

	By("Offboarding remote cluster")
	shared.OffBoardRemoteCluster(ctx, adminClient, remoteClient, testStartTime, remoteClusterName, env.TestNamespace)

	By("Deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)

	By("Deleting the Team")
	test.EventuallyDeleted(ctx, adminClient, testTeam)

	env.GenerateControllerLogs(ctx, testStartTime)

	By("Re-runing the Manager Deployment Pod with all controllers")
	shared.RerunManagerDeploymentPodWithAllControllers(ctx, adminClient)
})

var _ = ReportAfterSuite("Plugin validating webhook latency", func(_ Report) {
	measurement := experiment.Get("Single Create - No Owner Label")
	meanWithStdDev := formatMeanAndStdDevFromDurations(measurement.Durations)
	AddReportEntry(measurement.Name, shared.ReportEntryStringer{Data: map[string]string{
		"Mean": meanWithStdDev,
	}})

	measurement = experiment.Get("Single Create - With Owner Label")
	meanWithStdDev = formatMeanAndStdDevFromDurations(measurement.Durations)
	AddReportEntry(measurement.Name, shared.ReportEntryStringer{Data: map[string]string{
		"Mean": meanWithStdDev,
	}})

	measurement = experiment.Get("Parallel Create - No Owner Label")
	total := getTotalDuration(measurement.Durations)
	durationsCount := len(measurement.Durations)
	AddReportEntry(measurement.Name, shared.ReportEntryStringer{Data: map[string]string{
		"Mean":       fmt.Sprintf("%.2f ms/resource", (float64(total.Milliseconds()) / float64(durationsCount))),
		"Throughput": fmt.Sprintf("%.2f resources/sec", float64(durationsCount)/total.Seconds()),
		fmt.Sprintf("Total time to create %d resources", durationsCount): fmt.Sprintf("%.2fs", total.Seconds()),
	}})

	measurement = experiment.Get("Parallel Create - With Owner Label")
	total = getTotalDuration(measurement.Durations)
	durationsCount = len(measurement.Durations)
	AddReportEntry(measurement.Name, shared.ReportEntryStringer{Data: map[string]string{
		"Mean":       fmt.Sprintf("%.2f ms/resource", (float64(total.Milliseconds()) / float64(durationsCount))),
		"Throughput": fmt.Sprintf("%.2f resources/sec", float64(durationsCount)/total.Seconds()),
		fmt.Sprintf("Total time to create %d resources", durationsCount): fmt.Sprintf("%.2fs", total.Seconds()),
	}})
})

var _ = Describe("Webhook Performance", Ordered, func() {
	Context("Validating Plugin webhook latency", func() {
		When("Plugin does not have an owner label", func() {
			It("Measures performance of a single creation", func() {
				Eventually(func(g Gomega) {
					var testPlugin *greenhousev1alpha1.Plugin
					pluginName := "test-1-plugin-without-" + rand.String(8)
					duration := experiment.MeasureDuration("Single Create - No Owner Label", func() {
						testPlugin = fixtures.PreparePlugin(pluginName, env.TestNamespace,
							test.WithPluginDefinition(testPluginDefinition.Name),
							test.WithCluster(remoteClusterName),
							test.WithReleaseNamespace(env.TestNamespace),
							test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
							test.WithReleaseName(pluginName),
						)
						err := adminClient.Create(ctx, testPlugin)
						g.Expect(err).ToNot(HaveOccurred(), "error creating the Plugin")
					})
					GinkgoWriter.Printf("Create latency: %.1fms\n", float64(duration.Milliseconds()))

					test.EventuallyDeleted(ctx, adminClient, testPlugin)
				}).MustPassRepeatedly(numTestRuns).
					// Polling is used to avoid race conditions in repeated tests.
					WithPolling(500*time.Millisecond).
					Should(Succeed(), "Creation and deletion failed")
			})

			It("Measures performance under concurrent creation", func() {
				var wg sync.WaitGroup
				numWorkers := numTestRuns

				wg.Add(numWorkers)
				for i := range numWorkers {
					go func(workerId int) {
						defer wg.Done()
						defer GinkgoRecover()

						localCtx := context.Background()

						Eventually(func(g Gomega) {
							var testPlugin *greenhousev1alpha1.Plugin
							pluginName := "test-paral-plugin-without-" + rand.String(8)
							duration := experiment.MeasureDuration("Parallel Create - No Owner Label", func() {
								testPlugin = fixtures.PreparePlugin(pluginName, env.TestNamespace,
									test.WithPluginDefinition(testPluginDefinition.Name),
									test.WithCluster(remoteClusterName),
									test.WithReleaseNamespace(env.TestNamespace),
									test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
									test.WithReleaseName(pluginName),
								)
								err := adminClient.Create(localCtx, testPlugin)
								g.Expect(err).ToNot(HaveOccurred(), "error creating the Plugin")
							})
							GinkgoWriter.Printf("Create latency: %.1fms\n", float64(duration.Milliseconds()))

							test.EventuallyDeleted(localCtx, adminClient, testPlugin)
						}).Should(Succeed(), "Creation and deletion failed")
					}(i)
				}
				wg.Wait()
			})
		})

		When("Plugin has an owner label", func() {
			It("Measures performance of a single creation", func() {
				Eventually(func(g Gomega) {
					var testPlugin *greenhousev1alpha1.Plugin
					pluginName := "test-1-plugin-with-" + rand.String(8)
					duration := experiment.MeasureDuration("Single Create - With Owner Label", func() {
						testPlugin = fixtures.PreparePlugin(pluginName, env.TestNamespace,
							test.WithPluginDefinition(testPluginDefinition.Name),
							test.WithCluster(remoteClusterName),
							test.WithReleaseNamespace(env.TestNamespace),
							test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
							test.WithReleaseName(pluginName),
							test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeamName),
						)
						err := adminClient.Create(ctx, testPlugin)
						g.Expect(err).ToNot(HaveOccurred(), "error creating the Plugin")
					})
					GinkgoWriter.Printf("Create latency: %.1fms\n", float64(duration.Milliseconds()))

					test.EventuallyDeleted(ctx, adminClient, testPlugin)
				}).MustPassRepeatedly(numTestRuns).
					// Polling is used to avoid race conditions in repeated tests.
					WithPolling(500*time.Millisecond).
					Should(Succeed(), "Creation and deletion failed")
			})

			It("Measures performance under concurrent creation", func() {
				var wg sync.WaitGroup
				numWorkers := numTestRuns

				wg.Add(numWorkers)
				for i := range numWorkers {
					go func(workerId int) {
						defer wg.Done()
						defer GinkgoRecover()
						localCtx := context.Background()

						Eventually(func(g Gomega) {
							var testPlugin *greenhousev1alpha1.Plugin
							pluginName := "test-paral-plugin-with-" + rand.String(8)
							duration := experiment.MeasureDuration("Parallel Create - With Owner Label", func() {
								testPlugin = fixtures.PreparePlugin(pluginName, env.TestNamespace,
									test.WithPluginDefinition(testPluginDefinition.Name),
									test.WithCluster(remoteClusterName),
									test.WithReleaseNamespace(env.TestNamespace),
									test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}, nil),
									test.WithReleaseName(pluginName),
									test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, testTeamName),
								)
								err := adminClient.Create(localCtx, testPlugin)
								Expect(err).ToNot(HaveOccurred(), "error creating the Plugin")
							})
							GinkgoWriter.Printf("Create latency: %.1fms\n", float64(duration.Milliseconds()))

							test.EventuallyDeleted(localCtx, adminClient, testPlugin)
						}).Should(Succeed(), "Creation and deletion failed")
					}(i)
				}
				wg.Wait()
			})
		})
	})
})

func formatMeanAndStdDevFromDurations(durations []time.Duration) string {
	if len(durations) == 0 {
		return "No durations recorded"
	}
	totalDuration := getTotalDuration(durations)
	mean := float64(totalDuration.Milliseconds()) / float64(len(durations))

	var variance float64
	for _, d := range durations {
		ms := d.Seconds() * 1000
		variance += (ms - mean) * (ms - mean)
	}
	stdDev := math.Sqrt(variance / float64(len(durations)))
	return fmt.Sprintf("%.1fms ± %.1fms", mean, stdDev)
}
func getTotalDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total
}
