// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package reconcile

import (
	"context"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// hasCleanupFinalizer - returns true in case the cleanup finalizer exists in runtimeObject
func hasCleanupFinalizer(runtimeObject RuntimeObject) bool {
	return slices.Contains(runtimeObject.GetFinalizers(), greenhouseapis.CommonCleanupFinalizer)
}

// addFinalizer - Adds greenhouse cleanup finalizer to resource
func addFinalizer(ctx context.Context, kubeClient client.Client, runtimeObject RuntimeObject) (ctrl.Result, error) {
	if controllerutil.AddFinalizer(runtimeObject, greenhouseapis.CommonCleanupFinalizer) {
		return ctrl.Result{}, kubeClient.Update(ctx, runtimeObject)
	}
	return ctrl.Result{}, nil
}

// removeFinalizer - Removes greenhouse cleanup finalizer from resource
func removeFinalizer(ctx context.Context, kubeClient client.Client, runtimeObject RuntimeObject) (ctrl.Result, error) {
	if controllerutil.RemoveFinalizer(runtimeObject, greenhouseapis.CommonCleanupFinalizer) {
		return ctrl.Result{}, kubeClient.Update(ctx, runtimeObject)
	}
	return ctrl.Result{}, nil
}

func isResourceDeleted(runtimeObject RuntimeObject) bool {
	status := runtimeObject.GetConditions()
	if len(status.Conditions) == 0 {
		return false
	}
	deleteCondition := status.GetConditionByType(greenhousev1alpha1.DeleteCondition)
	if deleteCondition == nil {
		return false
	}
	return deleteCondition.IsTrue()
}

func setCondition(condition greenhousev1alpha1.Condition, runtimeObject RuntimeObject) {
	status := runtimeObject.GetConditions()
	status.SetConditions(condition)
}

// patchStatus - patches the status of the resource with the new status and returns the reconcile error
func patchStatus(ctx context.Context, new RuntimeObject, kubeClient client.Client, reconcileError error) error {
	old, err := getOriginalResourceFromContext(ctx)
	if err != nil {
		return err
	}
	err = kubeClient.Status().Patch(ctx, new, client.MergeFrom(old))
	if err != nil {
		return err
	}
	return reconcileError
}
