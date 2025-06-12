// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// WithVersion sets the version of a PluginDefinition
func WithVersion(version string) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Version = version
	}
}

// WithHelmChart sets the HelmChart of a PluginDefinition
func WithHelmChart(chart *greenhousev1alpha1.HelmChartReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = chart
	}
}

// WithoutHelmChart sets the HelmChart of a PluginDefinition to nil
func WithoutHelmChart() func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = nil
	}
}

// WithUIApplication sets the UI of a PluginDefinition
func WithUIApplication(ui *greenhousev1alpha1.UIApplicationReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.UIApplication = ui
	}
}

// AppendPluginOption sets the plugin option in plugin definition
func AppendPluginOption(option greenhousev1alpha1.PluginOption) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Options = append(pd.Spec.Options, option)
	}
}

// NewPluginDefinition returns a greenhousev1alpha1.PluginDefinition object. Opts can be used to set the desired state of the PluginDefinition.
func NewPluginDefinition(name string, opts ...func(*greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
	pd := &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Description: "TestPluginDefinition",
			Version:     "1.0.0",
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:       "./../../test/fixtures/myChart",
				Repository: "dummy",
				Version:    "1.0.0",
			},
		},
	}
	for _, o := range opts {
		o(pd)
	}
	return pd
}
