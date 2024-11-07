// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Connector{}, &ConnectorList{})
}

// add some comment

// +kubebuilder:object:root=true

// Connector is an object that contains the metadata about connectors used to login to Dex.
// The struct is redefined here as the upstream does not provide the CRDs in a re-usable fashion and the json tag for the config field is incorrect.
// See https://github.com/dexidp/dex/blob/v2.36.0/storage/storage.go#L358-L376
type Connector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	DexConnector      `json:",inline"`
}

// +kubebuilder:object:generate=true

type DexConnector struct {
	// ID that will uniquely identify the connector object.
	ID string `json:"id,omitempty"`
	// The Type of the connector. E.g. 'oidc' or 'ldap'
	Type string `json:"type,omitempty"`
	// The Name of the connector that is used when displaying it to the end user.
	Name string `json:"name,omitempty"`
	// Config holds all the configuration information specific to the connector type. Since there
	// no generic struct we can use for this purpose, it is stored as a byte stream.
	Config []byte `json:"config,omitempty"`
}

// +kubebuilder:object:root=true

// ConnectorList contains a list of Connectors.
type ConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Connector `json:"items"`
}
