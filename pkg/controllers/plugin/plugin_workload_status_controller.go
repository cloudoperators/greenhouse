// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/prometheus/client_golang/prometheus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

const (
	StatusRequeueInterval          = 2 * time.Minute
	helmReleaseNameAnnotation      = "meta.helm.sh/release-name"
	helmReleaseNamespaceAnnotation = "meta.helm.sh/release-namespace"
	jobNameLabel                   = "batch.kubernetes.io/job-name"
)

var objectFilter = []helm.ManifestObjectFilter{
	{APIVersion: "v1", Kind: "Pod"},
	{APIVersion: "v1", Kind: "Deployment"},
	{APIVersion: "v1", Kind: "StatefulSet"},
	{APIVersion: "v1", Kind: "DaemonSet"},
	{APIVersion: "v1", Kind: "ReplicaSet"},
	{APIVersion: "v1", Kind: "Job"},
	{APIVersion: "v1", Kind: "CronJob"},
	{APIVersion: "monitoring.coreos.com/v1", Kind: "Alertmanager"},
}

var (
	workloadStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "greenhouse_plugin_workload_status_up",
			Help: "The workload status of the plugin",
		},
		[]string{"org", "plugin", "pluginDefinition"},
	)
)

type PayloadStatus struct {
	Kind                string `json:"kind,omitempty"`
	Name                string `json:"name,omitempty"`
	Replicas            int32  `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`
	UpdatedReplicas     int32  `json:"updatedReplicas,omitempty" protobuf:"varint,3,opt,name=updatedReplicas"`
	ReadyReplicas       int32  `json:"readyReplicas,omitempty" protobuf:"varint,7,opt,name=readyReplicas"`
	AvailableReplicas   int32  `json:"availableReplicas,omitempty" protobuf:"varint,4,opt,name=availableReplicas"`
	UnavailableReplicas int32  `json:"unavailableReplicas,omitempty" protobuf:"varint,5,opt,name=unavailableReplicas"`
	Ready               bool   `json:"ready,omitempty"`
	Message             string `json:"message,omitempty"`
}

type ReleaseStatus struct {
	ReleaseName      string `json:"releaseName,omitempty" protobuf:"bytes,1,opt,name=releaseName"`
	ReleaseNamespace string `json:"releaseNamespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	HelmStatus       string `json:"helmStatus,omitempty"`
	ClusterName      string `json:"clusterName,omitempty"`
	PayloadStatus    []PayloadStatus
}

// WorkLoadStatusReconciler reconciles a Plugin and cluster object.
type WorkLoadStatusReconciler struct {
	client.Client
	recorder        record.EventRecorder
	KubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(workloadStatus)
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status,verbs=get;patch;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *WorkLoadStatusReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.kubeClientOpts = []clientutil.KubeClientOption{
		clientutil.WithRuntimeOptions(r.KubeRuntimeOpts),
		clientutil.WithPersistentConfig(),
	}
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Plugin{}).
		Complete(r)
}

func (r *WorkLoadStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var plugin = new(greenhousev1alpha1.Plugin)
	var pluginDefinition = new(greenhousev1alpha1.PluginDefinition)
	var releaseStatus = new(ReleaseStatus)
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: plugin.Spec.PluginDefinition}, pluginDefinition); err != nil {
		return ctrl.Result{}, err
	}
	// Nothing to do when the status of the plugin is empty and when the plugin does not have a Helm Chart
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) || plugin.Status.HelmChart == nil {
		return ctrl.Result{}, nil
	}

	if plugin.DeletionTimestamp != nil || pluginDefinition.Spec.HelmChart == nil {
		return ctrl.Result{}, nil
	}

	pluginStatus := initPluginStatus(plugin)
	defer func() {
		if statusErr := setPluginStatus(ctx, r.Client, plugin, pluginStatus); statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "failed to set status")
		}
	}()

	var restClientGetter genericclioptions.RESTClientGetter
	clusterAccessReadyCondition := greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ClusterAccessReadyCondition, "", "")
	// Check if the plugin is ready to be reconciled.
	cluster := &greenhousev1alpha1.Cluster{}
	if plugin.Spec.ClusterName != "" {
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: plugin.GetNamespace(), Name: plugin.Spec.ClusterName}, cluster)
		if err != nil {
			clusterAccessReadyCondition.Status = metav1.ConditionFalse
			clusterAccessReadyCondition.Message = fmt.Sprintf("Failed to get cluster %s: %s", plugin.Spec.ClusterName, err.Error())
			pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
			return ctrl.Result{}, err
		}
	}
	clusterAccessReadyCondition, restClientGetter = initClientGetter(ctx, r.Client, r.kubeClientOpts, *plugin, cluster, clusterAccessReadyCondition)
	pluginStatus.StatusConditions.SetConditions(clusterAccessReadyCondition)
	if !clusterAccessReadyCondition.IsTrue() {
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, fmt.Errorf("cannot access cluster: %s", clusterAccessReadyCondition.Message)
	}

	objClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return ctrl.Result{}, err
	}

	helmRelease, err := helm.GetReleaseForHelmChartFromPlugin(ctx, restClientGetter, plugin)
	// Skipping plugins that don't have a helm release, it could happen if the plugin is UI only
	if err == nil && plugin.Status.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).Status == metav1.ConditionFalse {
		releaseStatus.ReleaseName = helmRelease.Name
		releaseStatus.ReleaseNamespace = helmRelease.Namespace
		releaseStatus.ClusterName = plugin.Spec.ClusterName
		releaseStatus.HelmStatus = helmRelease.Info.Status.String()
		fileteredObjectMap, err := helm.ObjectMapFromManifest(restClientGetter, plugin.Namespace, helmRelease.Manifest, &helm.ManifestMultipleObjectFilter{
			Filters: objectFilter,
		})
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to get object map from manifest")
		}
		for key := range fileteredObjectMap {
			getPayloadStatus(ctx, releaseStatus, objClient, key.Name, releaseStatus.ReleaseNamespace, key.GVK)
		}

		workloadCondition := computeWorkloadCondition(plugin, pluginStatus, releaseStatus)
		pluginStatus.StatusConditions.SetConditions(workloadCondition)
	}
	return ctrl.Result{RequeueAfter: StatusRequeueInterval}, nil
}

// getPayloadStatus fetches the status of the object and updates the ReleaseStatus object
func getPayloadStatus(ctx context.Context, releaseStatus *ReleaseStatus, cl client.Client, objName, objNamespace string, gvk schema.GroupVersionKind) {
	status := new(PayloadStatus)
	status.Ready = true
	switch gvk.Kind {
	case "Deployment":
		remoteObject := &appsv1.Deployment{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting deployment", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		status.Replicas += remoteObject.Status.Replicas
		status.UpdatedReplicas += remoteObject.Status.UpdatedReplicas
		status.ReadyReplicas += remoteObject.Status.ReadyReplicas
		status.AvailableReplicas += remoteObject.Status.AvailableReplicas
		status.UnavailableReplicas += remoteObject.Status.UnavailableReplicas
		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "StatefulSet":
		remoteObject := &appsv1.StatefulSet{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting statefulset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		status.Replicas += remoteObject.Status.Replicas
		status.UpdatedReplicas += remoteObject.Status.UpdatedReplicas
		status.ReadyReplicas += remoteObject.Status.ReadyReplicas
		status.AvailableReplicas += remoteObject.Status.AvailableReplicas

		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "DaemonSet":
		remoteObject := &appsv1.DaemonSet{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting daemonset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		status.Replicas += remoteObject.Status.DesiredNumberScheduled
		status.UpdatedReplicas += remoteObject.Status.UpdatedNumberScheduled
		status.ReadyReplicas += remoteObject.Status.NumberReady
		status.AvailableReplicas += remoteObject.Status.NumberAvailable
		status.UnavailableReplicas += remoteObject.Status.NumberUnavailable

		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "ReplicaSet":
		remoteObject := &appsv1.ReplicaSet{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting replicaset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		status.Replicas += remoteObject.Status.Replicas
		status.ReadyReplicas += remoteObject.Status.ReadyReplicas
		status.AvailableReplicas += remoteObject.Status.AvailableReplicas

		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "Job":
		remoteObject := &batchv1.Job{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting job", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "CronJob":
		remoteObject := &batchv1.CronJob{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting cronjob", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "Pod":
		remoteObject := &corev1.Pod{}
		if err := cl.Get(ctx, types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting pod", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	case "Alertmanager":
		remoteObject := &corev1.PodList{}
		listOptions := &client.ListOptions{
			LabelSelector: labels.NewSelector(),
			Namespace:     objNamespace,
		}

		listOptions.LabelSelector.Matches(labels.Set{helmReleaseNameAnnotation: objName, helmReleaseNamespaceAnnotation: objNamespace})
		notFromCronJob, err := labels.NewRequirement(jobNameLabel, selection.DoesNotExist, []string{})
		if err != nil {
			log.FromContext(ctx).Error(err, "Error creating label selector", "name", objName, "pluginName", releaseStatus.ReleaseName)
		}
		listOptions.LabelSelector.Add(*notFromCronJob)
		if err := cl.List(ctx, remoteObject, listOptions); err != nil {
			log.FromContext(ctx).Error(err, "Error getting alertmanager", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			status.Ready = false
			status.Message = fmt.Sprintf("%s/%s", gvk.Kind, objName)
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		} else {
			releaseStatus.PayloadStatus = append(releaseStatus.PayloadStatus, *status)
		}
	}
}
