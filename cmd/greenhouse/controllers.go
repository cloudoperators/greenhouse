// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"sort"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	clustercontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	organizationcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	plugincontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/plugin"
	teamcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/team"
	teammembershipcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/teammembership"
	teamrbaccontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/teamrbac"
)

// knownControllers contains all controllers to be registered when starting the operator.
var knownControllers = map[string]func(controllerName string, mgr ctrl.Manager) error{
	// Organization controllers.
	"organizationController":     (&organizationcontrollers.OrganizationReconciler{}).SetupWithManager,
	"organizationRBAC":           (&organizationcontrollers.RBACReconciler{}).SetupWithManager,
	"organizationDEX":            startOrganizationDexReconciler,
	"organizationServiceProxy":   (&organizationcontrollers.ServiceProxyReconciler{}).SetupWithManager,
	"organizationTeamRoleSeeder": (&organizationcontrollers.TeamRoleSeederReconciler{}).SetupWithManager,

	// Team controllers.
	"teamPropagation": (&teamcontrollers.TeamPropagationReconciler{}).SetupWithManager,

	// TeamMembership controllers.
	"teamMembershipUpdater":     (&teammembershipcontrollers.TeamMembershipUpdaterController{}).SetupWithManager,
	"teamMembershipPropagation": (&teammembershipcontrollers.TeamMembershipPropagationReconciler{}).SetupWithManager,

	// Team RBAC controllers.
	"teamRoleBindingController": (&teamrbaccontrollers.TeamRoleBindingReconciler{}).SetupWithManager,

	// Plugin controllers.
	// "pluginPropagation": (&plugincontrollers.PluginPropagationReconciler{}).SetupWithManager,
	"pluginWorkLoadStatus": (&plugincontrollers.WorkLoadStatusReconciler{
		KubeRuntimeOpts: kubeClientOpts,
	}).SetupWithManager,

	// Plugin controllers.
	"pluginHelm": (&plugincontrollers.HelmReconciler{
		KubeRuntimeOpts: kubeClientOpts,
	}).SetupWithManager,
	"pluginPreset":    (&plugincontrollers.PluginPresetReconciler{}).SetupWithManager,
	"pluginChartTest": (&plugincontrollers.HelmChartTestReconciler{}).SetupWithManager,

	// Cluster controllers
	"bootStrap": (&clustercontrollers.BootstrapReconciler{}).SetupWithManager,
	"cluster":   startRemoteClusterReconciler,
	// "clusterPropagation":     (&clustercontrollers.ClusterPropagationReconciler{}).SetupWithManager,
	// "clusterHeadscaleAccess": startClusterHeadscaleAccessReconciler,
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
	for _, c := range enabledControllers {
		if controllerName == "*" || controllerName == c {
			return true
		}
	}
	return false
}

func startOrganizationDexReconciler(name string, mgr ctrl.Manager) error {
	namespace := "greenhouse"
	if v, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		namespace = v
	}
	return (&organizationcontrollers.DexReconciler{
		Namespace: namespace,
	}).SetupWithManager(name, mgr)
}

func startRemoteClusterReconciler(name string, mgr ctrl.Manager) error {
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

//nolint:unused
func startClusterHeadscaleAccessReconciler(name string, mgr ctrl.Manager) error {
	if renewRemoteClusterBearerTokenAfter > remoteClusterBearerTokenValidity {
		setupLog.Info("WARN: remoteClusterBearerTokenValidity is less than renewRemoteClusterBearerTokenAfter")
		setupLog.Info("Setting renewRemoteClusterBearerTokenAfter to half of remoteClusterBearerTokenValidity")
		renewRemoteClusterBearerTokenAfter = remoteClusterBearerTokenValidity / 2
	}
	if headscaleAPIKey == "" || headscaleAPIURL == "" {
		setupLog.Info("WARN: headscaleApiKey or headscaleApiUrl is not set")
		setupLog.Info("Skipping headscale access reconciler")
		return nil
	}

	if tailscaleProxy == "" {
		setupLog.Info("WARN: tailscaleProxy is not set")
		setupLog.Info("Skipping headscale access reconciler")
		return nil
	}

	return (&clustercontrollers.HeadscaleAccessReconciler{
		HeadscaleAPIKey:                          headscaleAPIKey,
		HeadscaleGRPCURL:                         headscaleAPIURL,
		TailscaleProxy:                           tailscaleProxy,
		HeadscalePreAuthenticationKeyMinValidity: 8 * time.Hour,
		RemoteClusterBearerTokenValidity:         remoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter:       renewRemoteClusterBearerTokenAfter,
	}).SetupWithManager(name, mgr)
}
