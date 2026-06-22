// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
)

const (
	// PluginPresetKind is the kind of the PluginPreset resource
	PluginPresetKind = "PluginPreset"

	// PluginReconcileFailed is set when Plugin creation or update failed.
	PluginReconcileFailed greenhousemetav1alpha1.ConditionReason = "PluginReconcileFailed"

	// PluginDefinitionNotFound is set when the PluginDefinition referenced by the PluginPreset does not exist.
	PluginDefinitionNotFound greenhousemetav1alpha1.ConditionReason = "PluginDefinitionNotFound"
)

// PluginPresetSpec defines the desired state of PluginPreset
type PluginPresetSpec struct {

	// PluginSpec is the spec of the plugin to be deployed by the PluginPreset.
	Plugin PluginPresetPluginSpec `json:"plugin"`

	// ClusterSelector is a label selector to select the clusters the plugin bundle should be deployed to.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`

	// ClusterOptionOverrides define plugin option values to override by the PluginPreset
	// +Optional
	ClusterOptionOverrides []ClusterOptionOverride `json:"clusterOptionOverrides,omitempty"`

	// WaitFor defines other Plugins to wait for before creating the Plugin.
	WaitFor []WaitForItem `json:"waitFor,omitempty"`

	// DeletionPolicy defines how Plugins owned by a PluginPreset are handled on deletion of the PluginPreset.
	// Supported values are "Delete" and "Retain". If not set, defaults to "Delete".
	// +Optional
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// PluginPresetPluginSpec defines the desired state of Plugin
type PluginPresetPluginSpec struct {
	// PluginDefinitionRef is the reference to the (Cluster-)PluginDefinition.
	PluginDefinitionRef PluginDefinitionReference `json:"pluginDefinitionRef"`

	// DisplayName is an optional name for the Plugin to be displayed in the Greenhouse UI.
	// This is especially helpful to distinguish multiple instances of a PluginDefinition in the same context.
	// Defaults to a normalized version of metadata.name.
	DisplayName string `json:"displayName,omitempty"`

	// Values are the values for a PluginDefinition instance.
	OptionValues []PluginPresetPluginOptionValue `json:"optionValues,omitempty"`

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

	// IgnoreDifferences defines paths to ignore when detecting drift between desired and actual state.
	// +Optional
	IgnoreDifferences []IgnoreDifference `json:"ignoreDifferences,omitempty"`
}

// PluginPresetPluginOptionValue is the value for a PluginOption.
type PluginPresetPluginOptionValue struct {
	// Name of the values.
	Name string `json:"name"`
	// Value is the actual value in plain text.
	Value *apiextensionsv1.JSON `json:"value,omitempty"`
	// ValueFrom references value in another source.
	ValueFrom *PluginPresetPluginValueFromSource `json:"valueFrom,omitempty"`
	// Expression is a YAML string with ${...} placeholders that will be evaluated as CEL expressions.
	Expression *string `json:"expression,omitempty"`
}

// PluginPresetPluginValueFromSource defines how to extract dynamic values
// only one of secret or ref can be set
// +kubebuilder:validation:XValidation:rule="!(has(self.secret) && has(self.ref))",message="both secret and ref cannot be set"
// +kubebuilder:validation:XValidation:rule="has(self.secret) || has(self.ref)",message="one of secret or ref must be set"
type PluginPresetPluginValueFromSource struct {
	// Secret references the v1.Secret containing the value that needs to be extracted
	Secret *SecretKeyReference `json:"secret,omitempty"`
	// Ref references values defined in another resource (Plugin, PluginPreset)
	Ref *ExternalValueSource `json:"ref,omitempty"`
}

// ExternalValueSource defines how to extract values from external resources
// +kubebuilder:validation:ExactlyOneOf=name;selector
type ExternalValueSource struct {
	// Kind is the resource kind to target
	// if not set, defaults to the same kind as the referencing resource (Plugin or PluginPreset)
	// +Optional
	// +kubebuilder:validation:Enum=Plugin;PluginPreset
	Kind string `json:"kind,omitempty"`

	// Name is the name of the resource to target
	// this field is mutually exclusive with LabelSelector
	// +Optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Selector selects the resources to target based on labels
	// this field is mutually exclusive with Name
	// +Optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Expression is a CEL expression to extract the value from the referenced resource
	// +kubebuilder:validation:Required
	Expression string `json:"expression"`
}

// ClusterOptionOverride defines which plugin option should be override in which cluster
// +Optional
type ClusterOptionOverride struct {
	ClusterName string                          `json:"clusterName"`
	Overrides   []PluginPresetPluginOptionValue `json:"overrides"`
}

const (
	// PluginSkippedCondition is set when the pluginPreset encounters a non-managed plugin.
	PluginSkippedCondition greenhousemetav1alpha1.ConditionType = "PluginSkipped"
	// PluginFailedCondition is set when the pluginPreset encounters a failure during the reconciliation of a plugin.
	PluginFailedCondition greenhousemetav1alpha1.ConditionType = "PluginFailed"
	// AllPluginsReadyCondition is set when all Plugins managed by the PluginPreset are created and ready.
	AllPluginsReadyCondition greenhousemetav1alpha1.ConditionType = "AllPluginsReady"
	// PluginDefinitionNotFoundCondition is set when the referenced PluginDefinition or ClusterPluginDefinition cannot be resolved.
	PluginDefinitionNotFoundCondition greenhousemetav1alpha1.ConditionType = "PluginDefinitionNotFound"
)

// PluginPresetStatus defines the observed state of PluginPreset
type PluginPresetStatus struct {
	// StatusConditions contain the different conditions that constitute the status of the PluginPreset.
	greenhousemetav1alpha1.StatusConditions `json:"statusConditions,omitempty"`

	// PluginStatuses contains statuses of Plugins managed by the PluginPreset.
	PluginStatuses []ManagedPluginStatus `json:"pluginStatuses,omitempty"`
	// TotalPlugins is the number of Plugins in total managed by the PluginPreset.
	TotalPlugins int `json:"totalPlugins,omitempty"`
	// ReadyPlugins is the number of ready Plugins managed by the PluginPreset.
	ReadyPlugins int `json:"readyPlugins,omitempty"`
	// FailedPlugins is the number of failed Plugins managed by the PluginPreset.
	FailedPlugins int `json:"failedPlugins,omitempty"`
	// PluginDefinitionVersion is the version of the PluginDefinition referenced by this PluginPreset.
	PluginDefinitionVersion string `json:"pluginDefinitionVersion,omitempty"`
}

// ManagedPluginStatus defines the Ready condition of a managed Plugin identified by its name.
type ManagedPluginStatus struct {
	PluginName     string                           `json:"pluginName,omitempty"`
	ReadyCondition greenhousemetav1alpha1.Condition `json:"readyCondition,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pp
//+kubebuilder:printcolumn:name="Plugin Definition",type=string,JSONPath=`.spec.plugin.pluginDefinitionRef.name`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.pluginDefinitionVersion`
//+kubebuilder:printcolumn:name="Release Namespace",type=string,JSONPath=`.spec.plugin.releaseNamespace`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.statusConditions.conditions[?(@.type == "Ready")].status`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PluginPreset is the Schema for the PluginPresets API
type PluginPreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginPresetSpec   `json:"spec,omitempty"`
	Status PluginPresetStatus `json:"status,omitempty"`
}

func (c *PluginPreset) GetConditions() greenhousemetav1alpha1.StatusConditions {
	return c.Status.StatusConditions
}

func (c *PluginPreset) SetCondition(condition greenhousemetav1alpha1.Condition) {
	c.Status.SetConditions(condition)
}

func (c *PluginPreset) RemoveCondition(conditionType greenhousemetav1alpha1.ConditionType) {
	c.Status.Conditions = slices.DeleteFunc(c.Status.Conditions, func(cond greenhousemetav1alpha1.Condition) bool {
		return cond.Type == conditionType
	})
}

func (c *PluginPreset) CanBeSuspended() bool {
	return false
}

//+kubebuilder:object:root=true

// PluginPresetList contains a list of PluginPresets
type PluginPresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginPreset `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &PluginPreset{}, &PluginPresetList{})
		return nil
	})
}
