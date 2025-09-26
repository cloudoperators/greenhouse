// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	"github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/fixtures"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
	"github.com/cloudoperators/greenhouse/internal/mocks"
)

func TestReconcile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconcile Suite")
}

var _ = Describe("Reconcile", func() {
	var (
		createdCondition         = greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.ReadyCondition, lifecycle.CreatedReason, "resource is successfully created")
		pendingCreationCondition = greenhousemetav1alpha1.UnknownCondition(greenhousemetav1alpha1.ReadyCondition, lifecycle.PendingCreationReason, "resource creation is pending")
		failingCreationCondition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.ReadyCondition, lifecycle.FailingCreationReason, "resource creation failed")
		deletedCondition         = greenhousemetav1alpha1.TrueCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.DeletedReason, "resource is successfully deleted")
		pendingDeletionCondition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.PendingDeletionReason, "resource deletion is pending")
		failingDeletionCondition = greenhousemetav1alpha1.FalseCondition(greenhousemetav1alpha1.DeleteCondition, lifecycle.FailingDeletionReason, "resource deletion failed: ")
	)
	var (
		mockClient      *mocks.MockClient
		mockReconciler  *mocks.MockReconciler
		statusWriter    *mocks.MockSubResourceWriter
		ctx             context.Context
		namespacedName  types.NamespacedName
		resourceForTest *fixtures.Dummy
		deletionTime    metav1.Time
	)

	BeforeEach(func() {
		statusWriter = &mocks.MockSubResourceWriter{}
		statusWriter.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient = &mocks.MockClient{}
		mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Status").Return(statusWriter)

		mockReconciler = &mocks.MockReconciler{}

		ctx = context.Background()
		namespacedName = types.NamespacedName{Name: "DummyResource", Namespace: "Dummy"}
		deletionTime = metav1.NewTime(time.Now())
	})

	type args struct {
		reconcileResult lifecycle.ReconcileResult
		deletionTime    *metav1.Time
		setupState      greenhousemetav1alpha1.Condition
		finalizers      []string
		reconcileError  error
	}

	ensureCreated := "EnsureCreated"
	ensureDeleted := "EnsureDeleted"

	DescribeTable("Reconcile",
		func(tt struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}) {
			if len(tt.args.finalizers) == 0 {
				tt.args.finalizers = []string{lifecycle.CommonCleanupFinalizer}
			}
			resourceForTest = &fixtures.Dummy{
				Spec:     fixtures.DummySpec{},
				Status:   fixtures.DummyStatus{StatusConditions: greenhousemetav1alpha1.StatusConditions{Conditions: []greenhousemetav1alpha1.Condition{tt.args.setupState}}},
				TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "DummyResource",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now()),
					DeletionTimestamp: tt.args.deletionTime,
					Finalizers:        tt.args.finalizers,
				},
			}

			wantNotCalled := ensureCreated
			if tt.wantMethod == ensureCreated {
				wantNotCalled = ensureDeleted
				mockReconciler.On(ensureCreated, mock.Anything, mock.Anything).Return(ctrl.Result{}, tt.args.reconcileResult, tt.args.reconcileError)
				mockReconciler.On(ensureDeleted, mock.Anything, mock.Anything).Return(ctrl.Result{}, nil, tt.args.reconcileError)
			} else {
				mockReconciler.On(ensureCreated, mock.Anything, mock.Anything).Return(ctrl.Result{}, nil, tt.args.reconcileError)
				mockReconciler.On(ensureDeleted, mock.Anything, mock.Anything).Return(ctrl.Result{}, tt.args.reconcileResult, tt.args.reconcileError)
			}

			_, err := lifecycle.Reconcile(ctx, mockClient, namespacedName, resourceForTest, mockReconciler, nil)

			if tt.args.reconcileError == nil {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(Equal(tt.args.reconcileError))
			}
			if !tt.verifyFinalizerRemoval {
				mockReconciler.AssertCalled(GinkgoT(), tt.wantMethod, mock.Anything, mock.Anything)
				mockReconciler.AssertNotCalled(GinkgoT(), wantNotCalled, mock.Anything, mock.Anything)
				statusWriter.AssertCalled(GinkgoT(), "Patch", mock.Anything, mock.Anything, mock.Anything)
			}
			expectedState := resourceForTest.Status.GetConditionByType(tt.wantSetupState.Type)
			// we cannot compare the whole condition because the lastTransitionTime is different
			Expect(expectedState.Type).To(Equal(tt.wantSetupState.Type))
			Expect(expectedState.Status).To(Equal(tt.wantSetupState.Status))
			Expect(expectedState.Reason).To(Equal(tt.wantSetupState.Reason))
			Expect(expectedState.Message).To(Equal(tt.wantSetupState.Message))
			if tt.verifyFinalizerRemoval {
				Expect(resourceForTest.GetFinalizers()).NotTo(ContainElement(lifecycle.CommonCleanupFinalizer))
			}
		},
		Entry("it should reach CREATED state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureCreated,
			wantSetupState:         createdCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      greenhousemetav1alpha1.Condition{},
				reconcileResult: lifecycle.Success,
				deletionTime:    nil,
			},
		}),
		Entry("it should be in PENDING_CREATION state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureCreated,
			wantSetupState:         pendingCreationCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      greenhousemetav1alpha1.Condition{},
				reconcileResult: lifecycle.Pending,
				deletionTime:    nil,
			},
		}),
		Entry("it should reach FAILING_CREATION state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureCreated,
			wantSetupState:         failingCreationCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      greenhousemetav1alpha1.Condition{},
				reconcileResult: lifecycle.Failed,
				deletionTime:    nil,
				reconcileError:  errors.New(""),
			},
		}),
		Entry("it should stay in CREATED state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureCreated,
			wantSetupState:         createdCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      createdCondition,
				reconcileResult: lifecycle.Success,
				deletionTime:    nil,
			},
		}),
		Entry("it should reach DELETED state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureDeleted,
			wantSetupState:         deletedCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      createdCondition,
				reconcileResult: lifecycle.Success,
				deletionTime:    &deletionTime,
			},
		}),
		Entry("it should reach PENDING_DELETION state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureDeleted,
			wantSetupState:         pendingDeletionCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      createdCondition,
				reconcileResult: lifecycle.Pending,
				deletionTime:    &deletionTime,
			},
		}),
		Entry("it should reach FAILING_DELETION state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureDeleted,
			wantSetupState:         failingDeletionCondition,
			verifyFinalizerRemoval: false,
			args: args{
				setupState:      createdCondition,
				reconcileResult: lifecycle.Failed,
				deletionTime:    &deletionTime,
				reconcileError:  errors.New(""),
			},
		}),
		Entry("it should not have finalizers if in DELETED state", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureDeleted,
			wantSetupState:         deletedCondition,
			verifyFinalizerRemoval: true,
			args: args{
				setupState:      deletedCondition,
				reconcileResult: lifecycle.Success,
				deletionTime:    &deletionTime,
			},
		}),
		Entry("it should not enter ensureCreated or ensureDeleted if deletionTime is set but no common finalizer", struct {
			args                   args
			wantMethod             string
			wantSetupState         greenhousemetav1alpha1.Condition
			verifyFinalizerRemoval bool
		}{
			wantMethod:             ensureDeleted,
			wantSetupState:         deletedCondition,
			verifyFinalizerRemoval: true,
			args: args{
				setupState:      deletedCondition,
				finalizers:      []string{"greenhouse.sap/unknown", lifecycle.CommonCleanupFinalizer},
				reconcileResult: lifecycle.Success,
				deletionTime:    &deletionTime,
			},
		}),
	)
})

var _ = Describe("Greenhouse Operation Annotations", func() {

	var (
		mockClient      *mocks.MockClient
		mockReconciler  *mocks.MockReconciler
		statusWriter    *mocks.MockSubResourceWriter
		ctx             context.Context
		namespacedName  types.NamespacedName
		resourceForTest *fixtures.Dummy
	)

	BeforeEach(func() {
		statusWriter = &mocks.MockSubResourceWriter{}
		statusWriter.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient = &mocks.MockClient{}
		mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Status").Return(statusWriter)

		mockReconciler = &mocks.MockReconciler{}

		ctx = context.Background()
		namespacedName = types.NamespacedName{Name: "DummyResource", Namespace: "Dummy"}
	})

	type testCase struct {
		annotations             map[string]string
		wantMethod              string
		verifyAnnotationRemoval bool
	}

	DescribeTable("Reconcile Greenhouse Operations",
		func(tt testCase) {
			resourceForTest = &fixtures.Dummy{
				TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "DummyResource",
					Namespace:         "default",
					Annotations:       tt.annotations,
					CreationTimestamp: metav1.NewTime(time.Now())},
				Spec: fixtures.DummySpec{},
			}

			_, err := lifecycle.Reconcile(ctx, mockClient, namespacedName, resourceForTest, mockReconciler, nil)
			Expect(err).ToNot(HaveOccurred())

			switch tt.verifyAnnotationRemoval {
			case true:
				Expect(resourceForTest.GetAnnotations()).NotTo(ContainElement(v1alpha1.GreenhouseOperation))
			default:
				Expect(resourceForTest.GetAnnotations()).To(Equal(tt.annotations))
			}
		},
		Entry("Greenhouse Operation 'reconcile' should be removed",
			testCase{
				verifyAnnotationRemoval: true,
				annotations:             map[string]string{v1alpha1.GreenhouseOperation: v1alpha1.GreenhouseOperationReconcile},
			}),
		Entry("Other Greenhouse Operation should not be removed",
			testCase{
				verifyAnnotationRemoval: false,
				annotations:             map[string]string{v1alpha1.GreenhouseOperation: "other"},
			}),
		Entry("No Greenhouse Operation should not be removed",
			testCase{
				verifyAnnotationRemoval: false,
				annotations:             map[string]string{},
			}),
	)
})
