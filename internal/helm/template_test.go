// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var _ = Describe("Template Processing", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Template Resolution", func() {
		DescribeTable("resolving templates with various Sprig functions",
			func(template string, expectedResult string, expectError bool, greenhouseValues []greenhousev1alpha1.PluginOptionValue) {
				// Combine the greenhouse values with the test template.
				optionValues := append(greenhouseValues, greenhousev1alpha1.PluginOptionValue{
					Name:     "testOption",
					Template: &template,
				})

				resolvedValues, err := ResolveTemplatedValues(ctx, optionValues)

				if expectError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(resolvedValues).ToNot(BeEmpty())

				// Find the testOption in the resolved values.
				var testOptionValue *greenhousev1alpha1.PluginOptionValue
				for i := range resolvedValues {
					if resolvedValues[i].Name == "testOption" {
						testOptionValue = &resolvedValues[i]
						break
					}
				}
				Expect(testOptionValue).ToNot(BeNil(), "testOption should be found in resolved values")
				Expect(testOptionValue.Template).To(BeNil())
				Expect(testOptionValue.Value).ToNot(BeNil())

				var resolvedValue string
				err = json.Unmarshal(testOptionValue.Value.Raw, &resolvedValue)
				Expect(err).ToNot(HaveOccurred())
				Expect(resolvedValue).To(Equal(expectedResult))
			},
			Entry("simple cluster name",
				"{{ .global.greenhouse.clusterName }}",
				"obs-eu-de-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("simple organization name",
				"{{ .global.greenhouse.organizationName }}",
				"test-org", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.organizationName", Value: test.MustReturnJSONFor("test-org")},
				}),
			Entry("upper case transformation",
				"{{ upper .global.greenhouse.clusterName }}",
				"OBS-EU-DE-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("lower case transformation",
				"{{ lower .global.greenhouse.clusterName }}",
				"obs-eu-de-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("title case transformation",
				"{{ title .global.greenhouse.organizationName }}",
				"Test-Org", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.organizationName", Value: test.MustReturnJSONFor("test-org")},
				}),
			Entry("regex split and join",
				`ingress.{{ regexSplit "-" .global.greenhouse.clusterName 2 | join "." }}.my.cloud.`,
				"ingress.obs.eu-de-1.my.cloud.", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("regex split all parts",
				`{{ regexSplit "-" .global.greenhouse.clusterName -1 | join "_" }}`,
				"obs_eu_de_1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("regex split first 3 parts",
				`{{ regexSplit "-" .global.greenhouse.clusterName 3 | join "-" }}`,
				"obs-eu-de-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("https URL construction",
				"https://{{ .global.greenhouse.clusterName }}.example.com",
				"https://obs-eu-de-1.example.com", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("conditional with if-else",
				`{{ if eq .global.greenhouse.clusterName "obs-eu-de-1" }}production{{ else }}staging{{ end }}`,
				"production", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("conditional with contains",
				`{{ if contains "eu" .global.greenhouse.clusterName }}europe{{ else }}other{{ end }}`,
				"europe", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("join cluster names",
				`{{ join "," .global.greenhouse.clusterNames }}`,
				"obs-eu-de-1,obs-eu-de-2", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterNames", Value: test.MustReturnJSONFor([]string{"obs-eu-de-1", "obs-eu-de-2"})},
				}),
			Entry("first cluster name",
				`{{ index .global.greenhouse.clusterNames 0 }}`,
				"obs-eu-de-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterNames", Value: test.MustReturnJSONFor([]string{"obs-eu-de-1", "obs-eu-de-2"})},
				}),
			Entry("trim and replace",
				`{{ trim "  test-value  " | replace "test" "demo" }}`,
				"demo-value", false,
				[]greenhousev1alpha1.PluginOptionValue{}),
			Entry("hasPrefix check",
				`{{ if hasPrefix "obs" .global.greenhouse.clusterName }}yes{{ else }}no{{ end }}`,
				"yes", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("string length",
				`{{ len .global.greenhouse.clusterName }}`,
				"11", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			// Cluster metadata tests.
			Entry("cluster metadata region",
				"{{ .global.greenhouse.metadata.region }}",
				"eu-de-1", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.metadata.region", Value: test.MustReturnJSONFor("eu-de-1")},
				}),
			Entry("cluster metadata environment",
				"{{ .global.greenhouse.metadata.environment }}",
				"production", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.metadata.environment", Value: test.MustReturnJSONFor("production")},
				}),
			Entry("metadata-based URL construction",
				"https://api.{{ .global.greenhouse.metadata.region }}.example.com",
				"https://api.eu-de-1.example.com", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.metadata.region", Value: test.MustReturnJSONFor("eu-de-1")},
				}),
			Entry("conditional with metadata environment",
				`{{ if eq .global.greenhouse.metadata.environment "production" }}prod-config{{ else }}dev-config{{ end }}`,
				"prod-config", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.metadata.environment", Value: test.MustReturnJSONFor("production")},
				}),
			Entry("database connection string",
				`postgres://user:pass@{{ .global.greenhouse.clusterName }}.db.example.com:5432/myapp`,
				"postgres://user:pass@obs-eu-de-1.db.example.com:5432/myapp", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("kubernetes service name",
				`{{ .global.greenhouse.organizationName }}-{{ regexSplit "-" .global.greenhouse.clusterName 2 | join "-" }}-svc`,
				"test-org-obs-eu-de-1-svc", false,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.organizationName", Value: test.MustReturnJSONFor("test-org")},
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
			Entry("invalid template syntax",
				"{{ invalid template }}",
				"", true,
				[]greenhousev1alpha1.PluginOptionValue{}),
			Entry("nonexistent field",
				"{{ .nonexistent.field }}",
				"<no value>", false,
				[]greenhousev1alpha1.PluginOptionValue{}),
			Entry("invalid sprig function",
				"{{ invalidFunction .global.greenhouse.clusterName }}",
				"", true,
				[]greenhousev1alpha1.PluginOptionValue{
					{Name: "global.greenhouse.clusterName", Value: test.MustReturnJSONFor("obs-eu-de-1")},
				}),
		)
	})

	Describe("Edge Cases", func() {
		It("should handle empty template list", func() {
			optionValues := []greenhousev1alpha1.PluginOptionValue{}
			resolvedValues, err := ResolveTemplatedValues(ctx, optionValues)
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
			resolvedValues, err := ResolveTemplatedValues(ctx, optionValues)
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
