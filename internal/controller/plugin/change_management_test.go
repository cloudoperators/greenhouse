// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/flux"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const changeManagementTestConfig = `endpoint: %s
headersSecretRef: change-headers
payloadTemplate: |
  {{ if ne .Release.Status "failed" }}{"organization": {{ .Organization | toJson }}, "plugin": {{ .Plugin.Name | toJson }}, "cluster": {{ .Cluster.Name | toJson }}, "digest": {{ .Release.Digest | toJson }}}{{ end }}
`

type changeEndpoint struct {
	*httptest.Server
	status   atomic.Int32
	attempts atomic.Int32
	mu       sync.Mutex
	reports  []changeReport
}

type changeReport struct {
	auth    string
	payload map[string]any
}

func newChangeEndpoint() *changeEndpoint {
	endpoint := &changeEndpoint{}
	endpoint.status.Store(http.StatusOK)
	endpoint.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint.attempts.Add(1)
		status := int(endpoint.status.Load())
		if status == http.StatusOK {
			report := changeReport{auth: r.Header.Get("Authorization")}
			if err := json.NewDecoder(r.Body).Decode(&report.payload); err != nil {
				status = http.StatusBadRequest
			} else {
				endpoint.mu.Lock()
				endpoint.reports = append(endpoint.reports, report)
				endpoint.mu.Unlock()
			}
		}
		w.WriteHeader(status)
	}))
	return endpoint
}

func (e *changeEndpoint) received() []changeReport {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.reports)
}

func releaseSnapshot(digest string, version int) *helmv2.Snapshot {
	now := metav1.Now()
	return &helmv2.Snapshot{
		Digest:        digest,
		Name:          "release",
		Namespace:     "release",
		Version:       version,
		Status:        "deployed",
		ChartName:     "dummy",
		ChartVersion:  "1.0.0",
		ConfigDigest:  "sha256:config",
		FirstDeployed: now,
		LastDeployed:  now,
	}
}

func setReleaseHistory(plugin *greenhousev1alpha1.Plugin, snapshots ...*helmv2.Snapshot) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		helmRelease := &helmv2.HelmRelease{}
		g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(plugin), helmRelease)).To(Succeed())
		helmRelease.Status.History = snapshots
		g.Expect(test.K8sClient.Status().Update(test.Ctx, helmRelease)).To(Succeed())
	}).Should(Succeed())
}

func triggerReconcile(g Gomega, plugin *greenhousev1alpha1.Plugin) {
	current := getPlugin(g, plugin)
	current.Labels["change-management-test/trigger"] = strconv.FormatInt(time.Now().UnixNano(), 10)
	g.Expect(test.K8sClient.Update(test.Ctx, current)).To(Succeed())
}

func getPlugin(g Gomega, plugin *greenhousev1alpha1.Plugin) *greenhousev1alpha1.Plugin {
	current := &greenhousev1alpha1.Plugin{}
	g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(plugin), current)).To(Succeed())
	return current
}

var _ = Describe("Change management", Ordered, func() {
	var (
		setup      *test.TestSetup
		endpoint   *changeEndpoint
		team       *greenhousev1alpha1.Team
		cluster    *greenhousev1alpha1.Cluster
		definition *greenhousev1alpha1.ClusterPluginDefinition
		plugin     *greenhousev1alpha1.Plugin
	)

	createPlugin := func(name string) *greenhousev1alpha1.Plugin {
		GinkgoHelper()
		created := setup.CreatePlugin(test.Ctx, name,
			test.WithClusterPluginDefinition(definition.Name),
			test.WithReleaseName(name),
			test.WithCluster(cluster.Name),
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(created), &helmv2.HelmRelease{})).To(Succeed())
		}).Should(Succeed(), "the HelmRelease should be created for the Plugin")
		return created
	}

	BeforeAll(func() {
		endpoint = newChangeEndpoint()
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, "change-management")
		Expect(test.K8sClient.Create(test.Ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "org-config", Namespace: setup.Namespace()},
		})).To(Succeed())
		setup.CreateOrganization(test.Ctx, setup.Namespace(), test.WithConfigMapRef("org-config"))
		team = setup.CreateTeam(test.Ctx, "change-management-team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		cluster = setup.CreateCluster(test.Ctx, "change-management-cluster",
			test.WithAccessMode(greenhousev1alpha1.ClusterAccessModeDirect),
			test.WithClusterLabel(greenhouseapis.LabelKeyOwnedBy, team.Name))
		setup.CreateSecret(test.Ctx, "change-headers", test.WithSecretData(map[string][]byte{"Authorization": []byte("Bearer token\n")}))
		definition = setup.CreateClusterPluginDefinition(test.Ctx, "change-management",
			test.WithVersion("1.0.0"),
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{Name: "dummy", Repository: "oci://greenhouse/helm-charts", Version: "1.0.0"}))
		test.MockHelmChartReady(test.Ctx, test.K8sClient, definition, flux.HelmRepositoryDefaultNamespace)
	})

	AfterAll(func() {
		endpoint.Close()
	})

	It("records the revision deployed before change management was configured without reporting it", func() {
		plugin = createPlugin("change-management")
		setReleaseHistory(plugin, releaseSnapshot("sha256:1", 1))

		configMap := &corev1.ConfigMap{}
		Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "org-config", Namespace: setup.Namespace()}, configMap)).To(Succeed())
		configMap.Data = map[string]string{changeManagementConfigKey: fmt.Sprintf(changeManagementTestConfig, endpoint.URL)}
		Expect(test.K8sClient.Update(test.Ctx, configMap)).To(Succeed())

		Eventually(func(g Gomega) {
			triggerReconcile(g, plugin)
			g.Expect(getPlugin(g, plugin).Status.ChangeManagement).To(Equal(&greenhousev1alpha1.ChangeManagementStatus{LastReportedDigest: "sha256:1"}))
		}).WithPolling(time.Second).Should(Succeed())
		Expect(endpoint.attempts.Load()).To(BeZero())
	})

	It("reports a new revision once", func() {
		setReleaseHistory(plugin, releaseSnapshot("sha256:2", 2), releaseSnapshot("sha256:1", 1))

		Eventually(func(g Gomega) {
			g.Expect(getPlugin(g, plugin).Status.ChangeManagement.LastReportedDigest).To(Equal("sha256:2"))
		}).Should(Succeed())
		Expect(endpoint.received()).To(ConsistOf(changeReport{
			auth: "Bearer token",
			payload: map[string]any{
				"organization": setup.Namespace(),
				"plugin":       plugin.Name,
				"cluster":      cluster.Name,
				"digest":       "sha256:2",
			},
		}))

		Eventually(func(g Gomega) { triggerReconcile(g, plugin) }).Should(Succeed())
		Consistently(endpoint.received).WithTimeout(2 * time.Second).Should(HaveLen(1))
	})

	It("retries a failed report until the endpoint accepts it", func() {
		endpoint.status.Store(http.StatusServiceUnavailable)
		setReleaseHistory(plugin, releaseSnapshot("sha256:3", 3), releaseSnapshot("sha256:2", 2))

		Eventually(func(g Gomega) {
			condition := getPlugin(g, plugin).Status.GetConditionByType(greenhousev1alpha1.ChangeReportedCondition)
			g.Expect(condition).ToNot(BeNil())
			g.Expect(condition.IsFalse()).To(BeTrue())
			g.Expect(condition.Message).To(ContainSubstring("503"))
		}).Should(Succeed())
		Expect(getPlugin(Default, plugin).Status.ChangeManagement.LastReportedDigest).To(Equal("sha256:2"))

		endpoint.status.Store(http.StatusOK)
		Eventually(func(g Gomega) {
			current := getPlugin(g, plugin)
			g.Expect(current.Status.ChangeManagement.LastReportedDigest).To(Equal("sha256:3"))
			g.Expect(current.Status.GetConditionByType(greenhousev1alpha1.ChangeReportedCondition).IsTrue()).To(BeTrue())
		}).Should(Succeed())
		Expect(endpoint.received()).To(HaveLen(2))
	})

	It("skips a revision the template renders nothing for", func() {
		attempts := endpoint.attempts.Load()
		failed := releaseSnapshot("sha256:4", 4)
		failed.Status = "failed"
		setReleaseHistory(plugin, failed, releaseSnapshot("sha256:3", 3))

		Eventually(func(g Gomega) {
			g.Expect(getPlugin(g, plugin).Status.ChangeManagement.LastReportedDigest).To(Equal("sha256:4"))
		}).Should(Succeed())
		Expect(endpoint.attempts.Load()).To(Equal(attempts))
	})

	It("reports the first install of a new Plugin", func() {
		newPlugin := createPlugin("change-management-new")
		Eventually(func(g Gomega) {
			g.Expect(getPlugin(g, newPlugin).Status.ChangeManagement).ToNot(BeNil())
		}).Should(Succeed())

		setReleaseHistory(newPlugin, releaseSnapshot("sha256:new", 1))
		Eventually(func(g Gomega) {
			g.Expect(getPlugin(g, newPlugin).Status.ChangeManagement.LastReportedDigest).To(Equal("sha256:new"))
		}).Should(Succeed())
		Expect(endpoint.received()).To(HaveLen(3))
		Expect(endpoint.received()[2].payload).To(HaveKeyWithValue("plugin", newPlugin.Name))
	})
})

var _ = Describe("Change payload", func() {
	properties := changeProperties{Organization: "demo", Release: &helmv2.Snapshot{Digest: "sha256:1"}}

	It("renders the template with sprig functions", func() {
		payload, err := renderChangePayload(`{"organization": {{ .Organization | upper | toJson }}, "digest": {{ .Release.Digest | toJson }}}`, properties)
		Expect(err).ToNot(HaveOccurred())
		Expect(payload).To(MatchJSON(`{"organization": "DEMO", "digest": "sha256:1"}`))
	})

	It("renders nothing when the template skips the revision", func() {
		payload, err := renderChangePayload(`{{ if eq .Release.Status "deployed" }}{"digest": {{ .Release.Digest | toJson }}}{{ end }}`+"\n", properties)
		Expect(err).ToNot(HaveOccurred())
		Expect(payload).To(BeNil())
	})

	It("does not expose the controller environment", func() {
		_, err := renderChangePayload(`{"home": {{ env "HOME" | toJson }}}`, properties)
		Expect(err).To(MatchError(ContainSubstring(`function "env" not defined`)))
	})

	It("rejects a payload that is not JSON", func() {
		_, err := renderChangePayload(`{"digest": {{ .Release.Digest }}}`, properties)
		Expect(err).To(MatchError(ContainSubstring("valid JSON")))
	})
})
