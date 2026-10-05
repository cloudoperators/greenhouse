// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"testing"
	"time"

	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster"
)

const (
	testClusterName = "test-cluster"
	testNamespace   = "test-org"
	testOwner       = "test-owner"
)

// newCluster returns a cluster carrying the owned-by label and a token expiring in 600s.
func newCluster() *greenhousev1alpha1.Cluster {
	return &greenhousev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testClusterName,
			Namespace: testNamespace,
			Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: testOwner},
		},
		Status: greenhousev1alpha1.ClusterStatus{
			KubernetesVersion:              "1.31.1",
			BearerTokenExpirationTimestamp: metav1.Time{Time: time.Now().Add(600 * time.Second)},
		},
	}
}

// resetMetrics clears the package-level gauges so series do not leak between tests.
func resetMetrics() {
	cluster.KubernetesVersionsGauge.Reset()
	cluster.SecondsToTokenExpiryGauge.Reset()
	cluster.ClusterReadyGauge.Reset()
}

func TestUpdateClusterMetrics(t *testing.T) {
	tests := []struct {
		name string
		// mutate adjusts the cluster before UpdateClusterMetrics is called.
		mutate func(c *greenhousev1alpha1.Cluster)
		// assert runs after UpdateClusterMetrics against the cluster that was passed in.
		assert func(t *testing.T, c *greenhousev1alpha1.Cluster)
	}{
		{
			name: "reports version, token validity and readiness for a non-workload-identity cluster",
			assert: func(t *testing.T, c *greenhousev1alpha1.Cluster) {
				require.EqualValues(t, 1,
					prometheusTest.ToFloat64(cluster.KubernetesVersionsGauge.WithLabelValues(c.Name, c.Namespace, c.Status.KubernetesVersion, testOwner)),
					"the kubernetes version series should be present")

				tokenExpiry := prometheusTest.ToFloat64(cluster.SecondsToTokenExpiryGauge.WithLabelValues(c.Name, c.Namespace, testOwner))
				require.GreaterOrEqual(t, tokenExpiry, float64(595), "token validity should reflect the expiration timestamp")
				require.LessOrEqual(t, tokenExpiry, float64(600), "token validity should reflect the expiration timestamp")

				require.EqualValues(t, 0,
					prometheusTest.ToFloat64(cluster.ClusterReadyGauge.WithLabelValues(c.Name, c.Namespace, testOwner)),
					"readiness should be reported and the cluster should not be ready")
			},
		},
		{
			name: "reports readiness as 1 for a ready cluster",
			mutate: func(c *greenhousev1alpha1.Cluster) {
				c.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ReadyCondition, "", ""))
			},
			assert: func(t *testing.T, c *greenhousev1alpha1.Cluster) {
				require.EqualValues(t, 1,
					prometheusTest.ToFloat64(cluster.ClusterReadyGauge.WithLabelValues(c.Name, c.Namespace, testOwner)),
					"a ready cluster should report readiness as 1")
			},
		},
		{
			name: "does not report token validity for a workload identity cluster",
			mutate: func(c *greenhousev1alpha1.Cluster) {
				c.Annotations = map[string]string{
					greenhouseapis.ClusterWorkloadIdentityAnnotation: greenhouseapis.ClusterWorkloadIdentityEnabled,
				}
			},
			assert: func(t *testing.T, c *greenhousev1alpha1.Cluster) {
				// WI clusters mint short-lived tokens per request and have no kubeconfig
				// token validity to report, so no series must be emitted.
				require.EqualValues(t, 0, prometheusTest.CollectAndCount(cluster.SecondsToTokenExpiryGauge),
					"the token validity series should not be emitted for a workload identity cluster")
				// readiness is still reported regardless of access mode.
				require.EqualValues(t, 0,
					prometheusTest.ToFloat64(cluster.ClusterReadyGauge.WithLabelValues(c.Name, c.Namespace, testOwner)),
					"readiness should still be reported for a workload identity cluster")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			c := newCluster()
			if tt.mutate != nil {
				tt.mutate(c)
			}
			cluster.UpdateClusterMetrics(c)
			tt.assert(t, c)
		})
	}
}

// TestUpdateClusterMetricsDropsStaleSeries covers the cases where a second call must drop a
// previously emitted series: a kubernetes version change and a migration to workload identity.
func TestUpdateClusterMetricsDropsStaleSeries(t *testing.T) {
	tests := []struct {
		name string
		// change mutates the cluster between the first and second UpdateClusterMetrics calls.
		change func(c *greenhousev1alpha1.Cluster)
		assert func(t *testing.T, c *greenhousev1alpha1.Cluster)
	}{
		{
			name: "drops the previous kubernetes version series when the version changes",
			change: func(c *greenhousev1alpha1.Cluster) {
				c.Status.KubernetesVersion = "1.32.0"
			},
			assert: func(t *testing.T, c *greenhousev1alpha1.Cluster) {
				require.EqualValues(t, 1,
					prometheusTest.ToFloat64(cluster.KubernetesVersionsGauge.WithLabelValues(c.Name, c.Namespace, "1.32.0", testOwner)),
					"the new kubernetes version series should be present")
				// If the previous version's series had survived, .Set(1) from the first call
				// would still be observable. A value of 0 means the series was dropped (and only
				// re-created here as a side effect of querying it).
				require.EqualValues(t, 0,
					prometheusTest.ToFloat64(cluster.KubernetesVersionsGauge.WithLabelValues(c.Name, c.Namespace, "1.31.1", testOwner)),
					"the previous kubernetes version series should have been removed")
			},
		},
		{
			name: "drops the token validity series when the cluster migrates to workload identity",
			change: func(c *greenhousev1alpha1.Cluster) {
				c.Annotations = map[string]string{
					greenhouseapis.ClusterWorkloadIdentityAnnotation: greenhouseapis.ClusterWorkloadIdentityEnabled,
				}
			},
			assert: func(t *testing.T, c *greenhousev1alpha1.Cluster) {
				require.EqualValues(t, 0, prometheusTest.CollectAndCount(cluster.SecondsToTokenExpiryGauge),
					"the stale token validity series should have been removed after migration to workload identity")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			c := newCluster()

			cluster.UpdateClusterMetrics(c)
			// the initial call seeds the series that the second call must drop.
			require.Greater(t,
				prometheusTest.ToFloat64(cluster.SecondsToTokenExpiryGauge.WithLabelValues(c.Name, c.Namespace, testOwner)),
				float64(0), "the initial call should emit a token validity series")

			tt.change(c)
			cluster.UpdateClusterMetrics(c)
			tt.assert(t, c)
		})
	}
}
