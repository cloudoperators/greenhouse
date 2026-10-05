// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package changemanagement

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChangeManagement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Change Management suite")
}
