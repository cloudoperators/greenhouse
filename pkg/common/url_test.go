// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"testing"

	"github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCommonURL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CommonURl")
}

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
		Expect(url).To(Equal("https://test-service--test-namespace--test-cluster.test-organisation.example.com"))
	})
})
