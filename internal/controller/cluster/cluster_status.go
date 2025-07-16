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
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	"github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/util"
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

		kubeConfigValidCondition, restClientGetter, k8sVersion, clusterSecret := r.reconcileClusterSecret(ctx, cluster)
		clusterAccessibleCondition := r.reconcileAccessibility(ctx, restClientGetter)
		resourcesDeployedCondition := r.reconcileBootstrapResources(ctx, cluster, restClientGetter, clusterSecret)

		allNodesReadyCondition := greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.AllNodesReady, "", "")
		clusterNodeStatus := make(map[string]greenhousev1alpha1.NodeStatus)
		// Can only reconcile node status if kubeconfig is valid
		if restClientGetter == nil || kubeConfigValidCondition.IsFalse() {
			allNodesReadyCondition.Message = "kubeconfig not valid - cannot know node status"
		} else {
			allNodesReadyCondition, clusterNodeStatus = r.reconcileNodeStatus(ctx, restClientGetter)
		}

		readyCondition := r.reconcileReadyStatus(kubeConfigValidCondition, clusterAccessibleCondition, resourcesDeployedCondition)

		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, cluster)

		conditions := []greenhousemetav1alpha1.Condition{
			readyCondition,
			allNodesReadyCondition,
			kubeConfigValidCondition,
			ownerLabelCondition,
			clusterAccessibleCondition,
			resourcesDeployedCondition,
		}
		deletionCondition := r.checkDeletionSchedule(logger, cluster)
		if !deletionCondition.IsUnknown() {
			conditions = append(conditions, deletionCondition)
		}
		cluster.Status.KubernetesVersion = k8sVersion
		cluster.Status.SetConditions(conditions...)
		cluster.Status.Nodes = clusterNodeStatus
	}
}

func (r *RemoteClusterReconciler) checkDeletionSchedule(logger logr.Logger, cluster *greenhousev1alpha1.Cluster) greenhousemetav1alpha1.Condition {
	deletionCondition := greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.DeleteCondition, "", "")
	scheduleExists, schedule, err := clientutil.ExtractDeletionSchedule(cluster.GetAnnotations())
	if err != nil {
		logger.Error(err, "failed to extract deletion schedule - ignoring deletion schedule")
	}
	if scheduleExists {
		deletionCondition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.ScheduledDeletionReason, "deletion scheduled at "+schedule.Format(time.DateTime))
	} else {
		// Remove the deletion condition if it exists as the deletion schedule annotation has been removed
		cluster.Status.Conditions = slices.DeleteFunc(cluster.Status.Conditions, func(condition greenhousemetav1alpha1.Condition) bool {
			return condition.Type == greenhousemetav1alpha1.DeleteCondition && condition.IsFalse()
		})
	}
	return deletionCondition
}

func (r *RemoteClusterReconciler) reconcileBootstrapResources(ctx context.Context, cluster *greenhousev1alpha1.Cluster, clientGetter genericclioptions.RESTClientGetter, secret *corev1.Secret) greenhousemetav1alpha1.Condition {
	if clientGetter == nil || secret == nil {
		return greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "managed resources could not be validated")
	}

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(clientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error())
	}

	if err := remoteClient.Get(ctx, client.ObjectKey{Name: cluster.GetNamespace()}, &corev1.Namespace{}); err != nil {
		condition := greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "")
		if apierrors.IsNotFound(err) {
			condition.Message = "Namespace not found in remote cluster"
		} else {
			condition.Message = err.Error()
		}

		return condition
	}

	if secret.Type != greenhouseapis.SecretTypeOIDCConfig {
		if err := remoteClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: utils.ServiceAccountName}, &corev1.ServiceAccount{}); err != nil {
			condition := greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "")
			if apierrors.IsNotFound(err) {
				condition.Message = "ServiceAccount not found in remote cluster"
			} else {
				condition.Message = err.Error()
			}

			return condition
		}

		if err := remoteClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName}, &rbacv1.ClusterRoleBinding{}); err != nil {
			condition := greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "")
			if apierrors.IsNotFound(err) {
				condition.Message = "ClusterRoleBinding not found in remote cluster"
			} else {
				condition.Message = err.Error()
			}

			return condition
		}
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "")
}

func (r *RemoteClusterReconciler) reconcileAccessibility(ctx context.Context, clientGetter genericclioptions.RESTClientGetter) greenhousemetav1alpha1.Condition {
	if clientGetter == nil {
		return greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.Accessible, "", "accessibility could not be validated")
	}

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(clientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.Accessible, "", err.Error())
	}

	missing := common.CheckClientClusterPermission(ctx, remoteClient, "", corev1.NamespaceAll)
	if len(missing) > 0 {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.Accessible, "", "missing cluster admin permission")
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.Accessible, "", "")
}

func (r *RemoteClusterReconciler) reconcileClusterSecret(
	ctx context.Context,
	cluster *greenhousev1alpha1.Cluster,
) (
	kubeConfigValidCondition greenhousemetav1alpha1.Condition,
	restClientGetter genericclioptions.RESTClientGetter,
	k8sVersion string,
	clusterSecret *corev1.Secret,
) {

	clusterSecret = new(corev1.Secret)
	kubeConfigValidCondition = greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.KubeConfigValid, "", "")
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
		return kubeConfigValidCondition, restClientGetter, k8sVersion, nil
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, clusterSecret.Namespace, clientutil.WithPersistentConfig())
	if err != nil {
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
		return
	}

	kubernetesVersion, err := clientutil.GetKubernetesVersion(restClientGetter)
	if err != nil {
		k8sVersion = clusterK8sVersionUnknown
		kubeConfigValidCondition.Status = metav1.ConditionFalse
		kubeConfigValidCondition.Message = err.Error()
		return
	}

	k8sVersion = kubernetesVersion.String()
	kubeConfigValidCondition.Status = metav1.ConditionTrue
	return
}

func (r *RemoteClusterReconciler) reconcileReadyStatus(conditions ...greenhousemetav1alpha1.Condition) (readyCondition greenhousemetav1alpha1.Condition) {
	readyCondition = greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, "", "")
	for _, condition := range conditions {
		if condition.IsFalse() {
			readyCondition.Status = metav1.ConditionFalse
			readyCondition.Message = "cannot access cluster"
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
	allNodesReadyCondition greenhousemetav1alpha1.Condition,
	clusterNodeStatus map[string]greenhousev1alpha1.NodeStatus,
) {

	clusterNodeStatus = make(map[string]greenhousev1alpha1.NodeStatus)
	allNodesReadyCondition = greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", "")

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
		greenhouseNodeStatusConditions := greenhousemetav1alpha1.StatusConditions{}
		for _, condition := range node.Status.Conditions {
			greenhouseNodeStatusConditions.SetConditions(greenhousemetav1alpha1.Condition{
				Type:               greenhousemetav1alpha1.ConditionType(condition.Type),
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
