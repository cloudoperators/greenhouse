// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// GroupName for greenhouse API resources.
	GroupName = "greenhouse.sap"

	// SecretTypeKubeConfig specifies a secret containing the kubeconfig for a cluster.
	SecretTypeKubeConfig corev1.SecretType = "greenhouse.sap/kubeconfig"

	// SecretTypeOIDCConfig specifies a secret containing the OIDC configuration for a cluster.
	SecretTypeOIDCConfig corev1.SecretType = "greenhouse.sap/oidc"

	// SecretTypeOrganization specifies a secret containing the kubeconfig for an organization.
	SecretTypeOrganization corev1.SecretType = "greenhouse.sap/orgsecret"

	// LabelKeyOrgConfigMap is used to identify organizational config map.
	LabelKeyOrgConfigMap = "greenhouse.sap/orgconfigmap"

	// KubeConfigKey is the key for the user-provided kubeconfig in the secret of type greenhouse.sap/kubeconfig.
	KubeConfigKey = "kubeconfig"

	// GreenHouseKubeConfigKey is the key for the kubeconfig in the secret of type greenhouse.sap/kubeconfig used by Greenhouse.
	// This kubeconfig should be used by Greenhouse controllers and their kubernetes clients to access the remote cluster.
	GreenHouseKubeConfigKey = "greenhousekubeconfig"

	// LabelKeyPluginPreset is used to identify the PluginPreset managing the plugin.
	LabelKeyPluginPreset = "greenhouse.sap/pluginpreset"

	// LabelKeyPlugin is used to identify corresponding PluginDefinition for the resource.
	LabelKeyPlugin = "greenhouse.sap/plugin"

	// LabelKeyPluginDefinition is used to identify corresponding PluginDefinition for the resource.
	LabelKeyPluginDefinition = "greenhouse.sap/plugindefinition"

	// LabelKeyClusterPluginDefinition is used to identify the corresponding ClusterPluginDefinition for the resource.
	LabelKeyClusterPluginDefinition = "greenhouse.sap/clusterplugindefinition"

	// LabelKeyCluster is used to identify corresponding Cluster for the resource.
	LabelKeyCluster = "greenhouse.sap/cluster"

	// LabelKeyExposeService is applied to services that are part of a PluginDefinitions Helm chart to expose them via the central Greenhouse infrastructure.
	LabelKeyExposeService = "greenhouse.sap/expose"

	// LabelKeyExposeNamedPort is specifying the port to be exposed by name. LabelKeyExposeService needs to be set. Defaults to the first port if the named port is not found.
	LabelKeyExposeNamedPort = "greenhouse.sap/exposeNamedPort"

	// LabelKeyOwnedBy is used to identify the owning support-group team of a resource.
	LabelKeyOwnedBy = "greenhouse.sap/owned-by"
)

// TeamRole and TeamRoleBinding constants
const (
	// LabelKeyRoleBinding is the key of the label that is used to identify the RoleBinding.
	LabelKeyRoleBinding = "greenhouse.sap/rolebinding"

	// LabelKeyRole is the key of the label that is used to identify the Role.
	LabelKeyRole = "greenhouse.sap/role"

	// RBACPrefix is the prefix for the Role and RoleBinding names.
	RBACPrefix = "greenhouse:"

	// PluginClusterNameField is the field in the Plugin spec mapping it to a Cluster.
	PluginClusterNameField = ".spec.clusterName"

	// RolebindingTeamRoleRefField is the field in the RoleBinding spec that references the TeamRole.
	RolebindingTeamRoleRefField = ".spec.teamRoleRef"

	// RolebindingTeamRefField is the field in the RoleBinding spec that references the Team.
	RolebindingTeamRefField = ".spec.teamRef"

	// ConfigMapRefField is the field in the Organization spec that references the ConfigMap containing organizational configuration data.
	ConfigMapRefField = ".spec.configMapRef"
)

// Team constants
const (
	// LabelKeySupportGroup is the key of the label that is used to mark a Team as a support group (greenhouse.sap/support-group:true).
	LabelKeySupportGroup = "greenhouse.sap/support-group"
)

// cluster annotations
const (
	// MarkClusterDeletionAnnotation is used to mark a cluster for deletion.
	MarkClusterDeletionAnnotation = "greenhouse.sap/delete-cluster"
	// ScheduleClusterDeletionAnnotation is used to schedule a cluster for deletion.
	// Timestamp is set by mutating webhook if cluster is marked for deletion.
	ScheduleClusterDeletionAnnotation = "greenhouse.sap/deletion-schedule"
	ClusterConnectivityAnnotation     = "greenhouse.sap/cluster-connectivity"
	ClusterConnectivityKubeconfig     = "kubeconfig"
	ClusterConnectivityOIDC           = "oidc"
	GreenhouseHelmDeliveryToolLabel   = "greenhouse.sap/deployment-tool"
	GreenhouseHelmDeliveryToolFlux    = "flux"
)

const (
	SecretAPIServerURLAnnotation          = "oidc.greenhouse.sap/api-server-url"
	SecretAPIServerCAKey                  = "ca.crt"
	OIDCAudience                          = "greenhouse"
	SecretOIDCConfigGeneratedOnAnnotation = "oidc.greenhouse.sap/oidc-token-last-updated"
)
