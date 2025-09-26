// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/common"
)

var _ = Describe("validate url methods", Ordered, func() {
	It("should correctly generate the url for an exposed service", func() {
		common.DNSDomain = "example.com"
		plugin := &v1alpha1.Plugin{
			Spec: v1alpha1.PluginSpec{
				ReleaseNamespace: "test-namespace",
				ClusterName:      "test-cluster",
			},
		}
		plugin.SetNamespace("test-organisation")

		url := common.URLForExposedServiceInPlugin("test-service", plugin)
		Expect(url).To(Equal("https://test-cluster--e30cc9f.test-organisation.example.com"))
	})

	It("should correctly extract the cluster from an host", func() {
		cluster, err := common.ExtractCluster("test-cluster--e30cc9f.test-organisation.example.com")

		Expect(err).ToNot(HaveOccurred())
		Expect(cluster).To(Equal("test-cluster"))
	})

	It("should return an error if the host is invalid", func() {
		invalidHosts := []string{
			"https://test-cluster-e30cc9f.test-organisation.example.com",
			"test-cluster-e30cc9f.example.com",
			"test-cluster--e30cc9f",
		}

		for _, host := range invalidHosts {
			_, err := common.ExtractCluster(host)
			Expect(err).To(HaveOccurred(), "Expected error for host: %s", host)
		}
	})
})
