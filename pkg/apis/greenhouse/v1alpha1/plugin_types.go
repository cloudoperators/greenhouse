// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginSpec defines the desired state of Plugin
type PluginSpec struct {
	// Description provides additional details of the plugin.
	Description string `json:"description,omitempty"`

	// HelmChart specifies where the Helm Chart for this plugin can be found.
	HelmChart *HelmChartReference `json:"helmChart,omitempty"`

	// UIApplication specifies a reference to a UI application
	UIApplication *UIApplicationReference `json:"uiApplication,omitempty"`

	// RequiredValues is a list of values required to create an instance of this Plugin.
	Options []PluginOption `json:"options,omitempty"`

	// Version of this plugin
	Version string `json:"version"`

	// Weight configures the order in which Plugins are shown in the Greenhouse UI.
	// Defaults to alphabetical sorting if not provided or on conflict.
	Weight *int32 `json:"weight,omitempty"`
}

// PluginOptionType specifies the type of PluginOption.
// +kubebuilder:validation:Enum=string;secret;bool;int;list;map
type PluginOptionType string

const (
	// PluginOptionTypeString is a valid value for PluginOptionType.
	PluginOptionTypeString PluginOptionType = "string"
	// PluginOptionTypeSecret is a valid value for PluginOptionType.
	PluginOptionTypeSecret PluginOptionType = "secret"
	// PluginOptionTypeBool is a valid value for PluginOptionType.
	PluginOptionTypeBool PluginOptionType = "bool"
	// PluginOptionTypeInt is a valid value for PluginOptionType.
	PluginOptionTypeInt PluginOptionType = "int"
	// PluginOptionTypeList is a valid value for PluginOptionType.
	PluginOptionTypeList PluginOptionType = "list"
	// PluginOptionTypeMap is a valid value for PluginOptionType.
	PluginOptionTypeMap PluginOptionType = "map"
)

type PluginOption struct {
	// Name/Key of the config option.
	Name string `json:"name"`

	// Default provides a default value for the option
	// +optional
	Default *apiextensionsv1.JSON `json:"default,omitempty"`

	// Description provides a human-readable text for the value as shown in the UI.
	Description string `json:"description,omitempty"`

	// DisplayName provides a human-readable label for the configuration option
	DisplayName string `json:"displayName,omitempty"`

	// Required indicates that this config option is required
	Required bool `json:"required"`

	// Type of this configuration option.
	Type PluginOptionType `json:"type"`

	// Regex specifies a match rule for validating configuration options.
	Regex string `json:"regex,omitempty"`
}

// IsValid returns nil if the PluginOption is valid.
// An error is returned for unknown types or if type and value of the option do not match.
func (p *PluginOption) IsValid() error {
	if p.Default == nil {
		return nil
	}
	switch p.Type {
	case PluginOptionTypeBool:
		var b bool
		return json.Unmarshal(p.Default.Raw, &b)
	case PluginOptionTypeInt:
		var i int
		return json.Unmarshal(p.Default.Raw, &i)
	case PluginOptionTypeString, PluginOptionTypeSecret:
		var s string
		return json.Unmarshal(p.Default.Raw, &s)
	case PluginOptionTypeList:
		var l []any
		return json.Unmarshal(p.Default.Raw, &l)
	case PluginOptionTypeMap:
		var m map[string]any
		return json.Unmarshal(p.Default.Raw, &m)
	default:
		return fmt.Errorf("unknown type %s", p.Type)
	}
}

// IsValidValue returns nil if the given value is valid for this PluginOption.
// An error is returned if val does not match the type of the PluginOption.
func (p *PluginOption) IsValidValue(val *apiextensionsv1.JSON) error {
	var actVal any
	if err := json.Unmarshal(val.Raw, &actVal); err != nil {
		return err
	}
	switch p.Type {
	case PluginOptionTypeBool:
		if _, ok := actVal.(bool); !ok {
			return fmt.Errorf("option %s is a bool value, got %T", p.Name, actVal)
		}
	case PluginOptionTypeInt:
		switch actVal.(type) {
		case int, float64:
			// json.Decoder unmarshals numbers as float64, so we need to check for float64 as well
			// known Bug in k8s & helm: e.g. https://github.com/kubernetes-sigs/yaml/issues/45
			return nil
		default:
			return fmt.Errorf("option %s is an int value, got %T", p.Name, actVal)
		}
	case PluginOptionTypeString:
		if _, ok := actVal.(string); !ok {
			return fmt.Errorf("option %s is a string value, got %T", p.Name, actVal)
		}
	case PluginOptionTypeList:
		if _, ok := actVal.([]any); !ok {
			return fmt.Errorf("option %s is a list value, got %T", p.Name, actVal)
		}
	case PluginOptionTypeMap:
		if _, ok := actVal.(map[string]any); !ok {
			return fmt.Errorf("option %s is a map value, got %T", p.Name, actVal)
		}
	case PluginOptionTypeSecret:
		return fmt.Errorf("option %s is a secret value, that should be derived from a secret reference", p.Name)
	default:
		return fmt.Errorf("option %s has unknown type, got %T", p.Name, actVal)
	}
	return nil
}

// GetDefault returns the default value for this option.
func (p *PluginOption) DefaultValue() (any, error) {
	if p == nil {
		return nil, nil
	}
	switch p.Type {
	case PluginOptionTypeBool:
		var b bool
		if err := json.Unmarshal(p.Default.Raw, &b); err != nil {
			return nil, err
		}
		return b, nil
	case PluginOptionTypeInt:
		var i int
		if err := json.Unmarshal(p.Default.Raw, &i); err != nil {
			return nil, err
		}
		return i, nil
	case PluginOptionTypeSecret, PluginOptionTypeString:
		var s string
		if err := json.Unmarshal(p.Default.Raw, &s); err != nil {
			return nil, err
		}
		return s, nil
	case PluginOptionTypeList:
		var l []any
		if err := json.Unmarshal(p.Default.Raw, &l); err != nil {
			return nil, err
		}
		return l, nil
	default:
		return nil, fmt.Errorf("unknown type %s", p.Type)
	}
}

// PluginStatus defines the observed state of Plugin
type PluginStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Plugin is the Schema for the plugins API
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginSpec   `json:"spec,omitempty"`
	Status PluginStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PluginList contains a list of Plugin
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Plugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Plugin{}, &PluginList{})
}
