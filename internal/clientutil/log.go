// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func LogIntoContextFromRequest(ctx context.Context, req ctrl.Request) context.Context {
	return LogIntoContext(ctx, "key", req.String())
}

func LogIntoContext(ctx context.Context, keysAndValues ...any) context.Context {
	return log.IntoContext(ctx, log.FromContext(ctx, keysAndValues...))
}
