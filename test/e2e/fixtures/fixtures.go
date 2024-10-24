// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

func CreateNginxPluginDefinition(ctx context.Context, setup *test.TestSetup) *greenhousev1alpha1.PluginDefinition {
	return setup.CreatePluginDefinition(ctx, "nginx-18.1.7",
		test.WithVersion("18.1.7"),
		test.WithHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "bitnamicharts/nginx",
				Repository: "oci://registry-1.docker.io",
				Version:    "18.1.7",
			},
		),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("false")},
			Description: "autoscaling.enabled",
			Name:        "autoscaling.enabled",
			Type:        "bool",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
			Description: "autoscaling.maxReplicas",
			Name:        "autoscaling.maxReplicas",
			Type:        "string",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("\"\"")},
			Description: "autoscaling.minReplicas",
			Name:        "autoscaling.minReplicas",
			Type:        "string",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
			Description: "containerSecurityContext.enabled",
			Name:        "containerSecurityContext.enabled",
			Type:        "bool",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("1")},
			Description: "replicaCount",
			Name:        "replicaCount",
			Type:        "int",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Default:     &apiextensionsv1.JSON{Raw: []byte("true")},
			Description: "podSecurityContext.enabled",
			Name:        "podSecurityContext.enabled",
			Type:        "bool",
		}),
	)
}

func CreateTestHookPluginDefinition(ctx context.Context, setup *test.TestSetup) *greenhousev1alpha1.PluginDefinition {
	return setup.CreatePluginDefinition(ctx, "test-hooks",
		test.WithVersion("0.1.0"),
		test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
			Name:       "./../../pkg/test/fixtures/testHook",
			Repository: "dummy",
			Version:    "0.1.0",
		}),
		test.AppendPluginOption(greenhousev1alpha1.PluginOption{
			Name:    "hook_enabled",
			Type:    "bool",
			Default: &apiextensionsv1.JSON{Raw: []byte("false")},
		}),
	)
}
