// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/cel"
	"github.com/cloudoperators/greenhouse/internal/test"
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

	Describe("EvaluatePluginExpression", func() {
		var plugin *greenhousev1alpha1.Plugin

		BeforeEach(func() {
			plugin = &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"app":         "test-app",
						"environment": "production",
					},
				},
				Spec: greenhousev1alpha1.PluginSpec{
					PluginDefinition: "test-definition",
					DisplayName:      "Test Plugin",
					ClusterName:      "test-cluster",
					OptionValues: []greenhousev1alpha1.PluginOptionValue{
						{
							Name:  "alertmanager.ingress.url",
							Value: test.MustReturnJSONFor("https://alertmanager.example.com"),
						},
						{
							Name:  "prometheus.ingress.url",
							Value: test.MustReturnJSONFor("https://prometheus.example.com"),
						},
						{
							Name:  "replica.count",
							Value: test.MustReturnJSONFor("3"),
						},
					},
				},
			}
		})

		It("should extract plugin name", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.metadata.name")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-plugin"))
		})

		It("should extract plugin namespace", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.metadata.namespace")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-namespace"))
		})

		It("should extract plugin label", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.metadata.labels.app")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-app"))
		})

		It("should extract spec.displayName", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.spec.displayName")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Test Plugin"))
		})

		It("should extract spec.clusterName", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.spec.clusterName")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("test-cluster"))
		})

		It("should filter optionValues by name and extract value", func() {
			expression := "plugin.spec.optionValues.filter(n, n.name == 'alertmanager.ingress.url').map(k, k.value)[0]"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("https://alertmanager.example.com"))
		})

		It("should filter optionValues for prometheus URL", func() {
			expression := "plugin.spec.optionValues.filter(n, n.name == 'prometheus.ingress.url').map(k, k.value)[0]"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("https://prometheus.example.com"))
		})

		It("should access optionValues as list", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.spec.optionValues.size()")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(3)))
		})

		It("should filter optionValues by name existence", func() {
			expression := "plugin.spec.optionValues.exists(n, n.name == 'alertmanager.ingress.url')"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return false when checking for non-existent optionValue", func() {
			expression := "plugin.spec.optionValues.exists(n, n.name == 'non.existent.value')"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("should evaluate conditional expressions", func() {
			expression := "plugin.metadata.labels.environment == 'production' ? 'prod' : 'dev'"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("prod"))
		})

		It("should evaluate logical AND", func() {
			expression := "plugin.metadata.name == 'test-plugin' && plugin.metadata.namespace == 'test-namespace'"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should evaluate logical OR", func() {
			expression := "plugin.metadata.name == 'wrong-name' || plugin.metadata.namespace == 'test-namespace'"
			result, err := evaluator.EvaluatePluginExpression(plugin, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return error for nil plugin", func() {
			result, err := evaluator.EvaluatePluginExpression(nil, "plugin.metadata.name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin cannot be nil"))
			Expect(result).To(BeNil())
		})

		It("should return error for empty expression", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.metadata.name ===")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(result).To(BeNil())
		})

		It("should return error when accessing non-existent fields", func() {
			result, err := evaluator.EvaluatePluginExpression(plugin, "plugin.spec.nonExistentField")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such key"))
			Expect(result).To(BeNil())
		})
	})

	Describe("EvaluatePluginListExpression", func() {
		var plugins []greenhousev1alpha1.Plugin

		BeforeEach(func() {
			plugins = []greenhousev1alpha1.Plugin{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alertmanager-1",
						Namespace: "monitoring",
						Labels: map[string]string{
							"app":  "alertmanager",
							"type": "monitoring",
						},
					},
					Spec: greenhousev1alpha1.PluginSpec{
						ClusterName: "cluster-1",
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "alertmanager.ingress.url",
								Value: test.MustReturnJSONFor("https://alertmanager-1.example.com"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-1",
						Namespace: "monitoring",
						Labels: map[string]string{
							"app":  "prometheus",
							"type": "monitoring",
						},
					},
					Spec: greenhousev1alpha1.PluginSpec{
						ClusterName: "cluster-1",
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "prometheus.ingress.url",
								Value: test.MustReturnJSONFor("https://prometheus-1.example.com"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-2",
						Namespace: "monitoring",
						Labels: map[string]string{
							"app":  "prometheus",
							"type": "monitoring",
						},
					},
					Spec: greenhousev1alpha1.PluginSpec{
						ClusterName: "cluster-2",
						OptionValues: []greenhousev1alpha1.PluginOptionValue{
							{
								Name:  "prometheus.ingress.url",
								Value: test.MustReturnJSONFor("https://prometheus-2.example.com"),
							},
						},
					},
				},
			}
		})

		It("should return the size of the plugin list", func() {
			result, err := evaluator.EvaluatePluginListExpression(plugins, "plugins.size()")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(3)))
		})

		It("should filter plugins by label", func() {
			expression := "plugins.filter(p, p.metadata.labels.app == 'prometheus').size()"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(2)))
		})

		It("should map plugin names", func() {
			expression := "plugins.map(p, p.metadata.name)"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(3))
			Expect(resultList).To(ContainElement("alertmanager-1"))
			Expect(resultList).To(ContainElement("prometheus-1"))
			Expect(resultList).To(ContainElement("prometheus-2"))
		})

		It("should extract prometheus URLs", func() {
			expression := `plugins.filter(p, p.metadata.labels.app == 'prometheus').map(p, p.spec.optionValues.filter(o, o.name == 'prometheus.ingress.url').map(v, v.value)[0])`
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(2))
			Expect(resultList).To(ContainElement("https://prometheus-1.example.com"))
			Expect(resultList).To(ContainElement("https://prometheus-2.example.com"))
		})

		It("should filter and count plugins by cluster", func() {
			expression := "plugins.filter(p, p.spec.clusterName == 'cluster-1').size()"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(2)))
		})

		It("should check if any plugin matches condition", func() {
			expression := "plugins.exists(p, p.metadata.labels.app == 'alertmanager')"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should check if all plugins match condition", func() {
			expression := "plugins.all(p, p.metadata.namespace == 'monitoring')"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return error for empty plugin list", func() {
			result, err := evaluator.EvaluatePluginListExpression([]greenhousev1alpha1.Plugin{}, "plugins.size()")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugins list cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for empty expression", func() {
			result, err := evaluator.EvaluatePluginListExpression(plugins, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expression cannot be empty"))
			Expect(result).To(BeNil())
		})

		It("should return error for invalid expression syntax", func() {
			result, err := evaluator.EvaluatePluginListExpression(plugins, "plugins.size() ===")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			Expect(result).To(BeNil())
		})

		It("should combine filter and map operations", func() {
			expression := "plugins.filter(p, p.metadata.labels.app == 'prometheus').map(p, p.metadata.name)"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())

			resultList, ok := result.([]any)
			Expect(ok).To(BeTrue())
			Expect(resultList).To(HaveLen(2))
			Expect(resultList).To(ContainElement("prometheus-1"))
			Expect(resultList).To(ContainElement("prometheus-2"))
		})

		It("should extract first matching plugin", func() {
			expression := "plugins.filter(p, p.metadata.labels.app == 'alertmanager')[0].metadata.name"
			result, err := evaluator.EvaluatePluginListExpression(plugins, expression)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("alertmanager-1"))
		})
	})
})
