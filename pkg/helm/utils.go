// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"time"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// Default Greenhouse helm timeout duration in seconds for install, upgrade and rollback actions.
const helmReleaseTimeoutSeconds int = 300

// GetHelmTimeout gets a timeout duration for helm release install, upgrade and rollback actions.
// Tries to get the value from HELM_RELEASE_TIMEOUT evironment variable, otherwise gets the default value.
// Mainly used for E2E tests, because in deployment mode this should always be set to the default 5 minutes.
func GetHelmTimeout() time.Duration {
	val := clientutil.GetIntEnvWithDefault("HELM_RELEASE_TIMEOUT", helmReleaseTimeoutSeconds)
	return time.Duration(val) * time.Second
}
