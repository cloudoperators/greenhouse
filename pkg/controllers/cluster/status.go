// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/cloudoperators/greenhouse/pkg/lifecycle"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

func (r *RemoteClusterReconciler) setConditions() lifecycle.Conditioner {
	return func(ctx context.Context, resource lifecycle.RuntimeObject) {
		logger := ctrl.LoggerFrom(ctx)
		cluster, ok := resource.(*greenhousev1alpha1.Cluster)
		if !ok {
			logger.Error(errors.New("invalid object type"), "object is not of Cluster type")
			return
		}
		if cluster.Spec.AccessMode != greenhousev1alpha1.ClusterAccessModeDirect {
			logger.Info("skipping status calculation for cluster with access mode " + string(cluster.Spec.AccessMode))
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

		readyCondition := r.reconcileReadyStatus(cluster, kubeConfigValidCondition)

		conditions = append(conditions, readyCondition, allNodesReadyCondition, kubeConfigValidCondition)

		deletionCondition := r.checkDeletionSchedule(logger, cluster)
		if deletionCondition.IsTrue() {
			conditions = append(conditions, deletionCondition)
		}
		cluster.Status.KubernetesVersion = k8sVersion
		cluster.Status.SetConditions(conditions...)
		cluster.Status.Nodes = clusterNodeStatus
	}
}

func (r *RemoteClusterReconciler) checkDeletionSchedule(logger logr.Logger, cluster *greenhousev1alpha1.Cluster) greenhousev1alpha1.Condition {
	deletionCondition := greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ClusterDeletionScheduled, "", "")
	scheduleExists, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		logger.Error(err, "failed to extract deletion schedule - ignoring deletion schedule")
	}
	if scheduleExists {
		deletionCondition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ClusterDeletionScheduled, "", "deletion scheduled at "+schedule.Format(time.DateTime))
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

	k8sVersion = "unknown"
	if cluster.Status.KubernetesVersion != "" {
		k8sVersion = cluster.Status.KubernetesVersion
	}

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
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
	}

	return
}

func (r *RemoteClusterReconciler) reconcileReadyStatus(cluster *greenhousev1alpha1.Cluster, kubeConfigValidCondition greenhousev1alpha1.Condition) (readyCondition greenhousev1alpha1.Condition) {
	readyCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ReadyCondition, "", "")

	if kubeConfigValidCondition.IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "kubeconfig not valid - cannot access cluster"
		// change ready condition message if headscale not ready
		if cluster.Spec.AccessMode == greenhousev1alpha1.ClusterAccessModeHeadscale {
			headscaleReadyCondition := cluster.Status.StatusConditions.GetConditionByType(greenhousev1alpha1.HeadscaleReady)
			if headscaleReadyCondition == nil || !headscaleReadyCondition.IsTrue() {
				readyCondition.Message = "Headscale connection not ready"
			}
		}
	} else {
		readyCondition.Status = metav1.ConditionTrue
	}
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
