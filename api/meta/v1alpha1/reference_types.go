// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// PluginOptionValue is the value for a PluginOption.
type PluginOptionValue struct {
	// Name of the values.
	Name string `json:"name"`
	// Value is the actual value in plain text.
	Value *apiextensionsv1.JSON `json:"value,omitempty"`
	// ValueFrom references a potentially confidential value in another source.
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
}

// ValueJSON returns the value as JSON.
func (v *PluginOptionValue) ValueJSON() string {
	if v.Value == nil {
		return ""
	}
	return string(v.Value.Raw)
}

// ValueFromSource is a valid source for a value.
type ValueFromSource struct {
	// Secret references the secret containing the value.
	Secret *SecretKeyReference `json:"secret,omitempty"`
}

// SecretKeyReference specifies the secret and key containing the value.
type SecretKeyReference struct {
	// Name of the secret in the same namespace.
	Name string `json:"name"`
	// Key in the secret to select the value from.
	Key string `json:"key"`
}
