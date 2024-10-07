// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package reconcile

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RuntimeObject interface {
	runtime.Object
	v1.Object
	GetConditions() greenhousev1alpha1.StatusConditions
}

type Reconciler interface {
	EnsureCreated(context.Context, RuntimeObject) (ctrl.Result, error)
	EnsureDeleted(context.Context, RuntimeObject) (ctrl.Result, error)
	GetEventRecorder() record.EventRecorder
}

func Reconcile(ctx context.Context, kubeClient client.Client, namespacedName types.NamespacedName, runtimeObject RuntimeObject, reconciler Reconciler) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	if err := kubeClient.Get(ctx, namespacedName, runtimeObject); err != nil {
		if apiErrors.IsNotFound(err) {
			// object was deleted in the meantime
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to load resource")
		return ctrl.Result{}, err
	}
	//https://github.com/kubernetes/kubernetes/issues/3030
	if runtimeObject.GetObjectKind().GroupVersionKind() == (schema.GroupVersionKind{}) {
		err := addTypeInformationToObject(runtimeObject)
		if err != nil {
			logger.Error(err, "GVK is missing in runtime object")
			return ctrl.Result{}, err
		}
	}

	ctx = createContextFromRuntimeObject(ctx, runtimeObject, reconciler.GetEventRecorder())

	shouldBeDeleted := runtimeObject.GetDeletionTimestamp() != nil

	// check whether finalizer is set
	if !shouldBeDeleted && !hasCleanupFinalizer(runtimeObject) {
		logger.Info("add finalizers")
		return addFinalizer(ctx, kubeClient, runtimeObject)
	}

	var (
		result ctrl.Result
		err    error
	)
	if shouldBeDeleted && hasCleanupFinalizer(runtimeObject) {
		if isResourceDeleted(runtimeObject) {
			// remove CleanupFinalizer if SetupState is Deleted. Once all finalizers have been
			// removed, the object will be deleted.
			logger.Info("remove finalizers")
			return removeFinalizer(ctx, kubeClient, runtimeObject)
		}

		logger.Info("ensure deleted")
		result, err = ensureDeleted(ctx, reconciler, runtimeObject)
	} else {
		result, err = ensureCreated(ctx, reconciler, runtimeObject)
	}

	return result, patchStatus(ctx, runtimeObject, kubeClient, err)
}

func ensureCreated(ctx context.Context, reconciler Reconciler, runtimeObject RuntimeObject) (ctrl.Result, error) {
	result, err := reconciler.EnsureCreated(ctx, runtimeObject)
	if err != nil {
		createCondition := greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ReadyCondition, "FailingCreation", err.Error())
		setCondition(createCondition, runtimeObject)
	}
	return result, err
}

func ensureDeleted(ctx context.Context, reconciler Reconciler, runtimeObject RuntimeObject) (ctrl.Result, error) {
	deleteCondition := greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, "PendingDeletion", "Resource is being deleted")
	setCondition(deleteCondition, runtimeObject)
	result, err := reconciler.EnsureDeleted(ctx, runtimeObject)
	if err == nil && result.IsZero() {
		deleteCondition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.DeleteCondition, "Deleted", "Resource is deleted")
		setCondition(deleteCondition, runtimeObject)
	}
	return result, err
}

// see https://github.com/kubernetes/kubernetes/issues/3030
func addTypeInformationToObject(obj runtime.Object) error {
	groupVersionKinds, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range groupVersionKinds {
		if gvk.Kind == "" || gvk.Version == "" || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}
	return nil
}
