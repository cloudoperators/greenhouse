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
				greenhousev1alpha1.PayloadSchedulable, "WorkerlessCluster",
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
		if allNodesReady != nil && allNodesReady.IsFalse() {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(
				greenhousev1alpha1.PayloadSchedulable, "", allNodesReady.Message,
			))
			return lifecycle.Continue(), nil
		}

		cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(
			greenhousev1alpha1.PayloadSchedulable, "", "",
		))
		return lifecycle.Continue(), nil
	}
}
