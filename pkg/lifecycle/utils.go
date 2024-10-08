// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"

	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

// hasCleanupFinalizer - returns true in case the cleanup finalizer exists in runtimeObject
func hasCleanupFinalizer(runtimeObject RuntimeObject) bool {
	return slices.Contains(runtimeObject.GetFinalizers(), CommonCleanupFinalizer)
}

// addFinalizer - Adds greenhouse cleanup finalizer to resource
func addFinalizer(ctx context.Context, kubeClient client.Client, runtimeObject RuntimeObject) (ctrl.Result, error) {
	if controllerutil.AddFinalizer(runtimeObject, CommonCleanupFinalizer) {
		return ctrl.Result{}, kubeClient.Update(ctx, runtimeObject)
	}
	return ctrl.Result{}, nil
}

// removeFinalizer - Removes greenhouse cleanup finalizer from resource
func removeFinalizer(ctx context.Context, kubeClient client.Client, runtimeObject RuntimeObject) (ctrl.Result, error) {
	if controllerutil.RemoveFinalizer(runtimeObject, CommonCleanupFinalizer) {
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

// patchStatus - patches the status of the resource with the new status and returns the reconcile error
// TODO: implement event recorder to fire ready condition change events
func patchStatus(ctx context.Context, newObject RuntimeObject, kubeClient client.Client, reconcileError error) error {
	oldObject, err := getOriginalResourceFromContext(ctx)
	if err != nil {
		return err
	}
	err = kubeClient.Status().Patch(ctx, newObject, client.MergeFrom(oldObject))
	if err != nil {
		return err
	}
	return reconcileError
}

// setupDeleteState - converts the reconcile result to a condition and sets it in the runtimeObject for deletion phase
func setupDeleteState(runtimeObject RuntimeObject, reconcileResult ReconcileResult, err error) {
	var condition greenhousev1alpha1.Condition
	switch reconcileResult {
	case Success:
		condition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.DeleteCondition, DeletedReason, "resource is successfully deleted")
	case Failed:
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		condition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, FailingDeletionReason, "resource deletion failed"+msg)
	default:
		condition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, PendingDeletionReason, "resource deletion is pending")
	}
	runtimeObject.SetCondition(condition)
}

// setupCreateState - if statusFunc is not passed to reconciler then the default status conditions are set in runtimeObject
func setupCreateState(runtimeObject RuntimeObject, reconcileResult ReconcileResult, err error) {
	var condition greenhousev1alpha1.Condition
	switch reconcileResult {
	case Success:
		condition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ReadyCondition, CreatedReason, "resource is successfully created")
	case Failed:
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		condition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ReadyCondition, FailingCreationReason, "resource creation failed"+msg)
	default:
		condition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.ReadyCondition, PendingCreationReason, "resource creation is pending")
	}
	runtimeObject.SetCondition(condition)
}
