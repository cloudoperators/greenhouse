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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginConfigSpec defines the desired state of PluginConfig
type PluginConfigSpec struct {
	// Plugin is the name of the plugin this instance is for.
	Plugin string `json:"plugin"`

	// DisplayName is an optional name for the plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a Plugin in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Disabled indicates that the plugin config is administratively disabled.
	Disabled bool `json:"disabled"`

	// Values are the values for a plugin instance.
	OptionValues []PluginOptionValue `json:"optionValues,omitempty"`

	// ClusterName is the name of the cluster the pluginConfig is deployed to. If not set, the pluginConfig is deployed to the greenhouse cluster.
	ClusterName string `json:"clusterName,omitempty"`
}

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
func (v *PluginOptionValue) ValueJSON() (string, error) {
	if v.Value == nil {
		return "", nil
	}
	return string(v.Value.Raw), nil
}

const (

	// ClusterAccessReadyCondition reflects if we can access the cluster a PluginConfig is to be deployed to.
	ClusterAccessReadyCondition ConditionType = "ClusterAccessReady"

	// HelmReconcileFailedCondition reflects the failed reconciliation of the corresponding helm release.
	HelmReconcileFailedCondition ConditionType = "HelmReconcileFailed"

	// HelmDriftDetectedCondition reflects the last time a drift between Release and Deployed Resources was detected.
	HelmDriftDetectedCondition ConditionType = "HelmDriftDetected"

	// StatusUpToDateCondition reflects the failed reconciliation of the PluginConfig.
	StatusUpToDateCondition ConditionType = "StatusUpToDate"

	// PluginNotFoundReason is set when the plugin is not found.
	PluginNotFoundReason ConditionReason = "PluginNotFound"

	// HelmUninstallFailedReason is set when the helm release could not be uninstalled.
	HelmUninstallFailedReason ConditionReason = "HelmUninstallFailed"
)

// PluginConfigStatus defines the observed state of PluginConfig
type PluginConfigStatus struct {
	// HelmReleaseStatus reflects the status of the latest HelmChart release.
	// This is only configured if the plugin is backed by HelmChart.
	HelmReleaseStatus *HelmReleaseStatus `json:"helmReleaseStatus,omitempty"`

	// Version contains the latest plugin version the config was last applied with successfully.
	Version string `json:"version,omitempty"`

	// HelmChart contains a reference the helm chart used for the deployed plugin version.
	HelmChart *HelmChartReference `json:"helmChart,omitempty"`

	// UIApplication contains a reference to the frontend that is used for the deployed plugin version.
	UIApplication *UIApplicationReference `json:"uiApplication,omitempty"`

	// Weight configures the order in which Plugins are shown in the Greenhouse UI.
	Weight *int32 `json:"weight,omitempty"`

	// Description provides additional details of the plugin.
	Description string `json:"description,omitempty"`

	// ExposedServices provides an overview of the PluginConfigs services that are centrally exposed.
	// It maps the exposed URL to the service found in the manifest.
	ExposedServices map[string]Service `json:"exposedServices,omitempty"`

	// StatusConditions contain the different conditions that constitute the status of the PluginConfig.
	StatusConditions `json:"statusConditions,omitempty"`
}

// Service references a Kubernetes service of a PluginConfig.
type Service struct {
	// Namespace is the namespace of the service in the target cluster.
	Namespace string `json:"namespace"`
	// Name is the name of the service in the target cluster.
	Name string `json:"name"`
	// Port is the port of the service.
	Port int32 `json:"port"`
	// Protocol is the protocol of the service.
	Protocol *string `json:"protocol,omitempty"`
}

// HelmReleaseStatus reflects the status of a Helm release.
type HelmReleaseStatus struct {
	// Status is the status of a HelmChart release.
	Status string `json:"status"`
	// FirstDeployed is the timestamp of the first deployment of the release.
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty"`
	// LastDeployed is the timestamp of the last deployment of the release.
	LastDeployed metav1.Time `json:"lastDeployed,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Display name",type=string,JSONPath=`.spec.displayName`
//+kubebuilder:printcolumn:name="Plugin",type=string,JSONPath=`.spec.plugin`
//+kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterName`
//+kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=`.spec.disabled`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PluginConfig is the Schema for the pluginconfigs API
type PluginConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginConfigSpec   `json:"spec,omitempty"`
	Status PluginConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PluginConfigList contains a list of PluginConfig
type PluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PluginConfig{}, &PluginConfigList{})
}
