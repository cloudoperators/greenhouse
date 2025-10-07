// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/internal/cel"
	"github.com/cloudoperators/greenhouse/internal/controller/fixtures"
)

func TestCEL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CEL Evaluator Suite")
}

var _ = Describe("CEL Evaluator", func() {
	var (
		evaluator *cel.Evaluator
		err       error
	)

	BeforeEach(func() {
		evaluator, err = cel.NewEvaluator()
		Expect(err).ToNot(HaveOccurred(), "evaluator should be created successfully")
		Expect(evaluator).ToNot(BeNil(), "evaluator should not be nil")
	})

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
			result, err := evaluator.Evaluate("object.metadata.name", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-dummy"))
		})

		It("should extract object namespace", func() {
			result, err := evaluator.Evaluate("object.metadata.namespace", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-namespace"))
		})

		It("should extract object label", func() {
			result, err := evaluator.Evaluate("object.metadata.labels.app", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-app"))
		})

		It("should extract spec.description", func() {
			result, err := evaluator.Evaluate("object.spec.description", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Test Description"))
		})

		It("should extract spec.property", func() {
			result, err := evaluator.Evaluate("object.spec.property", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-property"))
		})

		It("should extract spec.secondProperty", func() {
			result, err := evaluator.Evaluate("object.spec.secondProperty", dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-second-property"))
		})

		It("should evaluate conditional expressions", func() {
			expression := "object.metadata.labels.tier == 'backend' ? 'backend' : 'frontend'"
			result, err := evaluator.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("backend"))
		})

		It("should evaluate logical AND", func() {
			expression := "object.metadata.name == 'test-dummy' && object.metadata.namespace == 'test-namespace'"
			result, err := evaluator.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should evaluate logical OR", func() {
			expression := "object.metadata.name == 'wrong-name' || object.metadata.namespace == 'test-namespace'"
			result, err := evaluator.Evaluate(expression, dummy)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return error for nil object", func() {
			result, err := evaluator.Evaluate("object.metadata.name", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("object cannot be nil"))
			Expect(result).To(BeNil())
		})

		It("should return error for empty expression", func() {
			result, err := evaluator.Evaluate("", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			result, err := evaluator.Evaluate("object.metadata.name ===", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(result).To(BeNil())
		})

		It("should return error when accessing non-existent fields", func() {
			result, err := evaluator.Evaluate("object.spec.nonExistentField", dummy)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such key"))
			Expect(result).To(BeNil())
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

		It("should return the size of the list", func() {
			result, err := evaluator.Evaluate("objects.size()", dummies...)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(3)))
		})

		It("should filter objects by label", func() {
			expression := "objects.filter(d, d.metadata.labels.app == 'backend').size()"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(2)))
		})

		It("should map object names", func() {
			expression := "objects.map(d, d.metadata.name)"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(3))
			Expect(resultList).To(ContainElement("dummy-1"))
			Expect(resultList).To(ContainElement("dummy-2"))
			Expect(resultList).To(ContainElement("dummy-3"))
		})

		It("should extract properties from filtered objects", func() {
			expression := "objects.filter(d, d.metadata.labels.app == 'backend').map(d, d.spec.property)"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(2))
			Expect(resultList).To(ContainElement("value-1"))
			Expect(resultList).To(ContainElement("value-3"))
		})

		It("should check if any object matches condition", func() {
			expression := "objects.exists(d, d.metadata.labels.app == 'frontend')"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should check if all objects match condition", func() {
			expression := "objects.all(d, d.metadata.namespace == 'test')"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return error for empty object list", func() {
			result, err := evaluator.Evaluate("objects.size()")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one object must be provided"))
			Expect(result).To(BeNil())
		})

		It("should return error for empty expression", func() {
			result, err := evaluator.Evaluate("", dummies...)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			result, err := evaluator.Evaluate("objects.size() ===", dummies...)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(result).To(BeNil())
		})

		It("should combine filter and map operations", func() {
			expression := "objects.filter(d, d.metadata.labels.tier == 'api').map(d, d.metadata.name)"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(1))
			Expect(resultList).To(ContainElement("dummy-1"))
		})

		It("should extract first matching object", func() {
			expression := "objects.filter(d, d.metadata.labels.app == 'backend')[0].metadata.name"
			result, err := evaluator.Evaluate(expression, dummies...)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("dummy-1"))
		})
	})
})
