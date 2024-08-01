// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/helm"
)

const StatusRequeueInterval = 2 * time.Minute

type ReleaseStatus struct {
	ReleaseName         string       `json:"releaseName,omitempty" protobuf:"bytes,1,opt,name=releaseName"`
	ReleaseNamespace    string       `json:"releaseNamespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	HelmStatus          string       `json:"helmStatus,omitempty"`
	ClusterName         string       `json:"clusterName,omitempty"`
	Replicas            int32        `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`
	UpdatedReplicas     int32        `json:"updatedReplicas,omitempty" protobuf:"varint,3,opt,name=updatedReplicas"`
	ReadyReplicas       int32        `json:"readyReplicas,omitempty" protobuf:"varint,7,opt,name=readyReplicas"`
	AvailableReplicas   int32        `json:"availableReplicas,omitempty" protobuf:"varint,4,opt,name=availableReplicas"`
	UnavailableReplicas int32        `json:"unavailableReplicas,omitempty" protobuf:"varint,5,opt,name=unavailableReplicas"`
	ReleaseOK           bool         `json:"releaseOK,omitempty"`
	PodStatus           *[]PodStatus `json:"podStatus,omitempty"`
}

type PodStatus struct {
	Name       string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	NodeName   string `json:"nodeName,omitempty" protobuf:"bytes,10,opt,name=nodeName"`
	Phase      bool   `json:"phase,omitempty"`
	Conditions bool   `json:"conditions,omitempty"`
}

// WorkLoadStatusReconciler reconciles a Plugin and cluster object.
type WorkLoadStatusReconciler struct {
	client.Client
	recorder        record.EventRecorder
	KubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugin,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugin/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch;update

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
	var releaseStatus = new(ReleaseStatus)
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, err
	}

	if plugin.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	clusterAccessReadyCondition, restClientGetter := r.initClientGetter(ctx, *plugin, plugin.Status)
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
		//pluginStatus := initPluginStatus(plugin)
		releaseStatus.ReleaseName = plugin.Name
		releaseStatus.ReleaseNamespace = plugin.Spec.ReleaseNamespace
		switch {
		// If the release namespace is not set, use the plugin namespace
		case plugin.Spec.ReleaseNamespace == "":
			releaseStatus.ReleaseNamespace = plugin.Namespace
		// If the plugin namespace is not set, use the namespace from the plugin metadata
		case plugin.Namespace == "":
			releaseStatus.ReleaseNamespace = plugin.ObjectMeta.Namespace
		}
		releaseStatus.ClusterName = plugin.Spec.ClusterName
		releaseStatus.HelmStatus = helmRelease.Info.Status.String()
		objectMap, _ := helm.ObjectMapFromManifest(restClientGetter, plugin.Namespace, helmRelease.Manifest, &helm.ManifestObjectFilter{})

		for key := range objectMap {
			if key.GVK.Version == "v1" &&
				(key.GVK.Group == "apps" || key.GVK.Group == "batch") ||
				(key.GVK.Group == "monitoring.coreos.com" && key.GVK.Kind == "Alertmanager") {
				getStatusPayload(ctx, releaseStatus, objClient, key.Name, releaseStatus.ReleaseNamespace, key.GVK)
			}
		}

		marshaled, err := json.Marshal(releaseStatus)
		if err != nil {
			log.FromContext(ctx).Error(err, "releaseStatus marshaling error")
		}
		//setPluginStatus(plugin, releaseStatus)
		if releaseStatus.ReleaseOK {
			log.FromContext(ctx).Info("Release is OK", "pluginName", releaseStatus.ReleaseName, "Details", string(marshaled))
		}
	}
	return ctrl.Result{RequeueAfter: StatusRequeueInterval}, nil
}

// initClientGetter returns a RestClientGetter for the given Plugin.
// If the Plugin has a clusterName set, the RestClientGetter is initialized from the cluster secret.
// Otherwise, the RestClientGetter is initialized with in-cluster config
func (r *WorkLoadStatusReconciler) initClientGetter(
	ctx context.Context,
	plugin greenhousev1alpha1.Plugin,
	pluginStatus greenhousev1alpha1.PluginStatus,
) (
	clusterAccessReadyCondition greenhousev1alpha1.Condition,
	restClientGetter genericclioptions.RESTClientGetter,
) {

	clusterAccessReadyCondition = *pluginStatus.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition)
	clusterAccessReadyCondition.Status = metav1.ConditionTrue

	var err error

	// early return if spec.clusterName is not set
	if plugin.Spec.ClusterName == "" {
		restClientGetter, err = clientutil.NewRestClientGetterForInCluster(plugin.GetReleaseNamespace(), r.kubeClientOpts...)
		if err != nil {
			clusterAccessReadyCondition.Status = metav1.ConditionFalse
			clusterAccessReadyCondition.Message = fmt.Sprintf("cannot access greenhouse cluster: %s", err.Error())
			return clusterAccessReadyCondition, nil
		}
		return clusterAccessReadyCondition, restClientGetter
	}

	cluster := new(greenhousev1alpha1.Cluster)
	err = r.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("Failed to get cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}

	readyConditionInCluster := cluster.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)
	if readyConditionInCluster == nil || readyConditionInCluster.Status != metav1.ConditionTrue {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("cluster %s is not ready", plugin.Spec.ClusterName)
		return clusterAccessReadyCondition, nil
	}

	// get restclientGetter from cluster if clusterName is set
	secret := corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, &secret)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("Failed to get secret for cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}
	restClientGetter, err = clientutil.NewRestClientGetterFromSecret(&secret, plugin.GetReleaseNamespace(), r.kubeClientOpts...)
	if err != nil {
		clusterAccessReadyCondition.Status = metav1.ConditionFalse
		clusterAccessReadyCondition.Message = fmt.Sprintf("cannot access cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		return clusterAccessReadyCondition, nil
	}
	clusterAccessReadyCondition.Status = metav1.ConditionTrue
	clusterAccessReadyCondition.Message = ""
	return clusterAccessReadyCondition, restClientGetter
}

// getStatusPayload fetches the status of the object and updates the ReleaseStatus object
func getStatusPayload(ctx context.Context, releaseStatus *ReleaseStatus, cl client.Client, objName, objNamespace string, GVK schema.GroupVersionKind) {
	releaseStatus.ReleaseOK = true
	switch GVK.Kind {
	case "Deployment":
		remoteObject := &appsv1.Deployment{}
		if err := cl.Get(context.TODO(), types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting deployment", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		releaseStatus.Replicas += remoteObject.Status.Replicas
		releaseStatus.UpdatedReplicas += remoteObject.Status.UpdatedReplicas
		releaseStatus.ReadyReplicas += remoteObject.Status.ReadyReplicas
		releaseStatus.AvailableReplicas += remoteObject.Status.AvailableReplicas
		releaseStatus.UnavailableReplicas += remoteObject.Status.UnavailableReplicas
		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
			releaseStatus.PodStatus = fetchPodList(remoteObject.Spec.Selector.MatchLabels, objNamespace, cl)
		}
	case "StatefulSet":
		remoteObject := &appsv1.StatefulSet{}
		if err := cl.Get(context.TODO(), types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting statefulset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		releaseStatus.Replicas += remoteObject.Status.Replicas
		releaseStatus.UpdatedReplicas += remoteObject.Status.UpdatedReplicas
		releaseStatus.ReadyReplicas += remoteObject.Status.ReadyReplicas
		releaseStatus.AvailableReplicas += remoteObject.Status.AvailableReplicas

		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
			releaseStatus.PodStatus = fetchPodList(remoteObject.Spec.Selector.MatchLabels, objNamespace, cl)
		}

	case "DaemonSet":
		remoteObject := &appsv1.DaemonSet{}
		if err := cl.Get(context.TODO(), types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting daemonset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		releaseStatus.Replicas += remoteObject.Status.DesiredNumberScheduled
		releaseStatus.UpdatedReplicas += remoteObject.Status.UpdatedNumberScheduled
		releaseStatus.ReadyReplicas += remoteObject.Status.NumberReady
		releaseStatus.AvailableReplicas += remoteObject.Status.NumberAvailable
		releaseStatus.UnavailableReplicas += remoteObject.Status.NumberUnavailable

		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
			releaseStatus.PodStatus = fetchPodList(remoteObject.Spec.Selector.MatchLabels, objNamespace, cl)
		}

	case "ReplicaSet":
		remoteObject := &appsv1.ReplicaSet{}
		if err := cl.Get(context.TODO(), types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting replicaset", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		releaseStatus.Replicas += remoteObject.Status.Replicas
		releaseStatus.ReadyReplicas += remoteObject.Status.ReadyReplicas
		releaseStatus.AvailableReplicas += remoteObject.Status.AvailableReplicas

		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
			releaseStatus.PodStatus = fetchPodList(remoteObject.Spec.Selector.MatchLabels, objNamespace, cl)
		}

	case "CronJob":
		remoteObject := &batchv1.CronJob{}
		if err := cl.Get(context.TODO(), types.NamespacedName{Name: objName, Namespace: objNamespace}, remoteObject); err != nil {
			log.FromContext(ctx).Error(err, "Error getting cronjob", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
		}
	case "Alertmanager":
		remoteObject := &corev1.PodList{}
		listOptions := &client.ListOptions{
			LabelSelector: labels.NewSelector(),
			Namespace:     objNamespace,
		}
		listOptions.LabelSelector.Matches(labels.Set{"meta.helm.sh/release-name": objName, "meta.helm.sh/release-namespace": objNamespace})
		notFromCronJob, err := labels.NewRequirement("batch.kubernetes.io/job-name", selection.DoesNotExist, []string{})
		if err != nil {
			log.FromContext(ctx).Error(err, "Error creating label selector", "name", objName, "pluginName", releaseStatus.ReleaseName)
		}
		listOptions.LabelSelector.Add(*notFromCronJob)
		if err := cl.List(context.TODO(), remoteObject, listOptions); err != nil {
			log.FromContext(ctx).Error(err, "Error getting alertmanager", "name", objName, "pluginName", releaseStatus.ReleaseName)
			return
		}
		if !isPayloadReadyRunning(remoteObject) {
			releaseStatus.ReleaseOK = false
		}
	}
}
