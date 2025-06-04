// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// ClusterKubeconfigSpec stores the kubeconfig data for the cluster
// The idea is to use kubeconfig data locally with minimum effort (with local tools or plain kubectl):
// kubectl get cluster-kubeconfig $NAME -o yaml | yq -y .spec.kubeconfig
type ClusterKubeconfigSpec struct {
	Kubeconfig ClusterKubeconfigData `json:"kubeconfig,omitempty"`
}

// ClusterKubeconfigData stores the kubeconfig data ready to use kubectl or other local tooling
// It is a simplified version of clientcmdapi.Config: https://pkg.go.dev/k8s.io/client-go/tools/clientcmd/api#Config
type ClusterKubeconfigData struct {
	Kind           string                          `json:"kind,omitempty"`
	APIVersion     string                          `json:"apiVersion,omitempty"`
	Clusters       []ClusterKubeconfigClusterItem  `json:"clusters,omitempty"`
	AuthInfo       []ClusterKubeconfigAuthInfoItem `json:"users,omitempty"`
	Contexts       []ClusterKubeconfigContextItem  `json:"contexts,omitempty"`
	CurrentContext string                          `json:"current-context,omitempty"`
	Preferences    ClusterKubeconfigPreferences    `json:"preferences,omitempty"`
}

type ClusterKubeconfigClusterItem struct {
	Name    string                   `json:"name"`
	Cluster ClusterKubeconfigCluster `json:"cluster"`
}

type ClusterKubeconfigCluster struct {
	Server                   string `json:"server,omitempty"`
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
}

type ClusterKubeconfigAuthInfoItem struct {
	Name     string                    `json:"name"`
	AuthInfo ClusterKubeconfigAuthInfo `json:"user,omitempty"`
}

type ClusterKubeconfigAuthInfo struct {
	AuthProvider          clientcmdapi.AuthProviderConfig `json:"auth-provider,omitempty"`
	ClientCertificateData []byte                          `json:"client-certificate-data,omitempty"`
	ClientKeyData         []byte                          `json:"client-key-data,omitempty"`
}

type ClusterKubeconfigContextItem struct {
	Name    string                   `json:"name"`
	Context ClusterKubeconfigContext `json:"context,omitempty"`
}

type ClusterKubeconfigContext struct {
	Cluster   string `json:"cluster"`
	AuthInfo  string `json:"user"`
	Namespace string `json:"namespace,omitempty"`
}

type ClusterKubeconfigPreferences struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=cluster-kubeconfig;cluster-kubeconfigs

// ClusterKubeconfig is the Schema for the clusterkubeconfigs API
// ObjectMeta.OwnerReferences is used to link the ClusterKubeconfig to the Cluster
// ObjectMeta.Generation is used to detect changes in the ClusterKubeconfig and sync local kubeconfig files
// ObjectMeta.Name is designed to be the same with the Cluster name
type ClusterKubeconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterKubeconfigSpec   `json:"spec,omitempty"`
	Status ClusterKubeconfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type ClusterKubeconfigStatus struct {
	Conditions greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`
}

const (
	KubeconfigCreatedCondition         greenhousemetav1alpha1.ConditionType = "Created"
	KubeconfigReconcileFailedCondition greenhousemetav1alpha1.ConditionType = "ReconcileFailed"
	KubeconfigReadyCondition           greenhousemetav1alpha1.ConditionType = "Ready"
)

//+kubebuilder:object:root=true

// ClusterKubeconfigList contains a list of ClusterKubeconfig
type ClusterKubeconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterKubeconfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterKubeconfig{}, &ClusterKubeconfigList{})
}
