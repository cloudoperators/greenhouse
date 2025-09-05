// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/test"
)

func TestInternalCommonUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal Common suite")
}

var _ = BeforeSuite(func() {
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})
