// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginSpec defines the desired state of Plugin
type PluginSpec struct {
	// PluginDefinition is the name of the PluginDefinition this instance is for.
	PluginDefinition string `json:"pluginDefinition"`

	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Disabled indicates that the plugin is administratively disabled.
	Disabled bool `json:"disabled"`

	// Values are the values for a PluginDefinition instance.
	OptionValues []PluginOptionValue `json:"optionValues,omitempty"`

	// ClusterName is the name of the cluster the plugin is deployed to. If not set, the plugin is deployed to the greenhouse cluster.
	ClusterName string `json:"clusterName,omitempty"`

	// ReleaseNamespace is the namespace in the remote cluster to which the backend is deployed.
	// Defaults to the Greenhouse managed namespace if not set.
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`
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
func (v *PluginOptionValue) ValueJSON() string {
	if v.Value == nil {
		return ""
	}
	return string(v.Value.Raw)
}

const (

	// ClusterAccessReadyCondition reflects if we can access the cluster a Plugin is to be deployed to.
	ClusterAccessReadyCondition ConditionType = "ClusterAccessReady"

	// HelmReconcileFailedCondition reflects the failed reconciliation of the corresponding helm release.
	HelmReconcileFailedCondition ConditionType = "HelmReconcileFailed"

	// HelmDriftDetectedCondition reflects the last time a drift between Release and Deployed Resources was detected.
	HelmDriftDetectedCondition ConditionType = "HelmDriftDetected"

	// WorkloadReadyCondition reflects the readiness of the workload resources belonging to the Plugin.
	WorkloadReadyCondition ConditionType = "WorkloadReady"

	// StatusUpToDateCondition reflects the failed reconciliation of the Plugin.
	StatusUpToDateCondition ConditionType = "StatusUpToDate"

	// Deprecated: NoHelmChartTestFailuresCondition reflects the status of the HelmChart tests.
	NoHelmChartTestFailuresCondition ConditionType = "NoHelmChartTestFailures"

	// HelmChartTestSucceededCondition reflects the status of the HelmChart tests.
	HelmChartTestSucceededCondition ConditionType = "HelmChartTestSucceeded"

	// PluginDefinitionNotFoundReason is set when the pluginDefinition is not found.
	PluginDefinitionNotFoundReason ConditionReason = "PluginDefinitionNotFound"

	// HelmUninstallFailedReason is set when the helm release could not be uninstalled.
	HelmUninstallFailedReason ConditionReason = "HelmUninstallFailed"
)

// PluginStatus defines the observed state of Plugin
type PluginStatus struct {
	// HelmReleaseStatus reflects the status of the latest HelmChart release.
	// This is only configured if the pluginDefinition is backed by HelmChart.
	HelmReleaseStatus *HelmReleaseStatus `json:"helmReleaseStatus,omitempty"`

	// Version contains the latest pluginDefinition version the config was last applied with successfully.
	Version string `json:"version,omitempty"`

	// HelmChart contains a reference the helm chart used for the deployed pluginDefinition version.
	HelmChart *HelmChartReference `json:"helmChart,omitempty"`

	// UIApplication contains a reference to the frontend that is used for the deployed pluginDefinition version.
	UIApplication *UIApplicationReference `json:"uiApplication,omitempty"`

	// Weight configures the order in which Plugins are shown in the Greenhouse UI.
	Weight *int32 `json:"weight,omitempty"`

	// Description provides additional details of the plugin.
	Description string `json:"description,omitempty"`

	// ExposedServices provides an overview of the Plugins services that are centrally exposed.
	// It maps the exposed URL to the service found in the manifest.
	ExposedServices map[string]Service `json:"exposedServices,omitempty"`

	// StatusConditions contain the different conditions that constitute the status of the Plugin.
	StatusConditions `json:"statusConditions,omitempty"`
}

// Service references a Kubernetes service of a Plugin.
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
//+kubebuilder:printcolumn:name="Plugin Definition",type=string,JSONPath=`.spec.pluginDefinition`
//+kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterName`
//+kubebuilder:printcolumn:name="Release Namespace",type=string,JSONPath=`.spec.releaseNamespace`
//+kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=`.spec.disabled`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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

func (o *Plugin) GetConditions() StatusConditions {
	return o.Status.StatusConditions
}

func (o *Plugin) SetCondition(condition Condition) {
	o.Status.StatusConditions.SetConditions(condition)
}
