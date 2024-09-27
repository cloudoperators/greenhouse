// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package apis

import corev1 "k8s.io/api/core/v1"

const (
	// GroupName for greenhouse API resources.
	GroupName = "greenhouse.sap"

	// FinalizerCleanupHelmRelease is used to invoke the Helm release cleanup logic.
	FinalizerCleanupHelmRelease = "greenhouse.sap/helm"

	// FinalizerCleanupPluginPreset is used to invoke the PluginPreset cleanup logic.
	FinalizerCleanupPluginPreset = "greenhouse.sap/pluginpreset"

	// FinalizerCleanupCluster is used to invoke the cleanup of a registered cluster.
	FinalizerCleanupCluster = "greenhouse.sap/cluster"

	// FinalizerCleanupPropagatedResource is used to invoke the cleanup of remote resources.
	FinalizerCleanupPropagatedResource = "greenhouse.sap/propagatedResource"

	// SecretTypeKubeConfig specifies a secret containing the kubeconfig for a cluster.
	SecretTypeKubeConfig corev1.SecretType = "greenhouse.sap/kubeconfig"

	// KubeConfigKey is the key for the user-provided kubeconfig in the secret of type greenhouse.sap/kubeconfig.
	KubeConfigKey = "kubeconfig"

	// GreenHouseKubeConfigKey is the key for the kubeconfig in the secret of type greenhouse.sap/kubeconfig used by Greenhouse.
	// This kubeconfig should be used by Greenhouse controllers and their kubernetes clients to access the remote cluster.
	GreenHouseKubeConfigKey = "greenhousekubeconfig"

	// HeadscalePreAuthKey is the key for the Headscale pre-authentication key in a secret of type greenhouse.sap/kubeconfig.
	HeadscalePreAuthKey = "headscalePreAuthKey"

	// LabelKeyPluginPreset is used to identify the PluginPreset managing the plugin.
	LabelKeyPluginPreset = "greenhouse.sap/pluginpreset"

	// LabelKeyPlugin is used to identify corresponding PluginDefinition for the resource.
	LabelKeyPlugin = "greenhouse.sap/plugin"

	// LabelKeyPluginDefinition is used to identify corresponding PluginDefinition for the resource.
	LabelKeyPluginDefinition = "greenhouse.sap/plugindefinition"

	// LabelKeyCluster is used to identify corresponding Cluster for the resource.
	LabelKeyCluster = "greenhouse.sap/cluster"

	// HeadScaleKey is the key for the Headscale client deployment
	HeadScaleKey = "greenhouse.sap/headscale"

	// LabelAccessMode is used to force the access mode to headscale for a cluster.
	LabelAccessMode = "greenhouse.sap/access-mode"

	// LabelKeyExposeService is applied to services that are part of a PluginDefinitions Helm chart to expose them via the central Greenhouse infrastructure.
	LabelKeyExposeService = "greenhouse.sap/expose"

	// LabelKeyExposeNamedPort is specifying the port to be exposed by name. LabelKeyExposeService needs to be set. Defaults to the first port if the named port is not found.
	LabelKeyExposeNamedPort = "greenhouse.sap/exposeNamedPort"
)

// TeamRole and TeamRoleBinding constants
const (
	// FinalizerCleanupTeamRoleBinding is used to invoke the RoleBinding release cleanup logic.
	FinalizerCleanupTeamRoleBinding = "greenhouse.sap/rolebinding"

	// FinalizerCleanupRole is used to invoke the Role release cleanup logic.
	FinalizerCleanupRole = "greenhouse.sap/role"

	// LabelKeyRoleBinding is the key of the label that is used to identify the RoleBinding.
	LabelKeyRoleBinding = "greenhouse.sap/rolebinding"

	// LabelKeyRole is the key of the label that is used to identify the Role.
	LabelKeyRole = "greenhouse.sap/role"

	// RBACPrefix is the prefix for the Role and RoleBinding names.
	RBACPrefix = "greenhouse:"

	// PluginClusterNameField is the field in the Plugin spec mapping it to a Cluster.
	PluginClusterNameField = ".spec.clusterName"

	// RolebindingRoleRefField is the field in the RoleBinding spec that references the Role.
	RolebindingRoleRefField = ".spec.roleRef"

	// RolebindingTeamRefField is the field in the RoleBinding spec that references the Team.
	RolebindingTeamRefField = ".spec.teamRef"
)

// cluster deletion annotations
const (
	// MarkClusterDeletionAnnotation is used to mark a cluster for deletion.
	MarkClusterDeletionAnnotation = "greenhouse.sap/delete-cluster"
	// ScheduleClusterDeletionAnnotation is used to schedule a cluster for deletion.
	// Timestamp is set by mutating webhook if cluster is marked for deletion.
	ScheduleClusterDeletionAnnotation = "greenhouse.sap/deletion-schedule"
)
