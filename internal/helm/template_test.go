// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("Template Processing", func() {
	var (
		ctx    context.Context
		c      client.Client
		plugin *greenhousev1alpha1.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()

		c = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obs-eu-de-1",
					Namespace: "test-org",
					Labels: map[string]string{
						"metadata.greenhouse.sap/region":      "eu-de-1",
						"metadata.greenhouse.sap/environment": "production",
						"other-label":                         "should-be-ignored",
					},
				},
			},
			&greenhousev1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obs-eu-de-2",
					Namespace: "test-org",
				},
			},
			// Test team
			&greenhousev1alpha1.Team{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-team",
					Namespace: "test-org",
				},
			},
		).Build()

		plugin = &greenhousev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plugin",
				Namespace: "test-org",
			},
			Spec: greenhousev1alpha1.PluginSpec{
				ClusterName:      "obs-eu-de-1",
				ReleaseNamespace: "test-org",
				PluginDefinition: "disco",
			},
		}
	})

	Describe("Template Data Building", func() {
		It("should build template data with all greenhouse values including cluster metadata", func() {
			templateData, err := buildTemplateData(ctx, c, plugin)
			Expect(err).ToNot(HaveOccurred())
			Expect(templateData).ToNot(BeNil())

			global, exists := templateData["global"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			greenhouse, exists := global["greenhouse"].(map[string]interface{})
			Expect(exists).To(BeTrue())

			Expect(greenhouse["clusterName"]).To(Equal("obs-eu-de-1"))
			Expect(greenhouse["organizationName"]).To(Equal("test-org"))
			Expect(greenhouse["clusterNames"]).To(ContainElements("obs-eu-de-1", "obs-eu-de-2"))
			Expect(greenhouse["teamNames"]).To(ContainElement("test-team"))
			Expect(greenhouse["baseDomain"]).To(Equal(""))

			metadata, exists := greenhouse["metadata"].(map[string]interface{})
			Expect(exists).To(BeTrue())

			Expect(metadata["region"]).To(Equal("eu-de-1"))
			Expect(metadata["environment"]).To(Equal("production"))

			// Should not include non-metadata labels.
			_, exists = metadata["other-label"]
			Expect(exists).To(BeFalse())
		})

		It("should handle plugins without cluster name gracefully", func() {
			pluginWithoutCluster := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plugin-no-cluster",
					Namespace: "test-org",
				},
				Spec: greenhousev1alpha1.PluginSpec{
					ReleaseNamespace: "test-org",
					PluginDefinition: "disco",
				},
			}

			templateData, err := buildTemplateData(ctx, c, pluginWithoutCluster)
			Expect(err).ToNot(HaveOccurred())

			global, exists := templateData["global"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			greenhouse, exists := global["greenhouse"].(map[string]interface{})
			Expect(exists).To(BeTrue())

			// Should not have metadata when no cluster is specified.
			_, exists = greenhouse["metadata"]
			Expect(exists).To(BeFalse())
			_, exists = greenhouse["clusterName"]
			Expect(exists).To(BeFalse())
		})
	})

	Describe("Template Resolution", func() {
		DescribeTable("resolving templates with various Sprig functions",
			func(template string, expectedResult string, expectError bool) {
				optionValues := []greenhousev1alpha1.PluginOptionValue{
					{
						Name:     "testOption",
						Template: &template,
					},
				}

				resolvedValues, err := ResolveTemplatedValues(ctx, c, plugin, optionValues)

				if expectError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(resolvedValues).To(HaveLen(1))
				Expect(resolvedValues[0].Name).To(Equal("testOption"))
				Expect(resolvedValues[0].Template).To(BeNil())
				Expect(resolvedValues[0].Value).ToNot(BeNil())

				var resolvedValue string
				err = json.Unmarshal(resolvedValues[0].Value.Raw, &resolvedValue)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolvedValue).To(Equal(expectedResult))
			},
			Entry("simple cluster name",
				"{{ .global.greenhouse.clusterName }}",
				"obs-eu-de-1", false),
			Entry("simple organization name",
				"{{ .global.greenhouse.organizationName }}",
				"test-org", false),

			Entry("upper case transformation",
				"{{ upper .global.greenhouse.clusterName }}",
				"OBS-EU-DE-1", false),
			Entry("lower case transformation",
				"{{ lower .global.greenhouse.clusterName }}",
				"obs-eu-de-1", false),
			Entry("title case transformation",
				"{{ title .global.greenhouse.organizationName }}",
				"Test-Org", false),

			Entry("regex split and join",
				`ingress.{{ regexSplit "-" .global.greenhouse.clusterName 2 | join "." }}.my.cloud.`,
				"ingress.obs.eu-de-1.my.cloud.", false),

			Entry("regex split all parts",
				`{{ regexSplit "-" .global.greenhouse.clusterName -1 | join "_" }}`,
				"obs_eu_de_1", false),
			Entry("regex split first 3 parts",
				`{{ regexSplit "-" .global.greenhouse.clusterName 3 | join "-" }}`,
				"obs-eu-de-1", false),

			Entry("https URL construction",
				"https://{{ .global.greenhouse.clusterName }}.example.com",
				"https://obs-eu-de-1.example.com", false),

			Entry("conditional with if-else",
				`{{ if eq .global.greenhouse.clusterName "obs-eu-de-1" }}production{{ else }}staging{{ end }}`,
				"production", false),
			Entry("conditional with contains",
				`{{ if contains "eu" .global.greenhouse.clusterName }}europe{{ else }}other{{ end }}`,
				"europe", false),

			Entry("join cluster names",
				`{{ join "," .global.greenhouse.clusterNames }}`,
				"obs-eu-de-1,obs-eu-de-2", false),
			Entry("first cluster name",
				`{{ index .global.greenhouse.clusterNames 0 }}`,
				"obs-eu-de-1", false),

			Entry("trim and replace",
				`{{ trim "  test-value  " | replace "test" "demo" }}`,
				"demo-value", false),
			Entry("hasPrefix check",
				`{{ if hasPrefix "obs" .global.greenhouse.clusterName }}yes{{ else }}no{{ end }}`,
				"yes", false),

			Entry("string length",
				`{{ len .global.greenhouse.clusterName }}`,
				"11", false),

			// Cluster metadata tests.
			Entry("cluster metadata region",
				"{{ .global.greenhouse.metadata.region }}",
				"eu-de-1", false),
			Entry("cluster metadata environment",
				"{{ .global.greenhouse.metadata.environment }}",
				"production", false),
			Entry("metadata-based URL construction",
				"https://api.{{ .global.greenhouse.metadata.region }}.example.com",
				"https://api.eu-de-1.example.com", false),
			Entry("conditional with metadata environment",
				`{{ if eq .global.greenhouse.metadata.environment "production" }}prod-config{{ else }}dev-config{{ end }}`,
				"prod-config", false),

			Entry("database connection string",
				`postgres://user:pass@{{ .global.greenhouse.clusterName }}.db.example.com:5432/myapp`,
				"postgres://user:pass@obs-eu-de-1.db.example.com:5432/myapp", false),
			Entry("kubernetes service name",
				`{{ .global.greenhouse.organizationName }}-{{ regexSplit "-" .global.greenhouse.clusterName 2 | join "-" }}-svc`,
				"test-org-obs-eu-de-1-svc", false),

			Entry("invalid template syntax",
				"{{ invalid template }}",
				"", true),
			Entry("nonexistent field",
				"{{ .nonexistent.field }}",
				"<no value>", false),
			Entry("invalid sprig function",
				"{{ invalidFunction .global.greenhouse.clusterName }}",
				"", true),
		)
	})

	Describe("Edge Cases", func() {
		It("should handle empty template list", func() {
			optionValues := []greenhousev1alpha1.PluginOptionValue{}
			resolvedValues, err := ResolveTemplatedValues(ctx, c, plugin, optionValues)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolvedValues).To(BeEmpty())
		})

		It("should handle nil template values", func() {
			optionValues := []greenhousev1alpha1.PluginOptionValue{
				{
					Name:     "nilTemplate",
					Template: nil,
					Value:    &apiextensionsv1.JSON{Raw: []byte(`"fallback"`)},
				},
			}
			resolvedValues, err := ResolveTemplatedValues(ctx, c, plugin, optionValues)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolvedValues).To(HaveLen(1))
			Expect(resolvedValues[0].Template).To(BeNil())
			Expect(resolvedValues[0].Value).ToNot(BeNil())
		})
	})

	Describe("extractMetadataKey helper function", func() {
		DescribeTable("extracting metadata keys from various labels",
			func(labelKey, expectedKey string) {
				result := extractMetadataKey(labelKey)
				Expect(result).To(Equal(expectedKey))
			},
			Entry("valid metadata label region", "metadata.greenhouse.sap/region", "region"),
			Entry("valid metadata label environment", "metadata.greenhouse.sap/environment", "environment"),
			Entry("valid metadata label zone", "metadata.greenhouse.sap/zone", "zone"),
			Entry("non-metadata label", "other-label", ""),
			Entry("greenhouse label but not metadata", "greenhouse.sap/some-label", ""),
			Entry("empty label", "", ""),
			Entry("label starting with metadata but wrong format", "metadata/region", ""),
		)
	})
})
