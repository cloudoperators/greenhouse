// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func TestEnsureNodesReady(t *testing.T) {
	tests := []struct {
		name          string
		mode          greenhousev1alpha1.ClusterMode
		wantCondition bool
		wantNodes     bool
	}{
		{
			name:          "workerless removes AllNodesReady condition and nodes",
			mode:          greenhousev1alpha1.ClusterModeWorkerless,
			wantCondition: false,
			wantNodes:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &greenhousev1alpha1.Cluster{}
			cluster.Spec.Mode = tc.mode
			cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", ""))
			cluster.Status.Nodes = &greenhousev1alpha1.Nodes{Total: 3}

			p := &Phase{}
			if _, err := p.ensureNodesReady(cluster)(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			hasCond := cluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady) != nil
			if hasCond != tc.wantCondition {
				t.Errorf("AllNodesReady condition present=%v, want %v", hasCond, tc.wantCondition)
			}
			hasNodes := cluster.Status.Nodes != nil
			if hasNodes != tc.wantNodes {
				t.Errorf("Status.Nodes present=%v, want %v", hasNodes, tc.wantNodes)
			}
		})
	}
}

func TestEnsureWorkloadSchedulable(t *testing.T) {
	tests := []struct {
		name          string
		mode          greenhousev1alpha1.ClusterMode
		preconditions []greenhousemetav1alpha1.Condition
		wantStatus    metav1.ConditionStatus
		wantReason    greenhousemetav1alpha1.ConditionReason
	}{
		{
			name:       "workerless sets PayloadSchedulable=False with WorkerlessCluster reason",
			mode:       greenhousev1alpha1.ClusterModeWorkerless,
			wantStatus: metav1.ConditionFalse,
			wantReason: greenhousev1alpha1.WorkerlessClusterReason,
		},
		{
			name:       "workerless ignores missing KubeConfigValid and AllNodesReady",
			mode:       greenhousev1alpha1.ClusterModeWorkerless,
			wantStatus: metav1.ConditionFalse,
			wantReason: greenhousev1alpha1.WorkerlessClusterReason,
		},
		{
			name:       "default with all conditions passing sets PayloadSchedulable=True",
			mode:       greenhousev1alpha1.ClusterModeDefault,
			wantStatus: metav1.ConditionTrue,
		},
		{
			name: "default with KubeConfigValid=False sets PayloadSchedulable=False",
			mode: greenhousev1alpha1.ClusterModeDefault,
			preconditions: []greenhousemetav1alpha1.Condition{
				greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", "cert expired"),
			},
			wantStatus: metav1.ConditionFalse,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &greenhousev1alpha1.Cluster{}
			cluster.Spec.Mode = tc.mode
			for _, cond := range tc.preconditions {
				cluster.SetCondition(cond)
			}

			p := &Phase{}
			if _, err := p.ensureWorkloadSchedulable(cluster)(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			if cond == nil {
				t.Fatal("expected PayloadSchedulable condition to be set")
			}
			if cond.Status != tc.wantStatus {
				t.Errorf("PayloadSchedulable status=%v, want %v", cond.Status, tc.wantStatus)
			}
			if tc.wantReason != "" && cond.Reason != tc.wantReason {
				t.Errorf("PayloadSchedulable reason=%v, want %v", cond.Reason, tc.wantReason)
			}
		})
	}
}
