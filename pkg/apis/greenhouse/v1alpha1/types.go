// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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

package v1alpha1

import (
	"fmt"
)

// HelmChartReference references a Helm Chart in a chart repository.
type HelmChartReference struct {
	// Name of the HelmChart chart.
	Name string `json:"name"`
	// Repository of the HelmChart chart.
	Repository string `json:"repository"`
	// Version of the HelmChart chart.
	Version string `json:"version"`
}

// String returns the printable HelmChartReference.
func (h *HelmChartReference) String() string {
	return fmt.Sprintf("%s/%s:%s", h.Repository, h.Name, h.Version)
}

// ValueFromSource is a valid source for a value.
type ValueFromSource struct {
	// Secret references the secret containing the value.
	Secret *SecretKeyReference `json:"secret,omitempty"`
}

// SecretKeyReference specifies the secret and key containing the value.
type SecretKeyReference struct {
	// Name of the secret in the same namespace.
	Name string `json:"name"`
	// Key in the secret to select the value from.
	Key string `json:"key"`
}

// UIApplicationReference references the UI plugin to use.
type UIApplicationReference struct {
	// URL specifies the url to a built javascript asset.
	// By default, assets are loaded from the Juno asset server using the provided name and version.
	URL string `json:"url,omitempty"`

	// Name of the UI application.
	Name string `json:"name"`

	// Version of the frontend application.
	Version string `json:"version"`
}
