// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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

// UIApplicationReference references the UI pluginDefinition to use.
type UIApplicationReference struct {
	// URL specifies the url to a built javascript asset.
	// By default, assets are loaded from the Juno asset server using the provided name and version.
	URL string `json:"url,omitempty"`

	// Name of the UI application.
	Name string `json:"name"`

	// Version of the frontend application.
	Version string `json:"version"`
}
