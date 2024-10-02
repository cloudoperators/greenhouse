// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"os"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	clustercontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	organizationcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/organization"
	plugincontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/plugin"
	teamcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/team"
	teammembershipcontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/teammembership"
	teamrbaccontrollers "github.com/cloudoperators/greenhouse/pkg/controllers/teamrbac"
)

const (
	defaultRemoteClusterBearerTokenValidity   = 24 * time.Hour
	defaultRenewRemoteClusterBearerTokenAfter = 20 * time.Hour
)

// knownControllers contains all controllers to be registered when starting the e2e test suite

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
	"teamMembershipPropagation": (&teammembershipcontrollers.TeamMembershipPropagationReconciler{}).SetupWithManager,

	// Team RBAC controllers.
	"teamRoleBindingController": (&teamrbaccontrollers.TeamRoleBindingReconciler{}).SetupWithManager,

	// Plugin controllers.
	// "pluginPropagation": (&plugincontrollers.PluginPropagationReconciler{}).SetupWithManager,

	// Plugin controllers.
	"pluginHelm": (&plugincontrollers.HelmReconciler{
		KubeRuntimeOpts: clientutil.RuntimeOptions{QPS: 5, Burst: 10},
	}).SetupWithManager,
	"pluginPreset": (&plugincontrollers.PluginPresetReconciler{}).SetupWithManager,

	// Cluster controllers
	"bootStrap":           (&clustercontrollers.BootstrapReconciler{}).SetupWithManager,
	"clusterDirectAccess": startClusterDirectAccessReconciler,
	// "clusterPropagation":     (&clustercontrollers.ClusterPropagationReconciler{}).SetupWithManager,
	// "clusterHeadscaleAccess": startClusterHeadscaleAccessReconciler,
	"clusterStatus": (&clustercontrollers.ClusterStatusReconciler{}).SetupWithManager,
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

func startClusterDirectAccessReconciler(name string, mgr ctrl.Manager) error {
	return (&clustercontrollers.DirectAccessReconciler{
		RemoteClusterBearerTokenValidity:   defaultRemoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: defaultRenewRemoteClusterBearerTokenAfter,
	}).SetupWithManager(name, mgr)
}
