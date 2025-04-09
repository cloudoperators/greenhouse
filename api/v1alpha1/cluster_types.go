// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
)

// ClusterSpec defines the desired state of the Cluster.
type ClusterSpec struct {
	// AccessMode configures how the cluster is accessed from the Greenhouse operator.
	AccessMode ClusterAccessMode `json:"accessMode"`

	// KubeConfig contains specific values for `KubeConfig` for the cluster.
	KubeConfig ClusterKubeConfig `json:"kubeConfig,omitempty"`
}

// ClusterAccessMode configures the access mode to the customer cluster.
// +kubebuilder:validation:Enum=direct
type ClusterAccessMode string

// ClusterKubeConfig configures kube config values.
type ClusterKubeConfig struct {
	// MaxTokenValidity specifies the maximum duration for which a token remains valid in hours.
	// +kubebuilder:default:=72
	// +kubebuilder:validation:Minimum=24
	// +kubebuilder:validation:Maximum=72
	MaxTokenValidity int32 `json:"maxTokenValidity,omitempty"`
}

const (
	// ClusterAccessModeDirect configures direct access to the cluster.
	ClusterAccessModeDirect ClusterAccessMode = "direct"

	// AllNodesReady reflects the readiness status of all nodes of a cluster.
	AllNodesReady greenhouseapis.ConditionType = "AllNodesReady"

	// KubeConfigValid reflects the validity of the kubeconfig of a cluster.
	KubeConfigValid greenhouseapis.ConditionType = "KubeConfigValid"

	// MaxTokenValidity contains maximum bearer token validity duration. It is also default value.
	MaxTokenValidity = 72

	// MinTokenValidity contains maximum bearer token validity duration.
	MinTokenValidity = 24
)

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// KubernetesVersion reflects the detected Kubernetes version of the cluster.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// BearerTokenExpirationTimestamp reflects the expiration timestamp of the bearer token used to access the cluster.
	BearerTokenExpirationTimestamp metav1.Time `json:"bearerTokenExpirationTimestamp,omitempty"`
	// StatusConditions contain the different conditions that constitute the status of the Cluster.
	greenhouseapis.StatusConditions `json:"statusConditions,omitempty"`
	// Nodes provides a map of cluster node names to node statuses
	Nodes map[string]NodeStatus `json:"nodes,omitempty"`
}

// ClusterConditionType is a valid condition of a cluster.
type ClusterConditionType string

type NodeStatus struct {
	// We mirror the node conditions here for faster reference
	greenhouseapis.StatusConditions `json:"statusConditions,omitempty"`
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

func (c *Cluster) GetConditions() greenhouseapis.StatusConditions {
	return c.Status.StatusConditions
}

func (c *Cluster) SetCondition(condition greenhouseapis.Condition) {
	c.Status.StatusConditions.SetConditions(condition)
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

func (c *Cluster) SetDefaultTokenValidityIfNeeded() {
	if c.Spec.KubeConfig.MaxTokenValidity != 0 {
		return
	}

	c.Spec.KubeConfig.MaxTokenValidity = MaxTokenValidity
}
