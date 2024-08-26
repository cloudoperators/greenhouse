package cluster

import (
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	kubernetesVersionsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenhouse_cluster_k8s_versions_total",
		},
		[]string{"cluster", "namespace", "version"})

	secondsToTokenExpiryGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_cluster_kubeconfig_validity_seconds",
		},
		[]string{"cluster", "namespace"})
)

func init() {
	metrics.Registry.MustRegister(kubernetesVersionsCounter)
	metrics.Registry.MustRegister(secondsToTokenExpiryGauge)
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list

func UpdateMetrics(cluster *greenhousev1alpha1.Cluster) {
	kubernetesVersionLabels := prometheus.Labels{
		"cluster":   cluster.Name,
		"namespace": cluster.Namespace,
		"version":   cluster.Status.KubernetesVersion,
	}
	kubernetesVersionsCounter.With(kubernetesVersionLabels).Inc()

	secondsToExpiry := cluster.Status.BearerTokenExpirationTimestamp.Unix() - time.Now().Unix()
	secondsToExpiryLabels := prometheus.Labels{
		"cluster":   cluster.Name,
		"namespace": cluster.Namespace,
	}
	secondsToTokenExpiryGauge.With(secondsToExpiryLabels).Set(float64(secondsToExpiry))
}
