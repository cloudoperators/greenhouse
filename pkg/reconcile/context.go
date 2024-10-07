package reconcile

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

const msgEventDiscarded = "Event is discarded because no event recorder was found in context"

type reconcileRunKey struct{}

type reconcileRun struct {
	eventRecorder record.EventRecorder
	objectCopy    RuntimeObject
}

type dummyEventRecorder struct {
	logger logr.Logger
}

func (d dummyEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	d.logger.Info(msgEventDiscarded, "type", eventtype, "reason", reason, "message", message)
}

func (d dummyEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...any) {
	d.logger.Info(msgEventDiscarded, "type", eventtype, "reason", reason, "messageFmt", messageFmt, "args", args)
}

func (d dummyEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...any) {
	d.logger.Info(msgEventDiscarded, "annotations", annotations, "type", eventtype, "reason", reason, "messageFmt", messageFmt, "args", args)
}

// CreateContextFromRuntimeObject create a new context with a copy of the object attached.
func createContextFromRuntimeObject(ctx context.Context, object RuntimeObject, recorder record.EventRecorder) context.Context {
	if recorder == nil {
		recorder = dummyEventRecorder{ctrl.LoggerFrom(ctx)}
	}
	return context.WithValue(ctx, reconcileRunKey{}, &reconcileRun{
		eventRecorder: recorder,
		objectCopy:    object.DeepCopyObject().(RuntimeObject),
	})
}

func GetEventRecorderFromContext(ctx context.Context) record.EventRecorder {
	reconcileRun, err := getRunFromContext(ctx)
	if err != nil {
		return dummyEventRecorder{ctrl.LoggerFrom(ctx)}
	}

	return reconcileRun.eventRecorder
}

func getRunFromContext(ctx context.Context) (*reconcileRun, error) {
	val, ok := ctx.Value(reconcileRunKey{}).(*reconcileRun)
	if !ok {
		return nil, errors.New("could  not extract *reconcileRun from given context")
	}

	return val, nil
}

// GetOriginalResourceFromContext - returns the unmodified version of the RuntimeObject
func getOriginalResourceFromContext(ctx context.Context) (RuntimeObject, error) {
	reconcileRun, err := getRunFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// create another copy so that context can not be modified by accident
	return reconcileRun.objectCopy.DeepCopyObject().(RuntimeObject), nil
}
