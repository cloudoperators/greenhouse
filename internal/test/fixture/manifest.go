// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// Package fixture provides manifest fixtures for tests. It is kept free of dependencies so that unit tests
// can use it without pulling in the envtest setup of the test package.
package fixture

import (
	"fmt"
	"strings"
)

// PodManifest renders a Pod manifest with one container per image.
func PodManifest(images ...string) string {
	var manifest strings.Builder
	manifest.WriteString("apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n")
	for i, image := range images {
		fmt.Fprintf(&manifest, "  - name: container-%d\n    image: %s\n", i, image)
	}
	return manifest.String()
}
