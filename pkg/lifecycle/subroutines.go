// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Result is the return value of a SubRoutine. It embeds controller-runtime's
// reconcile.Result for RequeueAfter and carries two flags that short-circuit
// the chain:
//
//   - Pending: stop the chain, map to ReconcileResult Pending (requeue).
//   - Break:   stop the chain, map to ReconcileResult Failed (degraded status).
//
// A zero Result with nil error means "continue to the next subroutine".
type Result struct {
	reconcile.Result

	Pending bool
	Break   bool
}

// SubRoutine is a single step in a reconciliation chain. It closes over the
// resource being reconciled and any per-reconcile state held on the phase
// struct; the only argument is the context, so chains can be assembled as
// plain slices.
type SubRoutine func(context.Context) (Result, error)

// ExecuteSubRoutine runs subroutines in order until one signals exit or the
// chain completes. Mapping into ReconcileResult:
//
//   - Break or err != nil  -> Failed, stop.
//   - Pending or RequeueAfter > 0 -> Pending, stop (caller requeues).
//   - All routines return Continue -> Success.
func ExecuteSubRoutine(ctx context.Context, routines []SubRoutine) (ctrl.Result, ReconcileResult, error) {
	var (
		result Result
		err    error
	)

	for _, r := range routines {
		result, err = r(ctx)
		if result.Break {
			return result.Result, Failed, err
		}
		if err != nil {
			return result.Result, Failed, err
		}
		if result.Pending {
			return result.Result, Pending, nil
		}
		if result.RequeueAfter > 0 {
			return result.Result, Pending, nil
		}
	}

	return result.Result, Success, nil
}

// Continue signals "move to the next subroutine".
func Continue() Result {
	return Result{}
}

// Break signals "stop the chain, mark Failed". The accompanying error from the
// subroutine is what surfaces on status.
func Break() Result {
	return Result{Break: true}
}

// Requeue signals a 5-second requeue. Use for short-poll loops on remote state
// that is expected to settle quickly.
func Requeue() Result {
	return Result{Result: ctrl.Result{RequeueAfter: 5 * time.Second}}
}

// RequeueAfter signals a requeue with a caller-chosen delay.
func RequeueAfter(d time.Duration) Result {
	return Result{Result: ctrl.Result{RequeueAfter: d}}
}
