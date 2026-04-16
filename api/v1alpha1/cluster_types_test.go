// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("Cluster.ExposedServicesEnabled", func() {
	DescribeTable("should return expected result based on annotation",
		func(annotations map[string]string, expected bool) {
			cluster := &greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-cluster",
					Namespace:   "test-namespace",
					Annotations: annotations,
				},
			}
			Expect(cluster.ExposedServicesEnabled()).To(Equal(expected))
		},
		Entry("no annotations", nil, true),
		Entry("annotation not present", map[string]string{"some-other-annotation": "value"}, true),
		Entry("annotation set to 'true'", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: "true"}, false),
		Entry("annotation set to 'True' (case-insensitive)", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: "True"}, false),
		Entry("annotation set to 'TRUE' (case-insensitive)", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: "TRUE"}, false),
		Entry("annotation set to 'false'", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: "false"}, true),
		Entry("annotation set to empty string", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: ""}, true),
		Entry("annotation set to arbitrary non-true value", map[string]string{greenhousev1alpha1.ServiceProxyDisabledKey: "yes"}, true),
	)

	It("should return false for a nil cluster", func() {
		var cluster *greenhousev1alpha1.Cluster
		Expect(cluster.ExposedServicesEnabled()).To(BeFalse())
	})
})
