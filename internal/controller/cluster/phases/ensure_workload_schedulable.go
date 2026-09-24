// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

func (p *Phase) ensureWorkloadSchedulable(cluster *greenhousev1alpha1.Cluster) lifecycle.SubRoutine {
	return func(_ context.Context) (lifecycle.Result, error) {
		if cluster.Spec.Mode == greenhousev1alpha1.ClusterModeWorkerless {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PayloadSchedulable, greenhousev1alpha1.WorkerlessClusterReason,
				"cluster has no worker nodes - plugin workloads cannot be scheduled",
			))
			return lifecycle.Continue(), nil
		}

		kubeConfigValid := cluster.Status.GetConditionByType(greenhousev1alpha1.KubeConfigValid)
		if kubeConfigValid != nil && kubeConfigValid.IsFalse() {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PayloadSchedulable, "", "kubeconfig not valid - payloads cannot be scheduled",
			))
			return lifecycle.Continue(), nil
		}

		allNodesReady := cluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)
		if allNodesReady != nil && allNodesReady.IsFalse() && cluster.Status.Nodes == nil {
			// node observation failed (remote client or list error) — cannot verify schedulability
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PayloadSchedulable, "", "node observation failed - payloads cannot be scheduled: "+allNodesReady.Message,
			))
			return lifecycle.Continue(), nil
		}

		if cluster.Status.Nodes != nil && cluster.Status.Nodes.Ready == 0 && cluster.Status.Nodes.Total > 0 {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PayloadSchedulable, "", "no ready nodes - payloads cannot be scheduled",
			))
			return lifecycle.Continue(), nil
		}

		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.PayloadSchedulable, "", "",
		))
		return lifecycle.Continue(), nil
	}
}
