// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"fmt"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HelmChartTestReconciler struct {
	client.Client
	recorder        record.EventRecorder
	kubeRuntimeOpts clientutil.RuntimeOptions
	kubeClientOpts  []clientutil.KubeClientOption
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugindefinitions,verbs=get;list
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins,verbs=get;list
//+kubebuilder:rbac:groups=greenhouse.sap,resources=plugins/status,verbs=get;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters;teams,verbs=get;list

// SetupWithManager sets up the controller with the Manager.
func (r *HelmChartTestReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.kubeClientOpts = []clientutil.KubeClientOption{
		clientutil.WithRuntimeOptions(r.kubeRuntimeOpts),
		clientutil.WithPersistentConfig(),
	}

	r.recorder = mgr.GetEventRecorderFor(name)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Plugin{}).
		Complete(r)
}

func (r *HelmChartTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// logic goes here
	fmt.Printf("Reconciling Plugin from HelmChartTestReconciler: %s\n", req.NamespacedName)
	var plugin greenhousev1alpha1.Plugin
	if err := r.Get(ctx, req.NamespacedName, &plugin); err != nil {
		fmt.Printf("Error getting plugin: %v", err)
	}

	if plugin.Status.HelmReleaseStatus == nil || plugin.Status.HelmReleaseStatus.Status == "unknown" {
		fmt.Printf("The plugin status is unknown or not set %v is %v\n", plugin.Name, plugin.Status.HelmReleaseStatus.Status)
		return ctrl.Result{}, nil
	}

	fmt.Printf("The plugin status of %v is %v\n", plugin.Name, plugin.Status.HelmReleaseStatus.Status)

	return ctrl.Result{}, nil
}
