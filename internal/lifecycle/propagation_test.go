// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

var _ = DescribeTable("Label propagation scenarios",
	func(srcLabels map[string]string, srcAnno map[string]string, dstLabels map[string]string, prevState []string, expected map[string]string, expectedAnnotation bool, expectedKeys []string) {
		src := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "src",
				Namespace:   "default",
				Annotations: srcAnno,
				Labels:      srcLabels,
			},
		}

		dst := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dst",
				Namespace: "default",
				Labels:    dstLabels,
			},
		}

		if len(prevState) > 0 {
			data, _ := json.Marshal(map[string][]string{"labelKeys": prevState})
			dst.SetAnnotations(map[string]string{
				lifecycle.AppliedPropagatorAnnotation: string(data),
			})
		}

		updated := lifecycle.NewPropagator(src, dst).ApplyLabels()

		Expect(updated.GetLabels()).To(Equal(expected))
		if expectedAnnotation {
			annotation := updated.GetAnnotations()[lifecycle.AppliedPropagatorAnnotation]
			var actual struct {
				LabelKeys []string `json:"labelKeys"`
			}
			_ = json.Unmarshal([]byte(annotation), &actual) //nolint:errcheck
			Expect(actual.LabelKeys).To(ConsistOf(expectedKeys))
		} else {
			Expect(updated.GetAnnotations()).ToNot(HaveKey(lifecycle.AppliedPropagatorAnnotation))
		}
	},

	Entry("It should not propagate any labels as there is no propagation annotation",
		map[string]string{"region": "bar"},
		nil,
		map[string]string{},
		nil,
		map[string]string{},
		false,
		[]string{}),

	Entry("It should propagate declared label key",
		map[string]string{"region": "bar"},
		map[string]string{"greenhouse.sap/propagate-labels": `{"keys": ["region"]}`},
		map[string]string{},
		nil,
		map[string]string{"region": "bar"},
		true,
		[]string{"region"}),

	Entry("It should not propagate any labels as declared label key is missing",
		map[string]string{},
		map[string]string{"greenhouse.sap/propagate-labels": `{"keys": ["region"]}`},
		map[string]string{},
		[]string{"region"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should remove the previous declared label key due to state change",
		map[string]string{"region": "bar"},
		map[string]string{"greenhouse.sap/propagate-labels": `{"keys": ["support_group"]}`},
		map[string]string{},
		[]string{"region", "support_group"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should propagate all declared label keys",
		map[string]string{"region": "bar", "support_group": "x"},
		map[string]string{"greenhouse.sap/propagate-labels": `{"keys": ["region", "support_group"]}`},
		map[string]string{},
		[]string{"region"},
		map[string]string{"region": "bar", "support_group": "x"},
		true,
		[]string{"region", "support_group"}),
)
