// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const StatusRequeueInterval = 2 * time.Minute

// ClusterStatusReconciler reconciles the overall status of a remote cluster
type ClusterStatusReconciler struct {
	client.Client
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterStatusReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}).
		Complete(r)
}

func (r *ClusterStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cluster.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
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
	if deletionCondition.IsTrue() {
		conditions = append(conditions, deletionCondition)
	}

	// patch message and condition
	result, err := clientutil.PatchStatus(ctx, r.Client, cluster, func() error {
		cluster.Status.KubernetesVersion = k8sVersion
		cluster.Status.SetConditions(conditions...)
		cluster.Status.Nodes = clusterNodeStatus
		return nil
	})
	if err != nil {
		return reconcile.Result{}, err
	}
	if result != clientutil.OperationResultNone {
		logMessage := fmt.Sprintf("%s cluster.status", result)
		log.FromContext(ctx).V(5).Info(logMessage, "namespace", cluster.Namespace, "name", cluster.Name, "status", cluster.Status)
	}

	return ctrl.Result{RequeueAfter: StatusRequeueInterval}, nil
}

func (r *ClusterStatusReconciler) checkDeletionSchedule(logger logr.Logger, cluster *greenhousev1alpha1.Cluster) greenhousev1alpha1.Condition {
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

func (r *ClusterStatusReconciler) reconcileClusterSecret(
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

func (r *ClusterStatusReconciler) reconcileReadyStatus(kubeConfigValidCondition greenhousev1alpha1.Condition) (readyCondition greenhousev1alpha1.Condition) {
	readyCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ReadyCondition, "", "")

	if kubeConfigValidCondition.IsFalse() {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Message = "kubeconfig not valid - cannot access cluster"
	} else {
		readyCondition.Status = metav1.ConditionTrue
	}
	return
}

// reconcileNodeStatus returns the status of all nodes of the cluster and an all nodes ready condition.
func (r *ClusterStatusReconciler) reconcileNodeStatus(
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
