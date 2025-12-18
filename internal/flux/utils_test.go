// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package flux_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/internal/flux"
)

var _ = Describe("ChartURLToName", func() {
	DescribeTable("converts repository URLs to valid k8s names",
		func(url, expected string) {
			Expect(flux.ChartURLToName(url)).To(Equal(expected))
		},
		Entry("oci url", "oci://ghcr.io/cloudoperators/charts", "ghcr-io-cloudoperators-charts"),
		Entry("https url", "https://charts.example.com/repo", "charts-example-com-repo"),
		Entry("trailing slash stripped", "oci://ghcr.io/cloudoperators/charts/", "ghcr-io-cloudoperators-charts"),
		Entry("keppel registry from issue #1621", "oci://keppel.eu-de-1.cloud.sap/ccloud-helm/", "keppel-eu-de-1-cloud-sap-ccloud-helm"),
	)
})

var _ = Describe("GetSourceRepositoryType", func() {
	DescribeTable("returns correct repository type",
		func(url, expected string) {
			Expect(flux.GetSourceRepositoryType(url)).To(Equal(expected))
		},
		Entry("oci", "oci://ghcr.io/charts", "oci"),
		Entry("https", "https://charts.example.com", "default"),
	)
})
