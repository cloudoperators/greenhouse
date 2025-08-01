// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"fmt"
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

		var kubeConfigValidCondition greenhousemetav1alpha1.Condition
		var k8sVersion string
		clusterSecret, restClientGetter, err := r.getClusterSecretAndClientGetter(ctx, cluster)
		if err != nil {
			kubeConfigValidCondition = greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", err.Error())
			k8sVersion = clusterK8sVersionUnknown
		} else {
			kubeConfigValidCondition, k8sVersion = r.reconcileKubeConfigValid(restClientGetter)
		}

		var allNodesReadyCondition, clusterAccessibleCondition, resourcesDeployedCondition greenhousemetav1alpha1.Condition
		var nodes *greenhousev1alpha1.Nodes
		// Can only reconcile detailed status if kubeconfig is valid.
		if kubeConfigValidCondition.IsFalse() {
			allNodesReadyCondition = greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.AllNodesReady, "", "kubeconfig not valid - cannot know node status")
			clusterAccessibleCondition = greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.PermissionsVerified, "", "kubeconfig not valid - cannot validate cluster access")
			resourcesDeployedCondition = greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "kubeconfig not valid - cannot validate managed resources")
		} else {
			allNodesReadyCondition, nodes = r.reconcileNodeStatus(ctx, restClientGetter)
			clusterAccessibleCondition = r.reconcilePermissions(ctx, restClientGetter)
			resourcesDeployedCondition = r.reconcileBootstrapResources(ctx, restClientGetter, clusterSecret)
		}

		readyCondition := r.reconcileReadyStatus(kubeConfigValidCondition, resourcesDeployedCondition)

		ownerLabelCondition := util.ComputeOwnerLabelCondition(ctx, r.Client, cluster)
		util.UpdateOwnedByLabelMissingMetric(cluster, ownerLabelCondition.IsFalse())

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

		cluster.Status.Nodes = nodes
	}
}

func (r *RemoteClusterReconciler) getClusterSecretAndClientGetter(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (*corev1.Secret, genericclioptions.RESTClientGetter, error) {
	clusterSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return nil, nil, fmt.Errorf("failed to get cluster secret: %w", err)
	}

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(clusterSecret, clusterSecret.Namespace, clientutil.WithPersistentConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create REST client getter: %w", err)
	}

	return clusterSecret, restClientGetter, nil
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

func (r *RemoteClusterReconciler) reconcileBootstrapResources(ctx context.Context, clientGetter genericclioptions.RESTClientGetter, secret *corev1.Secret) greenhousemetav1alpha1.Condition {
	if secret == nil {
		return greenhousemetav1alpha1.UnknownCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "managed resources could not be validated")
	}

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(clientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error())
	}

	if err := remoteClient.Get(ctx, client.ObjectKey{Name: secret.GetNamespace()}, &corev1.Namespace{}); err != nil {
		if apierrors.IsNotFound(err) {
			return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "",
				fmt.Sprintf("Namespace %s not found in remote cluster", secret.GetNamespace()))
		}

		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error())
	}

	if secret.Type != greenhouseapis.SecretTypeOIDCConfig {
		if err := remoteClient.Get(ctx, client.ObjectKey{Namespace: secret.GetNamespace(), Name: utils.ServiceAccountName}, &corev1.ServiceAccount{}); err != nil {
			if apierrors.IsNotFound(err) {
				return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "",
					fmt.Sprintf("ServiceAccount %s in namespace %s not found in remote cluster", utils.ServiceAccountName, secret.GetNamespace()))
			}

			return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error())
		}

		if err := remoteClient.Get(ctx, client.ObjectKey{Name: utils.ServiceAccountName}, &rbacv1.ClusterRoleBinding{}); err != nil {
			if apierrors.IsNotFound(err) {
				return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "",
					fmt.Sprintf("ClusterRoleBinding %s not found in remote cluster", utils.ServiceAccountName))
			}

			return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", err.Error())
		}
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.ManagedResourcesDeployed, "", "")
}

func (r *RemoteClusterReconciler) reconcilePermissions(ctx context.Context, clientGetter genericclioptions.RESTClientGetter) greenhousemetav1alpha1.Condition {
	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(clientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.PermissionsVerified, "", err.Error())
	}

	missing := common.CheckClientClusterPermission(ctx, remoteClient, "", corev1.NamespaceAll)
	if len(missing) > 0 {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.PermissionsVerified, "", "missing cluster admin permission")
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.PermissionsVerified, "", "ServiceAccount has cluster admin permissions")
}

func (r *RemoteClusterReconciler) reconcileKubeConfigValid(restClientGetter genericclioptions.RESTClientGetter) (condition greenhousemetav1alpha1.Condition, version string) {
	kubernetesVersion, err := clientutil.GetKubernetesVersion(restClientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", err.Error()), clusterK8sVersionUnknown
	}

	return greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.KubeConfigValid, "", ""), kubernetesVersion.String()
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

// reconcileNodeStatus fetches the list of nodes from a remote cluster,
// evaluates each node's readiness, and compiles summary metrics and statuses.
// It returns:
//   - allNodesReadyCondition: a Condition indicating if all nodes are ready (true) or not (false), with a message summarizing non-ready nodes.
//   - nodes: a Nodes struct containing total count, ready count, and a map of non-ready node statuses.
func (r *RemoteClusterReconciler) reconcileNodeStatus(
	ctx context.Context,
	restClientGetter genericclioptions.RESTClientGetter,
) (
	allNodesReadyCondition greenhousemetav1alpha1.Condition,
	nodes *greenhousev1alpha1.Nodes,
) {

	remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	if err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.AllNodesReady, "", err.Error()), nil
	}

	nodeList := &corev1.NodeList{}
	if err := remoteClient.List(ctx, nodeList); err != nil {
		return greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.AllNodesReady, "", err.Error()), nil
	}

	notReadyNodes := []greenhousev1alpha1.NodeStatus{}
	var totalNodes, readyNodes int32
	allNodesReadyCondition = greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", "")

	for _, node := range nodeList.Items {
		totalNodes++

		nodeStatus := getNodeStatusIfNodeNotReady(node)
		if nodeStatus == nil {
			readyNodes++
			continue
		}

		notReadyNodes = append(notReadyNodes, *nodeStatus)

		allNodesReadyCondition.Status = metav1.ConditionFalse
		if allNodesReadyCondition.Message != "" {
			allNodesReadyCondition.Message += ", "
		}

		allNodesReadyCondition.Message += node.GetName() + " not ready"
	}

	return allNodesReadyCondition, &greenhousev1alpha1.Nodes{Total: totalNodes, Ready: readyNodes, NotReady: notReadyNodes}
}

// getNodeStatusIfNodeNotReady returns a NodeStatus when the NodeReady condition is explicitly False; otherwise nil.
func getNodeStatusIfNodeNotReady(node corev1.Node) *greenhousev1alpha1.NodeStatus {
	var readyCondition *corev1.NodeCondition
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == corev1.NodeReady {
			readyCondition = &node.Status.Conditions[i]
			break
		}
	}

	if readyCondition == nil || readyCondition.Status == corev1.ConditionTrue {
		return nil
	}

	return &greenhousev1alpha1.NodeStatus{Name: node.Name, Message: readyCondition.Message, LastTransitionTime: readyCondition.LastTransitionTime}
}
