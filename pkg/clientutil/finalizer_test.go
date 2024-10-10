package clientutil

import (
	"context"
	"errors"
	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
)

const (
	CommonCleanupFinalizer = "greenhouse.sap/cleanup"
)

type errorClient struct {
	client.Client
}

func (e *errorClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	return errors.New("simulated patch error")
}

func objectWithFinalizer(finalizers ...string) *fixtures.Dummy {
	var setFinalizers []string
	for _, finalizer := range finalizers {
		setFinalizers = append(setFinalizers, finalizer)
	}
	return &fixtures.Dummy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-object",
			Namespace:  "test-namespace",
			Finalizers: setFinalizers,
		},
	}
}

func objectWithoutFinalizer() *fixtures.Dummy {
	return &fixtures.Dummy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-object-no-finalizer",
			Namespace: "test-namespace",
		},
	}
}

func Test_HasFinalizer(t *testing.T) {
	type args struct {
		runtimeObject client.Object
		finalizer     string
	}

	objWithFinalizer := objectWithFinalizer(CommonCleanupFinalizer, "some.other.finalizer")

	testCases := []struct {
		name string
		args args
		want bool
	}{
		{name: "it should have the right finalizer", want: true, args: args{runtimeObject: objWithFinalizer, finalizer: CommonCleanupFinalizer}},
		{name: "it should not have the right finalizer", want: false, args: args{runtimeObject: objectWithoutFinalizer(), finalizer: CommonCleanupFinalizer}},
		{name: "it should not have the right finalizer", want: false, args: args{runtimeObject: objectWithFinalizer("some.another.finalizer"), finalizer: CommonCleanupFinalizer}},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if got := HasFinalizer(test.args.runtimeObject, test.args.finalizer); got != test.want {
				t.Errorf("HasFinalizer() = %v, want %v", got, test.want)
			}
		})
	}
}

func Test_EnsureFinalizer(t *testing.T) {
	err := fixtures.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	type args struct {
		ctx       context.Context
		c         client.Client
		o         client.Object
		finalizer string
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "it should add finalizer successfully",
			args: args{
				ctx:       context.Background(),
				c:         fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objectWithoutFinalizer()).Build(),
				o:         objectWithoutFinalizer(),
				finalizer: "some.other.finalizer",
			},
			wantErr: false,
		},
		{
			name: "it should error out while adding finalizer",
			args: args{
				ctx:       context.Background(),
				c:         &errorClient{Client: nil},
				o:         objectWithoutFinalizer(),
				finalizer: "some.another.finalizer",
			},
			wantErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err = EnsureFinalizer(test.args.ctx, test.args.c, test.args.o, test.args.finalizer)
			if (err != nil) != test.wantErr {
				t.Errorf("EnsureFinalizer() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && !controllerutil.ContainsFinalizer(test.args.o, test.args.finalizer) {
				t.Errorf("EnsureFinalizer() finalizer not added")
			}
		})
	}
}

func Test_RemoveFinalizer(t *testing.T) {
	err := fixtures.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	type args struct {
		ctx       context.Context
		c         client.Client
		o         client.Object
		finalizer string
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "it should remove finalizer successfully",
			args: args{
				ctx:       context.Background(),
				c:         fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objectWithFinalizer(CommonCleanupFinalizer)).Build(),
				o:         objectWithFinalizer(CommonCleanupFinalizer),
				finalizer: CommonCleanupFinalizer,
			},
			wantErr: false,
		},
		{
			name: "it should error out while removing finalizer",
			args: args{
				ctx:       context.Background(),
				c:         &errorClient{Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objectWithFinalizer(CommonCleanupFinalizer)).Build()},
				o:         objectWithFinalizer(CommonCleanupFinalizer),
				finalizer: CommonCleanupFinalizer,
			},
			wantErr: true,
		},
		{
			name: "it should not error out if finalizer is not present",
			args: args{
				ctx:       context.Background(),
				c:         nil,
				o:         objectWithoutFinalizer(),
				finalizer: CommonCleanupFinalizer,
			},
			wantErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err = RemoveFinalizer(test.args.ctx, test.args.c, test.args.o, test.args.finalizer)
			if (err != nil) != test.wantErr {
				t.Errorf("RemoveFinalizer() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && controllerutil.ContainsFinalizer(test.args.o, test.args.finalizer) {
				t.Errorf("RemoveFinalizer() finalizer not removed")
			}
		})
	}
}
