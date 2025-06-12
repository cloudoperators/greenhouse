// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	. "github.com/onsi/ginkgo/v2"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// WithPluginDefinition sets the PluginDefinition of a Plugin
func WithPluginDefinition(pluginDefinition string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.PluginDefinition = pluginDefinition
	}
}

// WithReleaseNamespace sets the ReleaseNamespace of a Plugin
func WithReleaseNamespace(releaseNamespace string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseNamespace = releaseNamespace
	}
}

// WithReleaseName sets the ReleaseName of a Plugin
func WithReleaseName(releaseName string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseName = releaseName
	}
}

// WithCluster sets the Cluster for a Plugin
func WithCluster(cluster string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ClusterName = cluster
	}
}

// WithPresetLabelValue sets the value of the greenhouseapis.LabelKeyPluginPreset label on a Plugin
// This label is used to indicate that the Plugin is managed by a PluginPreset.
func WithPresetLabelValue(value string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		if p.Labels == nil {
			p.Labels = make(map[string]string, 1)
		}
		p.Labels[greenhouseapis.LabelKeyPluginPreset] = value
	}
}

// WithPluginOptionValue sets the value of a PluginOptionValue
func WithPluginOptionValue(name string, value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		if value != nil && valueFrom != nil {
			Fail("value and valueFrom are mutually exclusive")
		}
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = value
				v.ValueFrom = valueFrom
				p.Spec.OptionValues[i] = v
				return
			}
		}
		p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:      name,
			Value:     value,
			ValueFrom: valueFrom,
		})
	}
}

// SetOptionValueForPlugin sets the value of a PluginOptionValue in plugin
func SetOptionValueForPlugin(plugin *greenhousev1alpha1.Plugin, key, value string) {
	for i, keyValue := range plugin.Spec.OptionValues {
		if keyValue.Name == key {
			plugin.Spec.OptionValues[i].Value.Raw = []byte(value)
			return
		}
	}
	plugin.Spec.OptionValues = append(plugin.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
		Name:  key,
		Value: &apiextensionsv1.JSON{Raw: []byte(value)},
	})
}

// NewPlugin returns a greenhousev1alpha1.Plugin object. Opts can be used to set the desired state of the Plugin.
func NewPlugin(name, namespace string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, o := range opts {
		o(plugin)
	}
	return plugin
}
