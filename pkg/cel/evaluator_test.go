// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/internal/controller/fixtures"
	"github.com/cloudoperators/greenhouse/pkg/cel"
)

func TestCEL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CEL Suite")
}

var _ = Describe("CEL Evaluator", func() {
	Describe("Evaluate", func() {
		var dummy *fixtures.Dummy

		BeforeEach(func() {
			dummy = &fixtures.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dummy",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"app":  "test-app",
						"tier": "backend",
					},
				},
				Spec: fixtures.DummySpec{
					Description:    "Test Description",
					Property:       "test-property",
					SecondProperty: "test-second-property",
				},
			}
		})

		It("should extract object name", func() {
			result, err := cel.Evaluate("object.metadata.name", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-dummy"))
		})

		It("should extract object namespace", func() {
			result, err := cel.Evaluate("object.metadata.namespace", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-namespace"))
		})

		It("should extract object label", func() {
			result, err := cel.Evaluate("object.metadata.labels.app", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-app"))
		})

		It("should extract spec.description", func() {
			result, err := cel.Evaluate("object.spec.description", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Test Description"))
		})

		It("should extract spec.property", func() {
			result, err := cel.Evaluate("object.spec.property", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-property"))
		})

		It("should extract spec.secondProperty", func() {
			result, err := cel.Evaluate("object.spec.secondProperty", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-second-property"))
		})

		It("should evaluate conditional expressions", func() {
			expression := "object.metadata.labels.tier == 'backend' ? 'backend' : 'frontend'"
			result, err := cel.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("backend"))
		})

		It("should evaluate logical AND", func() {
			expression := "object.metadata.name == 'test-dummy' && object.metadata.namespace == 'test-namespace'"
			result, err := cel.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should evaluate logical OR", func() {
			expression := "object.metadata.name == 'wrong-name' || object.metadata.namespace == 'test-namespace'"
			result, err := cel.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return error for nil object", func() {
			result, err := cel.Evaluate("object.metadata.name", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("object cannot be nil"))
			Expect(result).To(BeNil())
		})

		It("should return error for empty expression", func() {
			result, err := cel.Evaluate("", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			result, err := cel.Evaluate("object.metadata.name ===", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(result).To(BeNil())
		})

		It("should return error when accessing non-existent fields", func() {
			result, err := cel.Evaluate("object.spec.nonExistentField", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such key"))
			Expect(result).To(BeNil())
		})

		It("should evaluate into typed fields", func() {
			result, err := cel.EvaluateTyped[*fixtures.DummySpec]("object.spec", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result).To(BeAssignableToTypeOf(&fixtures.DummySpec{}))

			ptrStr, err := cel.EvaluateTyped[*string]("object.metadata.name", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(ptrStr).ToNot(BeNil())
			Expect(ptrStr).To(BeAssignableToTypeOf(new(string)))
			Expect(*ptrStr).To(Equal("test-dummy"))

			labels, err := cel.EvaluateTyped[map[string]string]("object.metadata.labels", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).ToNot(BeNil())
			Expect(labels).To(BeAssignableToTypeOf(map[string]string{}))
			Expect(labels["app"]).To(Equal("test-app"))
		})
	})

	Describe("EvaluateList", func() {
		var dummies []client.Object

		BeforeEach(func() {
			dummies = []client.Object{
				&fixtures.Dummy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dummy-1",
						Namespace: "test",
						Labels: map[string]string{
							"app":  "backend",
							"tier": "api",
						},
					},
					Spec: fixtures.DummySpec{
						Description: "First dummy",
						Property:    "value-1",
					},
				},
				&fixtures.Dummy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dummy-2",
						Namespace: "test",
						Labels: map[string]string{
							"app":  "frontend",
							"tier": "web",
						},
					},
					Spec: fixtures.DummySpec{
						Description: "Second dummy",
						Property:    "value-2",
					},
				},
				&fixtures.Dummy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dummy-3",
						Namespace: "test",
						Labels: map[string]string{
							"app":  "backend",
							"tier": "worker",
						},
					},
					Spec: fixtures.DummySpec{
						Description: "Third dummy",
						Property:    "value-3",
					},
				},
			}
		})

		It("should extract names from all objects", func() {
			results, err := cel.EvaluateList("object.metadata.name", dummies)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results).To(Equal([]any{"dummy-1", "dummy-2", "dummy-3"}))
		})

		It("should extract properties from all objects", func() {
			results, err := cel.EvaluateList("object.spec.property", dummies)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results).To(Equal([]any{"value-1", "value-2", "value-3"}))
		})

		It("should evaluate conditional expression for each object", func() {
			expression := "object.metadata.labels.app == 'backend' ? 'BE' : 'FE'"
			results, err := cel.EvaluateList(expression, dummies)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results).To(Equal([]any{"BE", "FE", "BE"}))
		})

		It("should return error for empty object list", func() {
			results, err := cel.EvaluateList("object.metadata.name", []client.Object{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one object must be provided"))
			Expect(results).To(BeNil())
		})

		It("should return error for empty expression", func() {
			results, err := cel.EvaluateList("", dummies)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(results).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			results, err := cel.EvaluateList("object.metadata.name ===", dummies)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(results).To(BeNil())
		})

		It("should return error when one object fails evaluation", func() {
			dummies[1] = nil
			results, err := cel.EvaluateList("object.metadata.name", dummies)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to evaluate object at index 1"))
			Expect(results).To(BeNil())
		})
	})
})
