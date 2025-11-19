// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cel

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("YAML Expression Evaluation", func() {
	var templateData map[string]any

	BeforeEach(func() {
		templateData = map[string]any{
			"global": map[string]any{
				"greenhouse": map[string]any{
					"clusterName": "obs-eu-de-1",
					"metadata": map[string]any{
						"region":      "eu-de-1",
						"environment": "production",
					},
				},
			},
		}
	})

	It("should resolve basic string interpolation", func() {
		yamlStr := "username: prometheus-${global.greenhouse.metadata.region}-thanos"
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).ToNot(HaveOccurred())
		var resultMap map[string]any
		err = json.Unmarshal(jsonBytes, &resultMap)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultMap["username"]).To(Equal("prometheus-eu-de-1-thanos"))
	})

	It("should handle static YAML without expressions", func() {
		yamlStr := `
name: static-value
port: 8080
enabled: true
`
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).ToNot(HaveOccurred())
		var resultMap map[string]any
		err = json.Unmarshal(jsonBytes, &resultMap)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultMap["name"]).To(Equal("static-value"))
		Expect(resultMap["port"]).To(BeNumerically("==", 8080))
		Expect(resultMap["enabled"]).To(Equal(true))
	})

	It("should resolve direct arrays with expressions", func() {
		yamlStr := `
- thanos-grpc.obs.${global.greenhouse.metadata.region}.cloud.sap:443
- thanos-grpc.mon.${global.greenhouse.metadata.region}.cloud.sap:443
`
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).ToNot(HaveOccurred())
		var resultArray []any
		err = json.Unmarshal(jsonBytes, &resultArray)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultArray).To(HaveLen(2))
		Expect(resultArray[0]).To(Equal("thanos-grpc.obs.eu-de-1.cloud.sap:443"))
		Expect(resultArray[1]).To(Equal("thanos-grpc.mon.eu-de-1.cloud.sap:443"))
	})

	It("should resolve nested maps with mixed static and dynamic values", func() {
		yamlStr := `
swift:
  auth_url: https://identity-3.greenhouse.dev/v3
  region_name: ${global.greenhouse.metadata.region}
  username: prometheus-${global.greenhouse.metadata.region}-thanos
database:
  host: db-${global.greenhouse.metadata.region}.cloud.sap
  port: 5432
`
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).ToNot(HaveOccurred())
		var resultMap map[string]any
		err = json.Unmarshal(jsonBytes, &resultMap)
		Expect(err).ToNot(HaveOccurred())

		swift, ok := resultMap["swift"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(swift["auth_url"]).To(Equal("https://identity-3.greenhouse.dev/v3"))
		Expect(swift["region_name"]).To(Equal("eu-de-1"))
		Expect(swift["username"]).To(Equal("prometheus-eu-de-1-thanos"))

		database, ok := resultMap["database"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(database["host"]).To(Equal("db-eu-de-1.cloud.sap"))
		Expect(database["port"]).To(BeNumerically("==", 5432))
	})

	It("should support CEL string functions", func() {
		yamlStr := `
uppercase: ${global.greenhouse.clusterName.upperAscii()}
transformed: ${global.greenhouse.clusterName.split('-').join('_')}
`
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).ToNot(HaveOccurred())
		var resultMap map[string]any
		err = json.Unmarshal(jsonBytes, &resultMap)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultMap["uppercase"]).To(Equal("OBS-EU-DE-1"))
		Expect(resultMap["transformed"]).To(Equal("obs_eu_de_1"))
	})

	It("should return error for empty expression", func() {
		jsonBytes, err := EvaluateYamlExpression("", templateData)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot be empty"))
		Expect(jsonBytes).To(BeNil())
	})

	It("should return error for invalid CEL expression", func() {
		yamlStr := "value: ${global.nonexistent.field}"
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to evaluate CEL expression"))
		Expect(jsonBytes).To(BeNil())
	})

	It("should return error for invalid YAML syntax", func() {
		yamlStr := `{name: value, list: [item1, item2`
		jsonBytes, err := EvaluateYamlExpression(yamlStr, templateData)

		Expect(err).To(HaveOccurred())
		Expect(jsonBytes).To(BeNil())
	})
})
