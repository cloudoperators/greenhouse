// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	ClusterReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_ready",
			Help: "Indicates whether the cluster is ready",
		},
		[]string{"clusterName", "namespace", "owned_by"})

	KubernetesVersionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_k8s_versions_total",
		},
		[]string{"clusterName", "namespace", "version", "owned_by"})

	SecondsToTokenExpiryGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_kubeconfig_validity_seconds",
		},
		[]string{"clusterName", "namespace", "owned_by"})
)

func init() {
	crmetrics.Registry.MustRegister(KubernetesVersionsGauge)
	crmetrics.Registry.MustRegister(SecondsToTokenExpiryGauge)
	crmetrics.Registry.MustRegister(ClusterReadyGauge)
}

func UpdateClusterMetrics(cluster *greenhousev1alpha1.Cluster) {
	kubernetesVersionLabels := prometheus.Labels{
		"clusterName":   cluster.Name,
		"namespace": cluster.Namespace,
		"version":   cluster.Status.KubernetesVersion,
		"owned_by":  cluster.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	KubernetesVersionsGauge.With(kubernetesVersionLabels).Set(float64(1))

	secondsToExpiry := cluster.Status.BearerTokenExpirationTimestamp.Unix() - time.Now().Unix()
	secondsToExpiryLabels := prometheus.Labels{
		"clusterName":   cluster.Name,
		"namespace": cluster.Namespace,
		"owned_by":  cluster.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	SecondsToTokenExpiryGauge.With(secondsToExpiryLabels).Set(float64(secondsToExpiry))

	clusterReadyLabels := prometheus.Labels{
		"clusterName":   cluster.Name,
		"namespace": cluster.Namespace,
		"owned_by":  cluster.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	if cluster.Status.IsReadyTrue() {
		ClusterReadyGauge.With(clusterReadyLabels).Set(float64(1))
	} else {
		ClusterReadyGauge.With(clusterReadyLabels).Set(float64(0))
	}
}
