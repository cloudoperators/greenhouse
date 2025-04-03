// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const clusterK8sVersionUnknown = "unknown"

func (r *RemoteClusterReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		cluster, ok := resource.(*greenhousev1alpha1.Cluster)
		if !ok {
			logger.Error(errors.New("resource is not a cluster"), "status setup failed")
			return
		}
		var conditions []greenhousev1alpha1.Condition

		kubeConfigValidCondition, restClientGetter, k8sVersion := r.reconcileClusterSecret(ctx, cluster)

		allNodesReadyCondition := greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.AllNodesReady, "", "")
		clusterNodeStatus := make(map[string]greenhousev1alpha1.NodeStatus)
		// Can only reconcile node status if kubeconfig is valid
		if restClientGetter == nil || kubeConfigValidCondition.IsFalse() {
			allNodesReadyCondition.Message = "kubeconfig not valid - cannot know node status"
		} else {
			allNodesReadyCondition, clusterNodeStatus = r.reconcileNodeStatus(ctx, restClientGetter)
		}

		readyCondition := r.reconcileReadyStatus(kubeConfigValidCondition)

		conditions = append(conditions, readyCondition, allNodesReadyCondition, kubeConfigValidCondition)

		deletionCondition := r.checkDeletionSchedule(logger, cluster)
		if !deletionCondition.IsUnknown() {
			conditions = append(conditions, deletionCondition)
		}
		cluster.Status.KubernetesVersion = k8sVersion
		cluster.Status.SetConditions(conditions...)
		cluster.Status.Nodes = clusterNodeStatus
	}
}

func (r *RemoteClusterReconciler) checkDeletionSchedule(logger logr.Logger, cluster *greenhousev1alpha1.Cluster) greenhousev1alpha1.Condition {
	deletionCondition := greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.DeleteCondition, "", "")
	scheduleExists, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		logger.Error(err, "failed to extract deletion schedule - ignoring deletion schedule")
	}
	if scheduleExists {
		deletionCondition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, lifecycle.ScheduledDeletionReason, "deletion scheduled at "+schedule.Format(time.DateTime))
	} else {
		// Remove the deletion condition if it exists as the deletion schedule annotation has been removed
		cluster.Status.StatusConditions.Conditions = slices.DeleteFunc(cluster.Status.StatusConditions.Conditions, func(condition greenhousev1alpha1.Condition) bool {
			return condition.Type == greenhousev1alpha1.DeleteCondition && condition.IsFalse()
		})
	}
	return deletionCondition
}

func (r *RemoteClusterReconciler) reconcileClusterSecret(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
) (
	kubeConfigValidCondition greenhousev1alpha1.Condition,
	restClientGetter genericclioptions.RESTClientGetter,
	k8sVersion string,
) {

	kubeConfigValidCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.KubeConfigValid, "", "")
	var clusterSecret = new(corev1.Secret)
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
		return
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, clusterSecret.Namespace, clientutil.WithPersistentConfig())
	if err != nil {
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
		return
	}

	if kubernetesVersion, err := clientutil.GetKubernetesVersion(restClientGetter); err == nil {
		k8sVersion = kubernetesVersion.String()
		kubeConfigValidCondition.Status = metav1.ConditionTrue
	} else {
		k8sVersion = clusterK8sVersionUnknown
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
	}

	return
}

func (r *RemoteClusterReconciler) reconcileReadyStatus(conditions ...greenhousev1alpha1.Condition) (readyCondition greenhousev1alpha1.Condition) {
	readyCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ReadyCondition, "", "")
	for _, condition := range conditions {
		if condition.IsFalse() {
			readyCondition.Status = metav1.ConditionFalse
			readyCondition.Message = "kubeconfig not valid - cannot access cluster"
			if condition.Message != "" {
				readyCondition.Message = condition.Message
			}
			return
		}
	}
	readyCondition.Status = metav1.ConditionTrue
	return
}

// reconcileNodeStatus returns the status of all nodes of the cluster and an all nodes ready condition.
func (r *RemoteClusterReconciler) reconcileNodeStatus(
	ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
) (
	allNodesReadyCondition greenhousev1alpha1.Condition,
	clusterNodeStatus map[string]greenhousev1alpha1.NodeStatus,
) {

	clusterNodeStatus = make(map[string]greenhousev1alpha1.NodeStatus)
	allNodesReadyCondition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", "")

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		allNodesReadyCondition.Status = metav1.ConditionFalse
		allNodesReadyCondition.Message = err.Error()
		return
	}

	nodeList := &corev1.NodeList{}

	if err := remoteClient.List(ctx, nodeList); err != nil {
		allNodesReadyCondition.Status = metav1.ConditionFalse
		allNodesReadyCondition.Message = err.Error()
		return
	}

	for _, node := range nodeList.Items {
		greenhouseNodeStatusConditions := greenhousev1alpha1.StatusConditions{}
		for _, condition := range node.Status.Conditions {
			greenhouseNodeStatusConditions.SetConditions(greenhousev1alpha1.Condition{
				Type:               greenhousev1alpha1.ConditionType(condition.Type),
				Status:             metav1.ConditionStatus(condition.Status),
				LastTransitionTime: condition.LastTransitionTime,
				Message:            condition.Message,
			})
		}

		nodeReady := greenhouseNodeStatusConditions.IsReadyTrue()

		clusterNodeStatus[node.GetName()] = greenhousev1alpha1.NodeStatus{
			Ready:            greenhouseNodeStatusConditions.IsReadyTrue(),
			StatusConditions: greenhouseNodeStatusConditions,
		}

		if !nodeReady {
			allNodesReadyCondition.Status = metav1.ConditionFalse
			if allNodesReadyCondition.Message != "" {
				allNodesReadyCondition.Message += ", "
			}
			allNodesReadyCondition.Message += node.GetName() + " not ready"
		}
	}
	return
}
