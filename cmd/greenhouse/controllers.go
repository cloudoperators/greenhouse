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
	"organizationController": startOrganizationReconciler,

	// Team controllers.
	"teamController": (&teamcontrollers.TeamController{}).SetupWithManager,

	// Team RBAC controllers.
	"teamRoleBindingController": (&teamrbaccontrollers.TeamRoleBindingReconciler{}).SetupWithManager,

	// Plugin controllers.
	"plugin":       startPluginReconciler,
	"pluginPreset": (&plugincontrollers.PluginPresetReconciler{}).SetupWithManager,

	"catalog":                 startCatalogReconciler,
	"pluginDefinition":        (&plugindefinitioncontroller.PluginDefinitionReconciler{}).SetupWithManager,
	"clusterPluginDefinition": (&plugindefinitioncontroller.ClusterPluginDefinitionReconciler{}).SetupWithManager,

	// Cluster controllers
	"bootStrap":         (&clustercontrollers.BootstrapReconciler{}).SetupWithManager,
	"clusterReconciler": startClusterReconciler,
	"kubeconfig":        (&clustercontrollers.KubeconfigReconciler{}).SetupWithManager,
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

// startOrganizationReconciler - initializes the organization reconciler
// resolves dex storage backend from greenhouse-feature-flags
// initializes the dex storage adapter interface in the organization reconciler
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
// Resolves expression evaluation and default deployment tool feature flags from greenhouse-feature-flags.
func startPluginReconciler(name string, mgr ctrl.Manager) error {
	return (&plugincontrollers.PluginReconciler{
		KubeRuntimeOpts:             kubeClientOpts,
		ExpressionEvaluationEnabled: featureFlags.IsExpressionEvaluationEnabled(),
		IntegrationEnabled:          featureFlags.IsIntegrationEnabled(),
		DefaultDeploymentTool:       featureFlags.GetDefaultDeploymentTool(),
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
	}).SetupWithManager(name, mgr)
}

func startCatalogReconciler(name string, mgr ctrl.Manager) error {
	return (&catalog.CatalogReconciler{
		Log:         ctrl.Log.WithName("controllers").WithName("catalogs"),
		StoragePath: artifactStoragePath,
		HttpRetry:   artifactRetries,
	}).SetupWithManager(name, mgr)
}
