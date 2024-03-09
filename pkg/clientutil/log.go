// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientutil

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func LogIntoContextFromRequest(ctx context.Context, req ctrl.Request) context.Context {
	return LogIntoContext(ctx, "key", req.String())
}

func LogIntoContext(ctx context.Context, keysAndValues ...interface{}) context.Context {
	return log.IntoContext(ctx, log.FromContext(ctx, keysAndValues...))
}
