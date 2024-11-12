// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"bytes"
	"io"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
)

func Log(args ...any) {
	args[0] = "===== 🤖 " + args[0].(string) //nolint:errcheck
	klog.InfoDepth(1, args...)
}

func Logf(format string, args ...any) {
	klog.InfofDepth(1, "===== 🤖 "+format, args...)
}

func LogErr(format string, args ...any) {
	klog.InfofDepth(1, "===== 😵 "+format, args...)
}

// FromYamlToK8sObject - Converts a YAML document to a Kubernetes object
// if yaml contains multiple documents, then corresponding kubernetes objects should be provided
func FromYamlToK8sObject(doc string, resources ...any) error {
	yamlBytes := []byte(doc)
	dec := kyaml.NewDocumentDecoder(io.NopCloser(bytes.NewReader(yamlBytes)))
	buffer := make([]byte, len(yamlBytes))

	for _, resource := range resources {
		n, err := dec.Read(buffer)
		if err != nil {
			return err
		}
		err = kyaml.Unmarshal(buffer[:n], resource)
		if err != nil {
			return err
		}
	}
	return nil
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

// PreparePlugin prepares structure of the plugin
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

// WithPluginDefinition sets the PluginDefinition of a Plugin
func WithPluginDefinition(pluginDefinition string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.PluginDefinition = pluginDefinition
	}
}

// WithCluster sets the Cluster for a Plugin
func WithCluster(cluster string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ClusterName = cluster
	}
}

// WithReleaseNamespace sets the ReleaseNamespace of a Plugin
func WithReleaseNamespace(releaseNamespace string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseNamespace = releaseNamespace
	}
}

// WithPluginOptionValue sets the value of a PluginOptionValue
func WithPluginOptionValue(name string, value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = value
				v.ValueFrom = valueFrom
				p.Spec.OptionValues[i] = v
				return
			}
			p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
				Name:      name,
				Value:     value,
				ValueFrom: valueFrom,
			})
		}
	}
}

// PreparePluginDefinition prepares structure of the plugin definition
func PreparePluginDefinition(name, namespace string, opts ...func(*greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
	pd := &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			HelmChart: &greenhousev1alpha1.HelmChartReference{}, // helm chart values are override later
		},
	}
	for _, o := range opts {
		o(pd)
	}

	return pd
}

// WithVersion sets the version of a PluginDefinition
func WithVersion(version string) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Version = version
	}
}

// WithDescription sets the description of a PluginDefinition
func WithDescription(description string) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Description = description
	}
}

// WithHelmChart sets the HelmChart of a PluginDefinition
func WithHelmChart(chart *greenhousev1alpha1.HelmChartReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = chart
	}
}

// AppendPluginOption sets the plugin option in plugin definition
func AppendPluginOption(option greenhousev1alpha1.PluginOption) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Options = append(pd.Spec.Options, option)
	}
}
