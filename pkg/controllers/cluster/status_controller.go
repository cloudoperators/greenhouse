// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"context"
	"fmt"
	"time"

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
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

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

	// patch message and condition
	result, err := clientutil.PatchStatus(ctx, r.Client, cluster, func() error {
		cluster.Status.KubernetesVersion = k8sVersion
		cluster.Status.SetConditions(readyCondition, allNodesReadyCondition, kubeConfigValidCondition)
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

func (r *ClusterStatusReconciler) reconcileReadyStatus(cluster *greenhousev1alpha1.Cluster, kubeConfigValidCondition greenhousev1alpha1.Condition) (readyCondition greenhousev1alpha1.Condition) {
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
				allNodesReadyCondition.Message = allNodesReadyCondition.Message + ", "
			}
			allNodesReadyCondition.Message = allNodesReadyCondition.Message + fmt.Sprintf("%s not ready", node.GetName())
		}
	}
	return
}
