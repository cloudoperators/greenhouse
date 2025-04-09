// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/dexidp/dex/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&OAuth2Client{}, &OAuth2ClientList{})
}

// +kubebuilder:object:root=true

// OAuth2Client is an OAUTH2 client.
type OAuth2Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Client            `json:",inline"`
}

// +kubebuilder:object:generate=true

type Client storage.Client

// +kubebuilder:object:root=true

// OAuth2ClientList contains a list of OAuth2Clients.
type OAuth2ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAuth2Client `json:"items"`
}
