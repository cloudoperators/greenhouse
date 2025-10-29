// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin CEL Expression Evaluation", func() {
	Describe("CEL Expression Resolution", func() {
		DescribeTable("resolving CEL expressions with various functions",
			func(expression string, expectedResult any, expectError bool, templateData map[string]any) {
				result, err := EvaluatePluginExpression(expression, templateData)

				if expectError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(expectedResult))
			},
			Entry("simple cluster name",
				"global.greenhouse.clusterName",
				"obs-eu-de-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("simple organization name",
				"global.greenhouse.organizationName",
				"test-org", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"organizationName": "test-org",
						},
					},
				}),
			Entry("upper case transformation",
				"global.greenhouse.clusterName.upperAscii()",
				"OBS-EU-DE-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("lower case transformation",
				"global.greenhouse.clusterName.lowerAscii()",
				"obs-eu-de-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("split and join with limit",
				"global.greenhouse.clusterName.split('-', 2).join('.')",
				"obs.eu-de-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("split all parts and join",
				"global.greenhouse.clusterName.split('-').join('_')",
				"obs_eu_de_1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("https URL construction",
				"'https://' + global.greenhouse.clusterName + '.example.com'",
				"https://obs-eu-de-1.example.com", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("complex URL construction with split/join",
				"'ingress.' + global.greenhouse.clusterName.split('-', 2).join('.') + '.my.cloud.'",
				"ingress.obs.eu-de-1.my.cloud.", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("conditional with ternary operator",
				"global.greenhouse.clusterName == 'obs-eu-de-1' ? 'production' : 'staging'",
				"production", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),

			Entry("conditional with contains",
				"global.greenhouse.clusterName.contains('eu') ? 'europe' : 'other'",
				"europe", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("join cluster names",
				"global.greenhouse.clusterNames.join(',')",
				"obs-eu-de-1,obs-eu-de-2", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterNames": []string{"obs-eu-de-1", "obs-eu-de-2"},
						},
					},
				}),
			Entry("first cluster name by index",
				"global.greenhouse.clusterNames[0]",
				"obs-eu-de-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterNames": []any{"obs-eu-de-1", "obs-eu-de-2"},
						},
					},
				}),
			Entry("trim and replace",
				"'  test-value  '.trim().replace('test', 'demo')",
				"demo-value", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{},
					},
				}),

			Entry("startsWith check",
				"global.greenhouse.clusterName.startsWith('obs') ? 'yes' : 'no'",
				"yes", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("string size",
				"size(global.greenhouse.clusterName)",
				int64(11), false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("cluster metadata region",
				"global.greenhouse.metadata.region",
				"eu-de-1", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"metadata": map[string]any{
								"region": "eu-de-1",
							},
						},
					},
				}),
			Entry("cluster metadata environment",
				"global.greenhouse.metadata.environment",
				"production", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"metadata": map[string]any{
								"environment": "production",
							},
						},
					},
				}),
			Entry("metadata-based URL construction",
				"'https://api.' + global.greenhouse.metadata.region + '.example.com'",
				"https://api.eu-de-1.example.com", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"metadata": map[string]any{
								"region": "eu-de-1",
							},
						},
					},
				}),
			Entry("conditional with metadata environment",
				"global.greenhouse.metadata.environment == 'production' ? 'prod-config' : 'dev-config'",
				"prod-config", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"metadata": map[string]any{
								"environment": "production",
							},
						},
					},
				}),
			Entry("database connection string",
				"'postgres://user:pass@' + global.greenhouse.clusterName + '.db.example.com:5432/myapp'",
				"postgres://user:pass@obs-eu-de-1.db.example.com:5432/myapp", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "obs-eu-de-1",
						},
					},
				}),
			Entry("kubernetes service name",
				"global.greenhouse.organizationName + '-' + global.greenhouse.clusterName.split('-', 2).join('-') + '-svc'",
				"test-org-obs-eu-de-1-svc", false,
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"organizationName": "test-org",
							"clusterName":      "obs-eu-de-1",
						},
					},
				}),

			Entry("empty expression",
				"",
				nil, true,
				map[string]any{}),
		)
	})

	Describe("Edge Cases", func() {
		It("should handle nonexistent field", func() {
			result, err := EvaluatePluginExpression(
				"global.nonexistent.field",
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterName": "test",
						},
					},
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
		It("should handle complex nested access", func() {
			result, err := EvaluatePluginExpression(
				"global.greenhouse.metadata.region",
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"metadata": map[string]any{
								"region": "us-west-1",
								"zone":   "a",
							},
						},
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("us-west-1"))
		})
		It("should handle list size", func() {
			result, err := EvaluatePluginExpression(
				"size(global.greenhouse.clusterNames)",
				map[string]any{
					"global": map[string]any{
						"greenhouse": map[string]any{
							"clusterNames": []any{"cluster1", "cluster2", "cluster3"},
						},
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(3)))
		})
	})
})
