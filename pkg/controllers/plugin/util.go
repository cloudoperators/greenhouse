// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

func getPortForExposedService(o runtime.Object) (*corev1.ServicePort, error) {
	svc, err := convertRuntimeObjectToCoreV1Service(o)
	if err != nil {
		return nil, err
	}

	if svc.Spec.Ports == nil || len(svc.Spec.Ports) == 0 {
		return nil, errors.New("service has no ports")
	}

	//Check for matching of named port set by label
	var namedPort string = svc.Labels[greenhouseapis.LabelKeyExposeNamedPort]

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

func convertRuntimeObjectToCoreV1Service(o interface{}) (*corev1.Service, error) {
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

func isPayloadReadyRunning(o interface{}) bool {
	switch obj := o.(type) {
	case *appsv1.Deployment:
		if obj.Status.ReadyReplicas == obj.Status.Replicas && obj.Status.ReadyReplicas == obj.Status.AvailableReplicas {
			return true
		}
	case *appsv1.StatefulSet:
		if obj.Status.ReadyReplicas == obj.Status.Replicas && obj.Status.ReadyReplicas == obj.Status.AvailableReplicas {
			return true
		}
	case *appsv1.DaemonSet:
		if obj.Status.NumberReady == obj.Status.DesiredNumberScheduled && obj.Status.NumberReady == obj.Status.NumberAvailable {
			return true
		}
	case *appsv1.ReplicaSet:
		if obj.Status.ReadyReplicas == obj.Status.Replicas && obj.Status.ReadyReplicas == obj.Status.AvailableReplicas {
			return true
		}
	case *batchv1.CronJob:
		// CronJob does not have a status field just for the job, so we need to check the last successful time
		if obj.Status.LastSuccessfulTime != nil {
			return true
		}
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

func fetchPodList(labelSelector map[string]string, nameSpace string, cl client.Client) *[]PodStatus {
	var podList = new(corev1.PodList)
	var podStatusList = new([]PodStatus)
	listOptions := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelSelector),
		Namespace:     nameSpace,
	}
	if err := cl.List(context.Background(), podList, listOptions); err != nil {
		return nil
	}

	for _, pod := range podList.Items {
		podStatus := PodStatus{
			Name:       pod.Name,
			NodeName:   pod.Spec.NodeName,
			Conditions: checkPodStatusCondition(pod.Status.Conditions),
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			podStatus.Phase = true
		case corev1.PodSucceeded:
			podStatus.Phase = true
		default:
			podStatus.Phase = false
		}
		*podStatusList = append(*podStatusList, podStatus)
	}
	return podStatusList
}

func checkPodStatusCondition(podCondition []corev1.PodCondition) bool {
	for _, condition := range podCondition {
		switch condition.Type {
		case corev1.PodInitialized:
			if condition.Status != corev1.ConditionTrue {
				return false
			}
		case corev1.PodReady:
			if condition.Status != corev1.ConditionTrue {
				return false
			}
		case corev1.ContainersReady:
			if condition.Status != corev1.ConditionTrue {
				return false
			}
		case corev1.PodScheduled:
			if condition.Status != corev1.ConditionTrue {
				return false
			}
		}
	}
	return true
}

func composeMessage(podStatusList *[]PodStatus) string {
	var message string
	for _, pod := range *podStatusList {
		if pod.Conditions == false {
			message = message + ",Pod Name: " + pod.Name + " on Node: " + pod.NodeName + "\n"
		}
	}
	return message
}

func computeReadyCondition(pluginStatus greenhousev1alpha1.PluginStatus, release *ReleaseStatus) greenhousev1alpha1.Condition {
	WorkloadReadyStatus := *pluginStatus.GetConditionByType(greenhousev1alpha1.WorkloadReadyCondition)
	if pluginStatus.GetConditionByType(greenhousev1alpha1.WorkloadReadyCondition) == nil {
		pluginStatus.SetConditions(greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.WorkloadReadyCondition, "", ""))
	}

	if release.ReleaseOK {
		WorkloadReadyStatus.Status = metav1.ConditionTrue
		WorkloadReadyStatus.Message = release.Message
	} else {
		WorkloadReadyStatus.Status = metav1.ConditionFalse
		WorkloadReadyStatus.Message = release.Message
	}
	return WorkloadReadyStatus
}

func logPluginStatus(ctx context.Context, releaseStatus *ReleaseStatus) {
	marshaled, err := json.Marshal(releaseStatus)
	if err != nil {
		log.FromContext(ctx).Info("releaseStatus marshaling error", "error", err)
	}
	if releaseStatus.ReleaseOK {
		log.FromContext(ctx).Info("release is running", "pluginName", releaseStatus.ReleaseName, "Details", string(marshaled))
	} else {
		log.FromContext(ctx).Info("release is not running", "pluginName", releaseStatus.ReleaseName, "Details", string(marshaled))
	}
}
