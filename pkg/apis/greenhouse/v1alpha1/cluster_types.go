// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Copyright 2023.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpec defines the desired state of the Cluster.
type ClusterSpec struct {
	// AccessMode configures how the cluster is accessed from the Greenhouse operator.
	AccessMode ClusterAccessMode `json:"accessMode"`
}

// ClusterAccessMode configures the access mode to the customer cluster.
// +kubebuilder:validation:Enum=direct;headscale
type ClusterAccessMode string

const (
	// ClusterAccessModeDirect configures direct access to the cluster.
	ClusterAccessModeDirect ClusterAccessMode = "direct"

	// ClusterAccessModeHeadscale configures headscale-based access to the cluster.
	ClusterAccessModeHeadscale ClusterAccessMode = "headscale"

	// AllNodesReady reflects the readiness status of all nodes of a cluster.
	AllNodesReady ConditionType = "AllNodesReady"

	// KubeConfigValid reflects the validity of the kubeconfig of a cluster.
	KubeConfigValid ConditionType = "KubeConfigValid"
)

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// KubernetesVersion reflects the detected Kubernetes version of the cluster.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// BearerTokenExpirationTimestamp reflects the expiration timestamp of the bearer token used to access the cluster.
	BearerTokenExpirationTimestamp metav1.Time `json:"bearerTokenExpirationTimestamp,omitempty"`
	// HeadScaleStatus contains the current status of the headscale client.
	HeadScaleStatus *HeadScaleMachineStatus `json:"headScaleStatus,omitempty"`
	// StatusConditions contain the different conditions that constitute the status of the Cluster.
	StatusConditions `json:"statusConditions,omitempty"`
	// Nodes provides a map of cluster node names to node statuses
	Nodes map[string]NodeStatus `json:"nodes,omitempty"`
}

// HeadScaleMachineStatus is the status of a Headscale machine.
type HeadScaleMachineStatus struct {
	ID          uint64      `json:"id,omitempty"`
	IPAddresses []string    `json:"ipAddresses,omitempty"`
	Name        string      `json:"name,omitempty"`
	Expiry      metav1.Time `json:"expiry,omitempty"`
	CreatedAt   metav1.Time `json:"createdAt,omitempty"`
	ForcedTags  []string    `json:"forcedTags,omitempty"`
	PreAuthKey  *PreAuthKey `json:"preAuthKey,omitempty"`
	Online      bool        `json:"online,omitempty"`
}

// PreAuthKey reflects the status of the pre-authentication key used by the Headscale machine.
type PreAuthKey struct {
	ID         string      `json:"id,omitempty"`
	User       string      `json:"user,omitempty"`
	Reusable   bool        `json:"reusable,omitempty"`
	Ephemeral  bool        `json:"ephemeral,omitempty"`
	Used       bool        `json:"used,omitempty"`
	CreatedAt  metav1.Time `json:"createdAt,omitempty"`
	Expiration metav1.Time `json:"expiration,omitempty"`
}

// ClusterConditionType is a valid condition of a cluster.
type ClusterConditionType string

const (
	// HeadscaleReady reflects the readiness status of the headscale access of a cluster.
	HeadscaleReady ConditionType = "HeadscaleReady"
)

type NodeStatus struct {
	// We mirror the node conditions here for faster reference
	StatusConditions `json:"statusConditions,omitempty"`
	// Fast track to the node ready condition.
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="AccessMode",type="string",JSONPath=".spec.accessMode"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// GetSecretName returns the Kubernetes secret containing sensitive data for this cluster.
// The secret is for internal usage only and its content must not be exposed to the user.
func (c *Cluster) GetSecretName() string {
	return c.GetName()
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
