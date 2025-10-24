// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

// PluginSpec defines the desired state of Plugin
type PluginSpec struct {
	// PluginDefinition is the name of the PluginDefinition this instance is for.
	//
	// Deprecated: Use PluginDefinitionRef instead. Future releases of greenhouse will remove this field.
	PluginDefinition string `json:"pluginDefinition"`

	// PluginDefinitionRef is the reference to the (Cluster-)PluginDefinition.
	PluginDefinitionRef PluginDefinitionReference `json:"pluginDefinitionRef"`

	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Values are the values for a PluginDefinition instance.
	OptionValues []PluginOptionValue `json:"optionValues,omitempty"`

	// ClusterName is the name of the cluster the plugin is deployed to. If not set, the plugin is deployed to the greenhouse cluster.
	ClusterName string `json:"clusterName,omitempty"`

	// ReleaseNamespace is the namespace in the remote cluster to which the backend is deployed.
	// Defaults to the Greenhouse managed namespace if not set.
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`

	// ReleaseName is the name of the helm release in the remote cluster to which the backend is deployed.
	// If the Plugin was already deployed, the Plugin's name is used as the release name.
	// If this Plugin is newly created, the releaseName is defaulted to the PluginDefinitions HelmChart name.
	// +Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ReleaseName is immutable"
	// +kubebuilder:validation:MaxLength=53
	ReleaseName string `json:"releaseName,omitempty"`

	// DeletionPolicy defines how Helm Releases created by a Plugin are handled upon deletion of the Plugin.
	// Supported values are "Delete" and "Retain". If not set, defaults to "Delete".
	// +Optional
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// WaitFor defines other Plugins to wait for before installing this Plugin.
	WaitFor []WaitForItem `json:"waitFor,omitempty"`

	// IgnoreDifferences defines paths to ignore when detecting drift between desired and actual state.
	// +Optional
	IgnoreDifferences []IgnoreDifference `json:"ignoreDifferences,omitempty"`
}

// PluginOptionValue is the value for a PluginOption.
type PluginOptionValue struct {
	// Name of the values.
	Name string `json:"name"`
	// Value is the actual value in plain text.
	Value *apiextensionsv1.JSON `json:"value,omitempty"`
	// ValueFrom references a potentially confidential value in another source.
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
	// Template is a Go string template that will be dynamically resolved for cluster-specific values.
	// Only PluginOptionValues declared as template will be templated by the PluginController for Flux.
	Template *string `json:"template,omitempty"`
	// CelExpression is a CEL expression that will be evaluated to resolve the value.
	// CEL expressions have access to global.greenhouse values for dynamic configuration.
	CelExpression *string `json:"celExpression,omitempty"`
}

// IgnoreDifference defines a set of paths to ignore for matching resources.
type IgnoreDifference struct {
	// Group matches the APIVersion group of the resources to ignore.
	// +Optional
	Group string `json:"group,omitempty"`
	// Version matches the APIVersion version of the resources to ignore.
	// +Optional
	Version string `json:"version,omitempty"`
	// Kind matches the Kind of the resources to ignore.
	// +Optional
	Kind string `json:"kind,omitempty"`
	// Name matches the name of the resources to ignore.
	// +Optional
	Name string `json:"name,omitempty"`
	// Paths is a list of JSON paths to ignore when detecting drifts.
	// +kubebuilder:validation:Required
	Paths []string `json:"paths,omitempty"`
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
	ClusterAccessReadyCondition greenhousemetav1alpha1.ConditionType = "ClusterAccessReady"

	// HelmReconcileFailedCondition reflects the failed reconciliation of the corresponding helm release.
	HelmReconcileFailedCondition greenhousemetav1alpha1.ConditionType = "HelmReconcileFailed"

	// HelmDriftDetectedCondition reflects the last time a drift between Release and Deployed Resources was detected.
	HelmDriftDetectedCondition greenhousemetav1alpha1.ConditionType = "HelmDriftDetected"

	// WorkloadReadyCondition reflects the readiness of the workload resources belonging to the Plugin.
	WorkloadReadyCondition greenhousemetav1alpha1.ConditionType = "WorkloadReady"

	// StatusUpToDateCondition reflects the failed reconciliation of the Plugin.
	StatusUpToDateCondition greenhousemetav1alpha1.ConditionType = "StatusUpToDate"

	// HelmChartTestSucceededCondition reflects the status of the HelmChart tests.
	HelmChartTestSucceededCondition greenhousemetav1alpha1.ConditionType = "HelmChartTestSucceeded"

	// WaitingForDependenciesCondition reflects if HelmRelease is waiting for other releases to be ready.
	WaitingForDependenciesCondition greenhousemetav1alpha1.ConditionType = "WaitingForDependencies"

	// PluginDefinitionNotFoundReason is set when the pluginDefinition is not found.
	PluginDefinitionNotFoundReason greenhousemetav1alpha1.ConditionReason = "PluginDefinitionNotFound"

	// HelmUninstallFailedReason is set when the helm release could not be uninstalled.
	HelmUninstallFailedReason greenhousemetav1alpha1.ConditionReason = "HelmUninstallFailed"
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
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`

	// LastReconciledAt contains the value when the reconcile was last triggered via annotation.
	// +Optional
	LastReconciledAt string `json:"lastReconciledAt,omitempty"`
}

// ServiceType defines the type of exposed service.
// +kubebuilder:validation:Enum=service;ingress
type ServiceType string

const (
	// ServiceTypeService indicates the service is exposed via service-proxy.
	ServiceTypeService ServiceType = "service"
	// ServiceTypeIngress indicates the service is exposed via ingress.
	ServiceTypeIngress ServiceType = "ingress"
)

// Service references a Kubernetes service of a Plugin.
type Service struct {
	// Namespace is the namespace of the service in the target cluster.
	Namespace string `json:"namespace"`
	// Name is the name of the service in the target cluster.
	Name string `json:"name"`
	// Port is the port of the service. Zero for ingresses where port is not applicable.
	Port int32 `json:"port,omitempty"`
	// Protocol is the protocol of the service.
	Protocol *string `json:"protocol,omitempty"`
	// Type is the type of exposed service.
	// +kubebuilder:default="service"
	Type ServiceType `json:"type"`
}

// HelmReleaseStatus reflects the status of a Helm release.
type HelmReleaseStatus struct {
	// Status is the status of a HelmChart release.
	Status string `json:"status"`
	// FirstDeployed is the timestamp of the first deployment of the release.
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty"`
	// LastDeployed is the timestamp of the last deployment of the release.
	LastDeployed metav1.Time `json:"lastDeployed,omitempty"`
	// PluginOptionChecksum is the checksum of plugin option values.
	PluginOptionChecksum string `json:"pluginOptionChecksum,omitempty"`
	// Diff contains the difference between the deployed helm chart and the helm chart in the last reconciliation
	Diff string `json:"diff,omitempty"`
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
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec.releaseName) || has(self.spec.releaseName)", message="ReleaseName is required once set"
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

func (o *Plugin) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return o.Status.StatusConditions
}

func (o *Plugin) SetCondition(condition greenhousemetav1alpha1.Condition) {
	o.Status.SetConditions(condition)
}

func (o *Plugin) UpdateLastReconciledAtStatus(value string) {
	o.Status.LastReconciledAt = value
}

func (o *Plugin) GetReleaseName() string {
	if o.Spec.ReleaseName != "" {
		return o.Spec.ReleaseName
	}
	return o.Name
}
