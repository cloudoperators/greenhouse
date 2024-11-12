// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"github.com/cloudoperators/greenhouse/e2e/shared"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

func PrepareNginxPluginDefinition(namespace string) *greenhousev1alpha1.PluginDefinition {
	return shared.PreparePluginDefinition("nginx-18.1.7", namespace,
		shared.WithVersion("18.1.7"),
		shared.WithDescription("TestPluginDefinition"),
		shared.WithHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "bitnamicharts/nginx",
				Repository: "oci://registry-1.docker.io",
				Version:    "18.1.7",
			},
		),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("false")},
			Description: "autoscaling.enabled",
			Name:        "autoscaling.enabled",
			Type:        "bool",
		}),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
			Description: "autoscaling.maxReplicas",
			Name:        "autoscaling.maxReplicas",
			Type:        "string",
		}),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
			Description: "autoscaling.minReplicas",
			Name:        "autoscaling.minReplicas",
			Type:        "string",
		}),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
			Description: "containerSecurityContext.enabled",
			Name:        "containerSecurityContext.enabled",
			Type:        "bool",
		}),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("1")},
			Description: "replicaCount",
			Name:        "replicaCount",
			Type:        "int",
		}),
		shared.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
			Description: "podSecurityContext.enabled",
			Name:        "podSecurityContext.enabled",
			Type:        "bool",
		}),
	)
}
