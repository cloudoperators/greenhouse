// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const controlPlaneNodeLabel = "node-role.kubernetes.io/control-plane"

func (p *Phase) ensureNodesReady(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(ctx context.Context) (lifecycle.Result, error) {
		remoteClient, err := clientutil.NewK8sClientFromRestClientGetter(p.RestClientGetter)
		if err != nil {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.AllNodesReady, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		nodeList := &corev1.NodeList{}
		if err := remoteClient.List(ctx, nodeList); err != nil {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.AllNodesReady, "", err.Error(),
			))
			return lifecycle.Continue(), nil
		}

		allReady := greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", "")
		var notReadyNodes []greenhousev1alpha1.NodeStatus
		var totalNodes, readyNodes int

		for _, node := range nodeList.Items {
			if _, ok := node.Labels[controlPlaneNodeLabel]; ok {
				continue
			}
			totalNodes++
			ns := getNodeStatusIfNotReady(node)
			if ns == nil {
				readyNodes++
				continue
			}
			notReadyNodes = append(notReadyNodes, *ns)
			allReady.Status = metav1.ConditionFalse
			if allReady.Message != "" {
				allReady.Message += ", "
			}
			allReady.Message += node.GetName() + " not ready"
		}

		cluster.SetCondition(allReady)
		cluster.Status.Nodes = &greenhousev1alpha1.Nodes{
			Total:    totalNodes,
			Ready:    readyNodes,
			NotReady: notReadyNodes,
		}
		return lifecycle.Continue(), nil
	}
}

// getNodeStatusIfNotReady returns a NodeStatus when the node is not ready; nil otherwise.
func getNodeStatusIfNotReady(node corev1.Node) *greenhousev1alpha1.NodeStatus {
	if node.Spec.Unschedulable {
		return &greenhousev1alpha1.NodeStatus{Name: node.Name, Message: "Node is unschedulable", LastTransitionTime: metav1.Now()}
	}
	for i := range node.Status.Conditions {
		c := &node.Status.Conditions[i]
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				return nil
			}
			return &greenhousev1alpha1.NodeStatus{Name: node.Name, Message: c.Message, LastTransitionTime: c.LastTransitionTime}
		}
	}
	return nil
}
