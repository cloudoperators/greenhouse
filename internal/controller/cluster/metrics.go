// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var (
	kubernetesVersionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_k8s_versions_total",
		},
		[]string{"cluster", "namespace", "version", "owned_by"})

	secondsToTokenExpiryGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_kubeconfig_validity_seconds",
		},
		[]string{"cluster", "namespace", "owned_by"})
)

func init() {
	metrics.Registry.MustRegister(kubernetesVersionsGauge)
	metrics.Registry.MustRegister(secondsToTokenExpiryGauge)
}

func updateMetrics(cluster *greenhousev1alpha1.Cluster) {
	kubernetesVersionLabels := prometheus.Labels{
		"cluster":   cluster.Name,
		"namespace": cluster.Namespace,
		"version":   cluster.Status.KubernetesVersion,
		"owned_by":  cluster.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	kubernetesVersionsGauge.With(kubernetesVersionLabels).Set(float64(1))

	secondsToExpiry := cluster.Status.BearerTokenExpirationTimestamp.Unix() - time.Now().Unix()
	secondsToExpiryLabels := prometheus.Labels{
		"cluster":   cluster.Name,
		"namespace": cluster.Namespace,
		"owned_by":  cluster.Labels[greenhouseapis.LabelKeyOwnedBy],
	}
	secondsToTokenExpiryGauge.With(secondsToExpiryLabels).Set(float64(secondsToExpiry))
}
