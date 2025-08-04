// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"reflect"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/helm"
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

func (r *PluginReconciler) reconcilePluginWorkloadStatus(
	ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
	plugin *greenhousev1alpha1.Plugin,
	pluginDefinition *greenhousev1alpha1.ClusterPluginDefinition,
) (*reconcileResult, error) {

	var releaseStatus = new(ReleaseStatus)

	// Nothing to do when the status of the plugin is empty and when the plugin does not have a Helm Chart
	if reflect.DeepEqual(plugin.Status, greenhousev1alpha1.PluginStatus{}) || plugin.Status.HelmChart == nil {
		return nil, nil
	}
	if pluginDefinition.Spec.HelmChart == nil {
		return nil, nil
	}

	objClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return nil, err
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

		computeWorkloadCondition(plugin, releaseStatus)
	}
	return &reconcileResult{requeueAfter: StatusRequeueInterval}, nil
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
