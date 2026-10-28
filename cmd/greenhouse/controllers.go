// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"slices"
	"sort"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/controller/catalog"
	clustercontrollers "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	organizationcontrollers "github.com/cloudoperators/greenhouse/internal/controller/organization"
	plugincontrollers "github.com/cloudoperators/greenhouse/internal/controller/plugin"
	plugindefinitioncontroller "github.com/cloudoperators/greenhouse/internal/controller/plugindefinition"
	teamcontrollers "github.com/cloudoperators/greenhouse/internal/controller/team"
	teamrbaccontrollers "github.com/cloudoperators/greenhouse/internal/controller/teamrbac"
	dexstore "github.com/cloudoperators/greenhouse/internal/dex"
)

// knownControllers contains all controllers to be registered when starting the operator.
var knownControllers = map[string]func(controllerName string, mgr ctrl.Manager) error{
	// Organization controllers.
	"organization": startOrganizationReconciler,

	// Team controllers.
	"team": startTeamReconciler,

	// Team RBAC controllers.
	"teamRoleBinding": startTeamRoleBindingReconciler,

	// Plugin controllers.
	"plugin":       startPluginReconciler,
	"pluginPreset": startPluginPresetReconciler,

	"catalog":                 startCatalogReconciler,
	"pluginDefinition":        startPluginDefinitionReconciler,
	"clusterPluginDefinition": startClusterPluginDefinitionReconciler,

	// Cluster controllers
	"bootstrap":  startBootstrapReconciler,
	"cluster":    startClusterReconciler,
	"kubeconfig": startKubeconfigReconciler,
}

// knownControllers lists the name of known controllers.
func knownControllersNames() []string {
	controllerStringSlice := make([]string, 0)
	for controllerName := range knownControllers {
		controllerStringSlice = append(controllerStringSlice, controllerName)
	}
	sort.Strings(controllerStringSlice)
	return controllerStringSlice
}

// isControllerEnabled checks whether the given controller or regex is enabled
func isControllerEnabled(controllerName string) bool {
	if slices.Contains(disabledControllers, controllerName) {
		return false
	}
	return controllerName == "*" || slices.Contains(enabledControllers, controllerName)
}

func startTeamReconciler(name string, mgr ctrl.Manager) error {
	return (&teamcontrollers.TeamController{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
	}).SetupWithManager(name, mgr)
}

func startTeamRoleBindingReconciler(name string, mgr ctrl.Manager) error {
	return (&teamrbaccontrollers.TeamRoleBindingReconciler{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
	}).SetupWithManager(name, mgr)
}

func startKubeconfigReconciler(name string, mgr ctrl.Manager) error {
	return (&clustercontrollers.KubeconfigReconciler{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
	}).SetupWithManager(name, mgr)
}

func startOrganizationReconciler(name string, mgr ctrl.Manager) error {
	namespace := clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace)
	backend := ptr.To(dexstore.K8s)
	if featureFlags != nil {
		backend = featureFlags.GetDexStorageType(context.Background())
	}
	return (&organizationcontrollers.OrganizationReconciler{
		Namespace:      namespace,
		DexStorageType: *backend,
	}).SetupWithManager(name, mgr)
}

// startPluginReconciler initializes the plugin reconciler.
func startPluginReconciler(name string, mgr ctrl.Manager) error {
	return (&plugincontrollers.PluginReconciler{
		KubeRuntimeOpts:         kubeClientOpts,
		OCIMirroringEnabled:     featureFlags.IsOCIMirroringEnabled(),
		StoragePath:             artifactStoragePath,
		HTTPRetry:               artifactRetries,
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 10),
		WorkloadIdentityEnabled: featureFlags.IsWorkloadIdentityEnabled(),
		ReleaseEventsEnabled:    featureFlags.IsReleaseEventsEnabled(),
		FeatureFlagsName:        clientutil.GetEnvOrDefault(featureFlagsEnv, defaultFeatureFlagConfigMapName),
		FeatureFlagsNamespace:   clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace),
	}).SetupWithManager(name, mgr)
}

func startPluginPresetReconciler(name string, mgr ctrl.Manager) error {
	return (&plugincontrollers.PluginPresetReconciler{
		MaxConcurrentReconciles:     managerConfig.GetMaxConcurrentReconciles(name, 1),
		ExpressionEvaluationEnabled: featureFlags.IsPresetExpressionEvaluationEnabled(),
		IntegrationEnabled:          featureFlags.IsPresetIntegrationEnabled(),
	}).SetupWithManager(name, mgr)
}

func startPluginDefinitionReconciler(name string, mgr ctrl.Manager) error {
	return (&plugindefinitioncontroller.PluginDefinitionReconciler{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
		OCIMirroringEnabled:     featureFlags.IsOCIMirroringEnabled(),
	}).SetupWithManager(name, mgr)
}

func startClusterPluginDefinitionReconciler(name string, mgr ctrl.Manager) error {
	return (&plugindefinitioncontroller.ClusterPluginDefinitionReconciler{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
		OCIMirroringEnabled:     featureFlags.IsOCIMirroringEnabled(),
	}).SetupWithManager(name, mgr)
}

func startClusterReconciler(name string, mgr ctrl.Manager) error {
	if renewRemoteClusterBearerTokenAfter > remoteClusterBearerTokenValidity {
		setupLog.Info("WARN: remoteClusterBearerTokenValidity is less than renewRemoteClusterBearerTokenAfter")
		setupLog.Info("Setting renewRemoteClusterBearerTokenAfter to half of remoteClusterBearerTokenValidity")
		renewRemoteClusterBearerTokenAfter = remoteClusterBearerTokenValidity / 2
	}
	return (&clustercontrollers.RemoteClusterReconciler{
		RemoteClusterBearerTokenValidity:   remoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: renewRemoteClusterBearerTokenAfter,
		MaxConcurrentReconciles:            managerConfig.GetMaxConcurrentReconciles(name, 5),
		WorkloadIdentityEnabled:            featureFlags.IsWorkloadIdentityEnabled(),
		FeatureFlagsName:                   clientutil.GetEnvOrDefault(featureFlagsEnv, defaultFeatureFlagConfigMapName),
		FeatureFlagsNamespace:              clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace),
	}).SetupWithManager(name, mgr)
}

func startBootstrapReconciler(name string, mgr ctrl.Manager) error {
	return (&clustercontrollers.BootstrapReconciler{
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 1),
		WorkloadIdentityEnabled: featureFlags.IsWorkloadIdentityEnabled(),
		FeatureFlagsName:        clientutil.GetEnvOrDefault(featureFlagsEnv, defaultFeatureFlagConfigMapName),
		FeatureFlagsNamespace:   clientutil.GetEnvOrDefault(podNamespaceEnv, defaultPodNamespace),
	}).SetupWithManager(name, mgr)
}

func startCatalogReconciler(name string, mgr ctrl.Manager) error {
	return (&catalog.CatalogReconciler{
		Log:                     ctrl.Log.WithName("controllers").WithName("catalogs"),
		StoragePath:             artifactStoragePath,
		HTTPRetry:               artifactRetries,
		MaxConcurrentReconciles: managerConfig.GetMaxConcurrentReconciles(name, 5),
	}).SetupWithManager(name, mgr)
}
