package lifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"github.com/cloudoperators/greenhouse/pkg/mocks"
)

var createdCondition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.ReadyCondition, lifecycle.CreatedReason, "resource is successfully created")
var pendingCreationCondition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ReadyCondition, lifecycle.PendingCreationReason, "resource is pending creation")
var failingCreationCondition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.ReadyCondition, lifecycle.FailingCreationReason, "resource creation failed")
var deletedCondition = greenhousev1alpha1.TrueCondition(greenhousev1alpha1.DeleteCondition, lifecycle.DeletedReason, "resource is successfully deleted")
var pendingDeletionCondition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, lifecycle.PendingDeletionReason, "resource is pending deletion")
var failingDeletionCondition = greenhousev1alpha1.FalseCondition(greenhousev1alpha1.DeleteCondition, lifecycle.FailingDeletionReason, "resource deletion failed")

func TestReconcile(t *testing.T) {
	deletionTime := metav1.NewTime(time.Now())

	statusWriter := &mocks.MockSubResourceWriter{}
	statusWriter.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockedClient := &mocks.MockClient{}
	mockedClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockedClient.On("Patch", mock.Anything, mock.Anything).Return(nil)
	mockedClient.On("Status").Return(statusWriter)

	ctx := context.Background()

	namespacedName := types.NamespacedName{Name: "DummyResource", Namespace: "Dummy"}

	type args struct {
		reconcileResult lifecycle.ReconcileResult
		deletionTime    *metav1.Time
		setupState      greenhousev1alpha1.Condition
		reconcileError  error
	}

	ensureCreated := "EnsureCreated"
	ensureDeleted := "EnsureDeleted"

	tests := []struct {
		name           string
		args           args
		wantMethod     string
		wantSetupState greenhousev1alpha1.Condition
	}{
		{name: "it should reach CREATED state", wantMethod: ensureCreated, wantSetupState: createdCondition, args: args{setupState: greenhousev1alpha1.Condition{}, reconcileResult: lifecycle.Success, deletionTime: nil}},
		{name: "it should be in PENDING_CREATION state", wantMethod: ensureCreated, wantSetupState: pendingCreationCondition, args: args{setupState: greenhousev1alpha1.Condition{}, reconcileResult: lifecycle.Pending, deletionTime: nil}},
		{name: "it should reach FAILING_CREATION state", wantMethod: ensureCreated, wantSetupState: failingCreationCondition, args: args{setupState: greenhousev1alpha1.Condition{}, reconcileResult: lifecycle.Failed, deletionTime: nil, reconcileError: errors.New("")}},
		{name: "it should stay in CREATED state", wantMethod: ensureCreated, wantSetupState: createdCondition, args: args{setupState: createdCondition, reconcileResult: lifecycle.Success, deletionTime: nil}},
		{name: "it should reach DELETED state", wantMethod: ensureDeleted, wantSetupState: deletedCondition, args: args{setupState: createdCondition, reconcileResult: lifecycle.Success, deletionTime: &deletionTime}},
		{name: "it should reach PENDING_DELETION state", wantMethod: ensureDeleted, wantSetupState: pendingDeletionCondition, args: args{setupState: createdCondition, reconcileResult: lifecycle.Pending, deletionTime: &deletionTime}},
		{name: "it should reach FAILING_DELETION state", wantMethod: ensureDeleted, wantSetupState: failingDeletionCondition, args: args{setupState: createdCondition, reconcileResult: lifecycle.Failed, deletionTime: &deletionTime, reconcileError: errors.New("")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceForTest := &fixtures.Dummy{
				Spec:     fixtures.DummySpec{},
				Status:   fixtures.DummyStatus{StatusConditions: greenhousev1alpha1.StatusConditions{Conditions: []greenhousev1alpha1.Condition{tt.args.setupState}}},
				TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "DummyResource",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now()),
					DeletionTimestamp: tt.args.deletionTime,
					Finalizers:        []string{lifecycle.CommonCleanupFinalizer},
				},
			}

			mockedReconciler := &mocks.MockReconciler{}
			wantNotCalled := ensureCreated
			if tt.wantMethod == ensureCreated {
				wantNotCalled = ensureDeleted
				mockedReconciler.On(ensureCreated, mock.Anything, mock.Anything).Return(ctrl.Result{}, tt.args.reconcileResult, tt.args.reconcileError)
				mockedReconciler.On(ensureDeleted, mock.Anything, mock.Anything).Return(ctrl.Result{}, nil, tt.args.reconcileError)
			} else {
				mockedReconciler.On(ensureCreated, mock.Anything, mock.Anything).Return(ctrl.Result{}, nil, tt.args.reconcileError)
				mockedReconciler.On(ensureDeleted, mock.Anything, mock.Anything).Return(ctrl.Result{}, tt.args.reconcileResult, tt.args.reconcileError)
			}

			_, err := lifecycle.Reconcile(ctx, mockedClient, namespacedName, resourceForTest, mockedReconciler, nil)

			require.Equal(t, tt.args.reconcileError, err)
			mockedReconciler.AssertCalled(t, tt.wantMethod, mock.Anything, mock.Anything)
			mockedReconciler.AssertNotCalled(t, wantNotCalled, mock.Anything, mock.Anything)
			statusWriter.AssertCalled(t, "Patch", mock.Anything, mock.Anything, mock.Anything)
			expectedState := resourceForTest.Status.GetConditionByType(tt.wantSetupState.Type)
			require.Equal(t, tt.wantSetupState.Reason, expectedState.Reason)
		})
	}
}
