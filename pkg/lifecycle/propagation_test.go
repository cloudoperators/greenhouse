// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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

		updated := lifecycle.NewPropagator(src, dst).Apply()

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
		map[string]string{"greenhouse.sap/propagate-labels": "region"},
		map[string]string{},
		nil,
		map[string]string{"region": "bar"},
		true,
		[]string{"region"}),

	Entry("It should not propagate any labels as declared label key is missing",
		map[string]string{},
		map[string]string{"greenhouse.sap/propagate-labels": "region"},
		map[string]string{},
		[]string{"region"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should remove the previous declared label key due to state change",
		map[string]string{"region": "bar"},
		map[string]string{"greenhouse.sap/propagate-labels": "support_group"},
		map[string]string{},
		[]string{"region", "support_group"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should propagate all declared label keys",
		map[string]string{"region": "bar", "support_group": "x"},
		map[string]string{"greenhouse.sap/propagate-labels": "support_group,region"},
		map[string]string{},
		[]string{"region"},
		map[string]string{"region": "bar", "support_group": "x"},
		true,
		[]string{"region", "support_group"}),

	Entry("It should propagate all labels matching the wildcard",
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2", "support_group": "x"},
		map[string]string{"greenhouse.sap/propagate-labels": "metadata.greenhouse.sap/*"},
		map[string]string{},
		nil,
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2"},
		true,
		[]string{"metadata.greenhouse.sap/test1", "metadata.greenhouse.sap/test2"}),

	Entry("It should not remove labels matching the wildcard after the state change",
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2", "support_group": "x"},
		map[string]string{"greenhouse.sap/propagate-labels": "metadata.greenhouse.sap/*"},
		map[string]string{"metadata.greenhouse.sap/test1": "value1"},
		[]string{"metadata.greenhouse.sap/test1"},
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2"},
		true,
		[]string{"metadata.greenhouse.sap/test1", "metadata.greenhouse.sap/test2"}),

	Entry("It should remove previous labels after wildcard removal in the state change",
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2", "support_group": "x"},
		map[string]string{"greenhouse.sap/propagate-labels": "metadata.greenhouse.sap/test2"},
		map[string]string{"metadata.greenhouse.sap/test1": "value1", "metadata.greenhouse.sap/test2": "value2"},
		[]string{"metadata.greenhouse.sap/test1", "metadata.greenhouse.sap/test2"},
		map[string]string{"metadata.greenhouse.sap/test2": "value2"},
		true,
		[]string{"metadata.greenhouse.sap/test2"}),
)

var _ = DescribeTable("Annotation propagation scenarios",
	func(srcAnno map[string]string, srcDecl map[string]string, dstAnno map[string]string, prevStateAnno []string, expected map[string]string, expectedAnnotation bool, expectedKeys []string) {
		// Build source object with both annotations and labels (labels irrelevant here)
		src := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "src",
				Namespace:   "default",
				Annotations: srcAnno,
			},
		}

		dst := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "dst",
				Namespace:   "default",
				Annotations: dstAnno,
			},
		}

		// Previous state may have annotationKeys tracked
		if len(prevStateAnno) > 0 {
			// marshal state with annotationKeys
			var s struct {
				AnnotationKeys []string `json:"annotationKeys"`
			}
			s.AnnotationKeys = prevStateAnno
			data, _ := json.Marshal(s)
			if dst.GetAnnotations() == nil {
				dst.SetAnnotations(map[string]string{})
			}
			dst.GetAnnotations()[lifecycle.AppliedPropagatorAnnotation] = string(data)
		}

		// Merge the declaration of which annotations to propagate into src annotations
		if src.GetAnnotations() == nil {
			src.SetAnnotations(map[string]string{})
		}
		for k, v := range srcDecl {
			src.GetAnnotations()[k] = v
		}

		updated := lifecycle.NewPropagator(src, dst).Apply()
		// Build expected annotations map including propagated keys but excluding tracking annotation
		// Start with the resulting annotations and remove the tracking annotation to compare
		got := map[string]string{}
		for k, v := range updated.GetAnnotations() {
			if k == lifecycle.AppliedPropagatorAnnotation {
				continue
			}
			got[k] = v
		}
		Expect(got).To(Equal(expected))
		if expectedAnnotation {
			annotation := updated.GetAnnotations()[lifecycle.AppliedPropagatorAnnotation]
			var actual struct {
				AnnotationKeys []string `json:"annotationKeys"`
			}
			_ = json.Unmarshal([]byte(annotation), &actual) //nolint:errcheck
			Expect(actual.AnnotationKeys).To(ConsistOf(expectedKeys))
		} else {
			// state annotation should not exist if nothing propagated
			Expect(updated.GetAnnotations()).ToNot(HaveKey(lifecycle.AppliedPropagatorAnnotation))
		}
	},

	Entry("It should not propagate any annotations as there is no propagation declaration",
		map[string]string{"team": "a"},
		nil,
		map[string]string{},
		nil,
		map[string]string{},
		false,
		[]string{}),

	Entry("It should propagate declared annotation key",
		map[string]string{"team": "a"},
		map[string]string{"greenhouse.sap/propagate-annotations": "team"},
		map[string]string{},
		nil,
		map[string]string{"team": "a"},
		true,
		[]string{"team"}),

	Entry("It should not propagate any annotations as declared annotation key is missing",
		map[string]string{},
		map[string]string{"greenhouse.sap/propagate-annotations": "team"},
		map[string]string{},
		[]string{"team"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should remove the previous declared annotation key due to state change",
		map[string]string{"team": "a"},
		map[string]string{"greenhouse.sap/propagate-annotations": "department"},
		map[string]string{},
		[]string{"team", "department"},
		map[string]string{},
		false,
		[]string{}),

	Entry("It should propagate all declared annotation keys",
		map[string]string{"team": "a", "owner": "x"},
		map[string]string{"greenhouse.sap/propagate-annotations": "owner,team"},
		map[string]string{},
		[]string{"team"},
		map[string]string{"team": "a", "owner": "x"},
		true,
		[]string{"team", "owner"}),
)
