// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGreenhouseCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Greenhouse Commands Suite")
}
