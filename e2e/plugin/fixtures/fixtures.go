// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

func PreparePluginDefinition(name, namespace string, opts ...func(definition *greenhousev1alpha1.ClusterPluginDefinition)) *greenhousev1alpha1.ClusterPluginDefinition {
	pd := &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Description: "TestPluginDefinition",
			Version:     "1.0.0",
			HelmChart:   &greenhousev1alpha1.HelmChartReference{}, // helm chart values are override later
		},
	}
	for _, o := range opts {
		o(pd)
	}

	return pd
}

func PrepareCertManagerPluginDefinition(namespace string) *greenhousev1alpha1.ClusterPluginDefinition {
	return PreparePluginDefinition("cert-manager-v1.17.0", namespace,
		test.WithVersion("v1.17.0"),
		test.WithHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "cert-manager",
				Repository: "https://charts.jetstack.io",
				Version:    "v1.17.0",
			},
		),
	)
}

func PreparePodInfoPluginDefinition(namespace, version string) *greenhousev1alpha1.ClusterPluginDefinition {
	return PreparePluginDefinition("podinfo", namespace,
		test.WithVersion(version),
		test.WithHelmChart(
			&greenhousev1alpha1.HelmChartReference{
				Name:       "podinfo",
				Repository: "oci://ghcr.io/stefanprodan/charts",
				Version:    version,
			},
		),
	)
}

func PrepareUIPluginDefinition(namespace string) *greenhousev1alpha1.ClusterPluginDefinition {
	return PreparePluginDefinition("ui-only", namespace,
		test.WithVersion("1.0.0"),
		test.WithoutHelmChart(),
		test.WithUIApplication(&greenhousev1alpha1.UIApplicationReference{
			Name:    "test-ui-app",
			Version: "0.0.1",
		}),
	)
}

func PreparePlugin(name, namespace string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	plugin := &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         name,
			Namespace:    namespace,
			GenerateName: name + "-gen",
		},
	}
	for _, o := range opts {
		o(plugin)
	}
	return plugin
}
