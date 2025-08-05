// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

type ReconcileResult string

const (
	CreatedReason         greenhousemetav1alpha1.ConditionReason = "Created"
	PendingCreationReason greenhousemetav1alpha1.ConditionReason = "PendingCreation"
	FailingCreationReason greenhousemetav1alpha1.ConditionReason = "FailingCreation"
	// ScheduledDeletionReason is used to indicate that the resource is scheduled for deletion
	ScheduledDeletionReason greenhousemetav1alpha1.ConditionReason = "ScheduledDeletion"
	PendingDeletionReason   greenhousemetav1alpha1.ConditionReason = "PendingDeletion"
	FailingDeletionReason   greenhousemetav1alpha1.ConditionReason = "FailingDeletion"
	DeletedReason           greenhousemetav1alpha1.ConditionReason = "Deleted"
	CommonCleanupFinalizer                                         = "greenhouse.sap/cleanup"

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
	GetConditions() greenhousemetav1alpha1.StatusConditions
	// SetCondition sets the status conditions of the object (must be implemented in respective types)
	SetCondition(greenhousemetav1alpha1.Condition)
}

// Reconciler is the interface that wraps the basic EnsureCreated and EnsureDeleted methods that a controller should implement
type Reconciler interface {
	EnsureCreated(context.Context, RuntimeObject) (ctrl.Result, ReconcileResult, error)
	EnsureDeleted(context.Context, RuntimeObject) (ctrl.Result, ReconcileResult, error)
}

// Reconcile - is a generic function that is used to reconcile the state of a resource
// It standardizes the reconciliation loop and provides a common way to set finalizers, remove finalizers, and update the status of the resource
// It splits the reconciliation into two phases: EnsureCreated and EnsureDeleted to keep the create / update and delete logic in controllers segregated
func Reconcile(ctx context.Context, kubeClient client.Client, namespacedName types.NamespacedName, runtimeObject RuntimeObject, reconciler Reconciler, statusFunc Conditioner) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	if err := kubeClient.Get(ctx, namespacedName, runtimeObject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// store the original object in the context
	ctx = createContextFromRuntimeObject(ctx, runtimeObject)

	shouldBeDeleted := runtimeObject.GetDeletionTimestamp() != nil
	hasFinalizer := controllerutil.ContainsFinalizer(runtimeObject, CommonCleanupFinalizer)

	// check whether finalizer is set
	if !shouldBeDeleted && !hasFinalizer {
		return ctrl.Result{}, ensureFinalizer(ctx, kubeClient, runtimeObject, CommonCleanupFinalizer)
	}

	var (
		result ctrl.Result
		err    error
	)
	if shouldBeDeleted {
		// in case of unknown finalizer, we need to ensure that the reconcile does not enter into ensureCreated phase
		if !hasFinalizer {
			return ctrl.Result{}, nil
		}
		// check if the resource is already deleted (a control state to decide whether to remove finalizer)
		// at this point the remote resource is already cleaned up so garbage collection can be done
		if isResourceDeleted(runtimeObject) {
			err = removeFinalizer(ctx, kubeClient, runtimeObject, CommonCleanupFinalizer)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// if the resource is not deleted yet, we need to ensure it is deleted
		result, err = ensureDeleted(ctx, logger, reconciler, runtimeObject)
	} else {
		// if it is not in deletion phase then we ensure it is in desired created state
		result, err = ensureCreated(ctx, logger, reconciler, runtimeObject, statusFunc)
	}

	// patch the final status of the resource to end the reconciliation loop
	return result, patchStatus(ctx, kubeClient, runtimeObject, err)
}

// isResourceDeleted - returns true if the resource has a true Deleted condition
// This is used to determine if the resource is in deletion phase has finished its cleanup
func isResourceDeleted(runtimeObject RuntimeObject) bool {
	status := runtimeObject.GetConditions()
	deleteCondition := status.GetConditionByType(greenhousemetav1alpha1.DeleteCondition)
	if deleteCondition == nil {
		return false
	}
	return deleteCondition.IsTrue() && deleteCondition.Reason == DeletedReason
}

// ensureCreated - invokes the controller's EnsureCreated method and invokes the statusFunc to update the status of the resource
func ensureCreated(ctx context.Context, logger logr.Logger, reconciler Reconciler, runtimeObject RuntimeObject, statusFunc Conditioner) (ctrl.Result, error) {
	logger.Info("ensure created")
	result, reconcileResult, err := reconciler.EnsureCreated(ctx, runtimeObject)
	if statusFunc != nil {
		statusFunc(ctx, runtimeObject)
	} else {
		setupCreateState(runtimeObject, reconcileResult, err)
	}
	return result, err
}

// ensureDeleted - invokes the controller's EnsureDeleted method and sets the status of the resource to deleted
func ensureDeleted(ctx context.Context, logger logr.Logger, reconciler Reconciler, runtimeObject RuntimeObject) (ctrl.Result, error) {
	logger.Info("ensure deleted")
	setupDeleteState(runtimeObject, Pending, nil)
	result, reconcileResult, err := reconciler.EnsureDeleted(ctx, runtimeObject)
	setupDeleteState(runtimeObject, reconcileResult, err)
	return result, err
}

// setupDeleteState - converts the reconcile result to a condition and sets it in the runtimeObject for deletion phase
func setupDeleteState(runtimeObject RuntimeObject, reconcileResult ReconcileResult, err error) {
	var condition greenhousemetav1alpha1.Condition
	switch reconcileResult {
	case Success:
		condition = greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.DeleteCondition, DeletedReason, "resource is successfully deleted")
	case Failed:
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		condition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, FailingDeletionReason, "resource deletion failed: "+msg)
	default:
		condition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, PendingDeletionReason, "resource deletion is pending")
	}
	runtimeObject.SetCondition(condition)
}

// setupCreateState - if statusFunc is not passed to reconciler then the default status conditions are set in runtimeObject
func setupCreateState(runtimeObject RuntimeObject, reconcileResult ReconcileResult, err error) {
	var condition greenhousemetav1alpha1.Condition
	switch reconcileResult {
	case Success:
		condition = greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ReadyCondition, CreatedReason, "resource is successfully created")
	case Failed:
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		condition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, FailingCreationReason, "resource creation failed"+msg)
	default:
		condition = greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, PendingCreationReason, "resource creation is pending")
	}
	runtimeObject.SetCondition(condition)
}

// patchStatus - patches the status of the resource with the new status and returns the reconcile error
func patchStatus(ctx context.Context, kubeClient client.Client, newObject RuntimeObject, reconcileError error) error {
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

// ensureFinalizer - ensures a finalizer is present on the object. Returns an error on failure.
func ensureFinalizer(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if controllerutil.AddFinalizer(o, finalizer) {
		return c.Update(ctx, o)
	}
	return nil
}

// removeFinalizer - removes a finalizer from an object. Returns an error on failure.
func removeFinalizer(ctx context.Context, c client.Client, o client.Object, finalizer string) error {
	if controllerutil.RemoveFinalizer(o, finalizer) {
		return c.Update(ctx, o)
	}
	return nil
}

type Result struct {
	reconcile.Result
	Exit  bool
	Break bool
}

type ReconcileRoutine func(context.Context) (Result, error)

func ExecuteReconcileRoutine(ctx context.Context, routines []ReconcileRoutine) (ctrl.Result, ReconcileResult, error) {
	var result Result
	var err error

	for _, operation := range routines {
		result, err = operation(ctx)
		if result.Exit {
			return ctrl.Result{}, Success, nil
		}
		if result.Break {
			return ctrl.Result{}, Failed, err
		}
		if result.RequeueAfter > 0 {
			return result.Result, Pending, err
		}
		if err != nil {
			return result.Result, Failed, err
		}
	}

	return result.Result, Success, nil
}

// Requeue - returns a Result that indicates the controller should requeue the request after a short delay of 5 seconds
func Requeue() Result {
	return Result{Result: ctrl.Result{RequeueAfter: 5 * time.Second}}
}

// RequeueAfter - returns a Result that indicates the controller should requeue the request after the specified duration
func RequeueAfter(duration time.Duration) Result {
	return Result{Result: ctrl.Result{RequeueAfter: duration}}
}

// Exit - returns a Result that indicates the controller should exit the reconciliation loop with a success status
// This is typically used when there is no further reconciliation routine needs to be executed
// early exit strategy
func Exit() Result {
	return Result{Exit: true}
}

// Continue - returns a Result that indicates the controller should continue with the next reconciliation routine
func Continue() Result {
	return Result{}
}

// Break - returns a Result that indicates the controller should break out of the reconciliation routine
// This should be used when an error occurs
func Break() Result {
	return Result{Break: true}
}

func InitConditions(runtimeObject RuntimeObject, conditionTypes []greenhousemetav1alpha1.ConditionType) {
	statusConditions := runtimeObject.GetConditions()
	for _, t := range conditionTypes {
		if statusConditions.GetConditionByType(t) == nil {
			runtimeObject.SetCondition(greenhousemetav1alpha1.UnknownCondition(t, "", ""))
		}
	}
}
