package cluster

import (
	"context"
	"time"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const metricsRequeueInterval = 2 * time.Minute

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

// ClusterMetricsReconciler reconciles the overall status of a remote cluster
type ClusterMetricsReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterMetricsReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}).
		Complete(r)
}

func (r *ClusterMetricsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, err
	}

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

	return ctrl.Result{RequeueAfter: metricsRequeueInterval}, nil
}
