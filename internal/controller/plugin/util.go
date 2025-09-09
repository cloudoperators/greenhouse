// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const (
	deliveryToolLabel = "greenhouse.sap/deployment-tool"
)

// exposedConditions are the conditions that are exposed in the StatusConditions of the Plugin.
var exposedConditions = []greenhousemetav1alpha1.ConditionType{
	greenhousemetav1alpha1.ReadyCondition,
	greenhousev1alpha1.ClusterAccessReadyCondition,
	greenhousev1alpha1.HelmDriftDetectedCondition,
	greenhousev1alpha1.HelmReconcileFailedCondition,
	greenhousev1alpha1.StatusUpToDateCondition,
	greenhousev1alpha1.HelmChartTestSucceededCondition,
	greenhousev1alpha1.WorkloadReadyCondition,
	greenhousemetav1alpha1.OwnerLabelSetCondition,
}

type reconcileResult struct {
	requeueAfter time.Duration
}

// InitPluginStatus initializes all empty Plugin Conditions to "unknown"
func InitPluginStatus(plugin *greenhousev1alpha1.Plugin) greenhousev1alpha1.PluginStatus {
	for _, t := range exposedConditions {
		if plugin.Status.GetConditionByType(t) == nil {
			plugin.SetCondition(greenhousemetav1alpha1.UnknownCondition(t, "", ""))
		}
	}
	if plugin.Status.HelmReleaseStatus == nil {
		plugin.Status.HelmReleaseStatus = &greenhousev1alpha1.HelmReleaseStatus{Status: "unknown"}
	}
	return plugin.Status
}

// InitClientGetter returns a RestClientGetter for the given Plugin.
// If the Plugin has a clusterName set, the RestClientGetter is initialized from the cluster secret.
// Otherwise, the RestClientGetter is initialized with in-cluster config
func InitClientGetter(
	ctx context.Context,
	k8sClient client.Client,
	kubeClientOpts []clientutil.KubeClientOption,
	plugin greenhousev1alpha1.Plugin,
) (genericclioptions.RESTClientGetter, error) {

	// early return if spec.clusterName is not set
	if plugin.Spec.ClusterName == "" {
		restClientGetter, err := clientutil.NewRestClientGetterForInCluster(plugin.Spec.ReleaseNamespace, kubeClientOpts...)
		if err != nil {
			errorMessage := "cannot access greenhouse cluster: " + err.Error()
			plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
			return nil, errors.New(errorMessage)
		}
		plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
		return restClientGetter, nil
	}

	cluster := new(greenhousev1alpha1.Cluster)
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}

	readyConditionInCluster := cluster.Status.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
	if readyConditionInCluster == nil || readyConditionInCluster.Status != metav1.ConditionTrue {
		errorMessage := fmt.Sprintf("cluster %s is not ready", plugin.Spec.ClusterName)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}

	// get restclientGetter from cluster if clusterName is set
	secret := corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, &secret)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to get secret for cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}
	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(&secret, plugin.Spec.ReleaseNamespace, kubeClientOpts...)
	if err != nil {
		errorMessage := fmt.Sprintf("cannot access cluster %s: %s", plugin.Spec.ClusterName, err.Error())
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(
			greenhousev1alpha1.ClusterAccessReadyCondition, "", errorMessage))
		return nil, errors.New(errorMessage)
	}
	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(
		greenhousev1alpha1.ClusterAccessReadyCondition, "", ""))
	return restClientGetter, nil
}

func getPortForExposedService(o runtime.Object) (*corev1.ServicePort, error) {
	svc, err := convertRuntimeObjectToCoreV1Service(o)
	if err != nil {
		return nil, err
	}

	if len(svc.Spec.Ports) == 0 {
		return nil, errors.New("service has no ports")
	}

	// Check for matching of named port set by label
	var namedPort = svc.Labels[greenhouseapis.LabelKeyExposeNamedPort]

	if namedPort != "" {
		for _, port := range svc.Spec.Ports {
			if port.Name == namedPort {
				return port.DeepCopy(), nil
			}
		}
	}

	// Default to first port
	return svc.Spec.Ports[0].DeepCopy(), nil
}

func convertRuntimeObjectToCoreV1Service(o any) (*corev1.Service, error) {
	switch obj := o.(type) {
	case *corev1.Service:
		// If it's already a corev1.Service, no conversion needed
		return obj, nil
	case *unstructured.Unstructured:
		// If it's an unstructured object, convert it to corev1.Service
		var service corev1.Service
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &service)
		return &service, errors.Wrap(err, "failed to convert to corev1.Service from unstructured object")
	default:
		return nil, fmt.Errorf("unsupported runtime.Object type: %T", obj)
	}
}

// isPayloadReadyRunning checking if the payload is ready and running
func isPayloadReadyRunning(o any) bool {
	switch obj := o.(type) {
	case *appsv1.Deployment:
		if (obj.Status.ReadyReplicas == obj.Status.Replicas) && (obj.Status.Replicas == obj.Status.AvailableReplicas) {
			return true
		}
	case *appsv1.StatefulSet:
		if (obj.Status.ReadyReplicas == obj.Status.Replicas) && (obj.Status.Replicas == obj.Status.AvailableReplicas) {
			return true
		}
	case *appsv1.DaemonSet:
		if (obj.Status.NumberReady == obj.Status.DesiredNumberScheduled) && (obj.Status.DesiredNumberScheduled == obj.Status.NumberAvailable) {
			return true
		}
	case *appsv1.ReplicaSet:
		if (obj.Status.ReadyReplicas == obj.Status.Replicas) && (obj.Status.Replicas == obj.Status.AvailableReplicas) {
			return true
		}
	case *batchv1.Job:
		if obj.Status.CompletionTime != nil {
			return true
		}
	case *batchv1.CronJob:
		// CronJob does not have a status field just for the job, so we need to check the last successful time
		if obj.Status.LastSuccessfulTime == obj.Status.LastScheduleTime {
			return true
		}
	case *corev1.Pod:
		if obj.Status.Phase != corev1.PodRunning {
			return false
		}
		return true
	case *corev1.PodList:
		// Check if all pods are running, if one of them is not running, return false
		for _, pod := range obj.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false
			}
		}
		return true
	}
	return false
}

// allResourceReady checks if all resources are ready
func allResourceReady(payloadStatus []PayloadStatus) bool {
	for _, status := range payloadStatus {
		if !status.Ready {
			return false
		}
	}
	return true
}

// computeWorkloadCondition computes the ReadyCondition for the Plugin and sets the workload metrics and condition message.
func computeWorkloadCondition(plugin *greenhousev1alpha1.Plugin, release *ReleaseStatus) {
	if !allResourceReady(release.PayloadStatus) {
		UpdatePluginWorkloadMetrics(plugin, 0)
		errorMessage := "Following workload resources are not ready: [ "
		for _, status := range release.PayloadStatus {
			if !status.Ready {
				errorMessage += ", " + status.Message
			}
		}
		errorMessage = strings.TrimPrefix(errorMessage, ", ")
		errorMessage += " ]"
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.WorkloadReadyCondition, "", errorMessage))
		return
	}

	UpdatePluginWorkloadMetrics(plugin, 1)
	plugin.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.WorkloadReadyCondition, "", "Workload is running"))
}

// computeReadyCondition computes the ReadyCondition for the Plugin based on various status conditions
func computeReadyCondition(
	conditions greenhousemetav1alpha1.StatusConditions,
) (readyCondition greenhousemetav1alpha1.Condition) {

	readyCondition = *conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)

	// If the Cluster is not ready, the Plugin could not be ready
	if conditions.GetConditionByType(greenhousev1alpha1.ClusterAccessReadyCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "cluster access not ready"
		return readyCondition
	}
	// If the Helm reconcile failed, the Plugin is not up to date / ready
	if conditions.GetConditionByType(greenhousev1alpha1.HelmReconcileFailedCondition).IsTrue() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm reconcile failed"
		return readyCondition
	}
	if conditions.GetConditionByType(greenhousev1alpha1.HelmChartTestSucceededCondition).IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "Helm Chart Test failed"
		return readyCondition
	}
	// If the Workload deployed by the Plugin is not ready, the Plugin is not ready
	workloadCondition := conditions.GetConditionByType(greenhousev1alpha1.WorkloadReadyCondition)
	if workloadCondition.IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = workloadCondition.Message
		return readyCondition
	}
	// In other cases, the Plugin is ready
	readyCondition.Status = metav1.ConditionTrue
	readyCondition.Message = "ready"
	return readyCondition
}

func shouldReconcileOrRequeue(ctx context.Context, c client.Client, plugin *greenhousev1alpha1.Plugin) (*reconcileResult, error) {
	logger := ctrl.LoggerFrom(ctx)
	if plugin.Spec.ClusterName == "" {
		logger.Info("plugin does not have a clusterName set, will skip requeue")
		return nil, nil
	}
	cluster := &greenhousev1alpha1.Cluster{}
	err := c.Get(ctx, types.NamespacedName{Namespace: plugin.Namespace, Name: plugin.Spec.ClusterName}, cluster)
	if err != nil {
		return nil, err
	}
	scheduleExists, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		return nil, err
	}
	if scheduleExists {
		msg := fmt.Sprintf("cluster %s is scheduled for deletion at %s", plugin.Spec.ClusterName, schedule)
		plugin.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.ScheduledDeletionReason, msg))
		requeueAfter := time.Until(schedule)
		return &reconcileResult{
			requeueAfter: requeueAfter,
		}, nil
	}

	return nil, nil
}
