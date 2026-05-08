// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudoperators/greenhouse/internal/controller/fixtures"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

var _ = Describe("Context", func() {
	Describe("ReceiveObjectCopy", func() {
		It("should receive the old copy", func() {
			testResource := &fixtures.Dummy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "default",
					Labels: map[string]string{
						"key1": "value1",
					},
					Annotations: map[string]string{
						"annotation1": "value1",
					},
				},
			}

			ctx := lifecycle.CreateContextFromRuntimeObject(context.Background(), testResource)
			testResource.GetLabels()["key1"] = "value2"
			origResource, err := lifecycle.GetOriginalResourceFromContext(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(origResource.GetLabels()["key1"]).To(Equal("value1"))
			Expect(testResource.GetLabels()["key1"]).To(Equal("value2"))
		})
	})
})
