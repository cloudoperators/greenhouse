// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

var _ = Describe("ResolveExpressions with feature flag disabled", func() {
	var (
		ctx               context.Context
		baseOptionValues  []greenhousev1alpha1.PluginOptionValue
		globalClusterName string
		globalRegion      string
		globalEnvironment string
	)

	BeforeEach(func() {
		ctx = context.Background()

		globalClusterName = "test-cluster"
		globalRegion = "eu-de-1"
		globalEnvironment = "production"

		clusterNameJSON, _ := json.Marshal(globalClusterName)
		regionJSON, _ := json.Marshal(globalRegion)
		envJSON, _ := json.Marshal(globalEnvironment)

		baseOptionValues = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "global.greenhouse.clusterName",
				Value: &apiextensionsv1.JSON{Raw: clusterNameJSON},
			},
			{
				Name:  "global.greenhouse.metadata.region",
				Value: &apiextensionsv1.JSON{Raw: regionJSON},
			},
			{
				Name:  "global.greenhouse.metadata.environment",
				Value: &apiextensionsv1.JSON{Raw: envJSON},
			},
		}
	})

	It("should treat expression as literal string when disabled", func() {
		expression := "prometheus-${global.greenhouse.metadata.region}-user"
		optionValues := append(baseOptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:       "username",
			Expression: &expression,
		})

		result, err := ResolveExpressions(ctx, optionValues, false)
		Expect(err).ToNot(HaveOccurred())

		var username string
		for _, opt := range result {
			if opt.Name == "username" {
				Expect(opt.Value).ToNot(BeNil())
				err := json.Unmarshal(opt.Value.Raw, &username)
				Expect(err).ToNot(HaveOccurred())
				break
			}
		}

		Expect(username).To(Equal("prometheus-${global.greenhouse.metadata.region}-user"))
	})

	It("should treat multi-line expression as literal string when disabled", func() {
		expression := `endpoint: thanos-grpc.obs.${global.greenhouse.metadata.region}.cloudoperators.dev:443
cluster: ${global.greenhouse.clusterName}
env: ${global.greenhouse.metadata.environment}`

		optionValues := append(baseOptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:       "config",
			Expression: &expression,
		})

		result, err := ResolveExpressions(ctx, optionValues, false)
		Expect(err).ToNot(HaveOccurred())

		var config string
		for _, opt := range result {
			if opt.Name == "config" {
				Expect(opt.Value).ToNot(BeNil())
				err := json.Unmarshal(opt.Value.Raw, &config)
				Expect(err).ToNot(HaveOccurred())
				break
			}
		}

		Expect(config).To(Equal(`endpoint: thanos-grpc.obs.${global.greenhouse.metadata.region}.ccloudoperators.dev:443
cluster: ${global.greenhouse.clusterName}
env: ${global.greenhouse.metadata.environment}`))
	})

	It("should treat CEL expression as literal string when disabled", func() {
		expression := "${global.greenhouse.clusterName.upperAscii()}"
		optionValues := append(baseOptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:       "clusterLabel",
			Expression: &expression,
		})

		result, err := ResolveExpressions(ctx, optionValues, false)
		Expect(err).ToNot(HaveOccurred())

		var clusterLabel string
		for _, opt := range result {
			if opt.Name == "clusterLabel" {
				Expect(opt.Value).ToNot(BeNil())
				err := json.Unmarshal(opt.Value.Raw, &clusterLabel)
				Expect(err).ToNot(HaveOccurred())
				break
			}
		}

		Expect(clusterLabel).To(Equal("${global.greenhouse.clusterName.upperAscii()}"))
	})
})
