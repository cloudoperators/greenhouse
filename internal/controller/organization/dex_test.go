// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/controller/organization"
	dexstore "github.com/cloudoperators/greenhouse/internal/dex"
)

var _ = Describe("incrementResourceVersion", func() {
	DescribeTable("should correctly increment the resource version for postgres storage",
		func(input, expected string) {
			Expect(organization.ExportIncrementResourceVersion(dexstore.Postgres, input)).To(Equal(expected))
		},
		Entry("empty string defaults to 1", "", "1"),
		Entry("zero increments to 1", "0", "1"),
		Entry("1 increments to 2", "1", "2"),
		Entry("42 increments to 43", "42", "43"),
		Entry("non-numeric defaults to 1", "abc", "1"),
	)
	DescribeTable("should return the same resource version for non-postgres storage",
		func(input, expected string) {
			Expect(organization.ExportIncrementResourceVersion(dexstore.K8s, input)).To(Equal(expected))
		},
		Entry("empty string remains empty", "", ""),
		Entry("numeric string remains unchanged", "42", "42"),
		Entry("non-numeric string remains unchanged", "abc", "abc"),
	)
})
