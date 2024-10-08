// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

type ReconcileResult string

const (
	CreatedReason          greenhousev1alpha1.ConditionReason = "Created"
	PendingCreationReason  greenhousev1alpha1.ConditionReason = "PendingCreation"
	FailingCreationReason  greenhousev1alpha1.ConditionReason = "FailingCreation"
	PendingDeletionReason  greenhousev1alpha1.ConditionReason = "PendingDeletion"
	FailingDeletionReason  greenhousev1alpha1.ConditionReason = "FailingDeletion"
	DeletedReason          greenhousev1alpha1.ConditionReason = "Deleted"
	CommonCleanupFinalizer                                    = "greenhouse.sap/cleanup"

	// Success should be returned in case the operator reached its target state
	Success ReconcileResult = "Success"
	// Failed should be returned in case the operator wasn't able to reach its target state and without external changes it's unlikely that this will succeed in the next try
	Failed ReconcileResult = "Failed"
	// Pending should be returned in case the operator is still trying to reach the target state (Requeue, waiting for remote resource to be cleaned up, etc.)
	Pending ReconcileResult = "Pending"
)

// Conditioner is a function that can be used to set the status conditions of the object at a later point in the reconciliation process
// Provided by the caller of the Reconcile function
type Conditioner func(context.Context, RuntimeObject)

// RuntimeObject is an interface that generalizes the CR object that is being reconciled
type RuntimeObject interface {
	runtime.Object
	v1.Object
	// GetConditions returns the status conditions of the object (must be implemented in respective types)
	GetConditions() greenhousev1alpha1.StatusConditions
	// SetCondition sets the status conditions of the object (must be implemented in respective types)
	SetCondition(greenhousev1alpha1.Condition)
}

// Reconciler is the interface that wraps the basic EnsureCreated and EnsureDeleted methods that a controller should implement
type Reconciler interface {
	EnsureCreated(context.Context, RuntimeObject) (ctrl.Result, ReconcileResult, error)
	EnsureDeleted(context.Context, RuntimeObject) (ctrl.Result, ReconcileResult, error)
	GetEventRecorder() record.EventRecorder
}

// Reconcile - is a generic function that is used to reconcile the state of a resource
// It standardizes the reconciliation loop and provides a common way to set finalizers, remove finalizers, and update the status of the resource
// It splits the reconciliation into two phases: EnsureCreated and EnsureDeleted to keep the create / update and delete logic in controllers segregated
func Reconcile(ctx context.Context, kubeClient client.Client, namespacedName types.NamespacedName, runtimeObject RuntimeObject, reconciler Reconciler, statusFunc Conditioner) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	if err := kubeClient.Get(ctx, namespacedName, runtimeObject); err != nil {
		if apiErrors.IsNotFound(err) {
			// object was deleted in the meantime
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to load resource")
		return ctrl.Result{}, err
	}
	// store the original object in the context
	ctx = createContextFromRuntimeObject(ctx, runtimeObject, reconciler.GetEventRecorder())

	shouldBeDeleted := runtimeObject.GetDeletionTimestamp() != nil

	// check whether finalizer is set
	if !shouldBeDeleted && !hasCleanupFinalizer(runtimeObject) {
		logger.Info("add finalizer")
		return addFinalizer(ctx, kubeClient, runtimeObject)
	}

	var (
		result ctrl.Result
		err    error
	)
	if shouldBeDeleted && hasCleanupFinalizer(runtimeObject) {
		// check if the resource is already deleted (a control state to decide whether to remove finalizer)
		// at this point the remote resource is already cleaned up so garbage collection can be done
		if isResourceDeleted(runtimeObject) {
			logger.Info("remove finalizers")
			return removeFinalizer(ctx, kubeClient, runtimeObject)
		}
		// if the resource is not deleted yet, we need to ensure it is deleted
		logger.Info("ensure deleted")
		result, err = ensureDeleted(ctx, reconciler, runtimeObject)
	} else {
		// if it is not in deletion phase then we ensure it is in desired created state
		result, err = ensureCreated(ctx, reconciler, statusFunc, runtimeObject)
	}

	// patch the final status of the resource to end the reconciliation loop
	return result, patchStatus(ctx, runtimeObject, kubeClient, err)
}

// ensureCreated - invokes the controller's EnsureCreated method and invokes the statusFunc to update the status of the resource
func ensureCreated(ctx context.Context, reconciler Reconciler, statusFunc Conditioner, runtimeObject RuntimeObject) (ctrl.Result, error) {
	result, reconcileResult, err := reconciler.EnsureCreated(ctx, runtimeObject)
	if statusFunc != nil {
		statusFunc(ctx, runtimeObject)
	} else {
		// if no statusFunc is provided, we can use defaults
		setupCreateState(runtimeObject, reconcileResult, err)
	}
	return result, err
}

// ensureDeleted - invokes the controller's EnsureDeleted method and sets the status of the resource to deleted
func ensureDeleted(ctx context.Context, reconciler Reconciler, runtimeObject RuntimeObject) (ctrl.Result, error) {
	setupDeleteState(runtimeObject, Pending, nil)
	result, reconcileResult, err := reconciler.EnsureDeleted(ctx, runtimeObject)
	setupDeleteState(runtimeObject, reconcileResult, err)
	return result, err
}
