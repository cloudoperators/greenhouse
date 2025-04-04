// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/x509"
	"slices"
	"strings"

	"sort"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
	clustercontrollers "github.com/cloudoperators/greenhouse/internal/controller/cluster"
	organizationcontrollers "github.com/cloudoperators/greenhouse/internal/controller/organization"
	plugincontrollers "github.com/cloudoperators/greenhouse/internal/controller/plugin"
	teammembershipcontrollers "github.com/cloudoperators/greenhouse/internal/controller/teammembership"
	teamrbaccontrollers "github.com/cloudoperators/greenhouse/internal/controller/teamrbac"
	dexstore "github.com/cloudoperators/greenhouse/internal/dex"
)

// knownControllers contains all controllers to be registered when starting the operator.
var knownControllers = map[string]func(controllerName string, mgr ctrl.Manager) error{
	// Certificate generation for webhooks.
	"cert-controller": startCertController,

	// Organization controllers.
	"organizationController": startOrganizationReconciler,

	// TeamMembership controllers.
	"teamMembershipUpdater": (&teammembershipcontrollers.TeamMembershipUpdaterController{}).SetupWithManager,

	// Team RBAC controllers.
	"teamRoleBindingController": (&teamrbaccontrollers.TeamRoleBindingReconciler{}).SetupWithManager,

	// Plugin controllers.
	"plugin": (&plugincontrollers.PluginReconciler{
		KubeRuntimeOpts: kubeClientOpts,
	}).SetupWithManager,
	"pluginPreset": (&plugincontrollers.PluginPresetReconciler{}).SetupWithManager,

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
	for _, c := range enabledControllers {
		if controllerName == "*" || controllerName == c {
			return true
		}
	}
	return false
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

func loadExtraDnsNamesForCertificate(mgr ctrl.Manager) []string {
	allDnsNames := []string{"greenhouse-webhook-service.greenhouse.svc.cluster.local"}

	setupLog.Info("Loading ExtraDNSNames from ConfigMap...")

	configMap := &corev1.ConfigMap{}
	// Note: mgr.GetClient() will fail here because the cache is not ready yet.
	k8sclient := mgr.GetAPIReader()
	err := k8sclient.Get(context.TODO(), types.NamespacedName{Name: "greenhouse-webhook-server-config", Namespace: "greenhouse"}, configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			setupLog.Info("ConfigMap not found. " + err.Error())
			return allDnsNames
		}
		setupLog.Info("failed to load ConfigMap for WebhookServer, leaving default DNSNames")
		return allDnsNames
	}

	value, exists := configMap.Data["extraDnsNames"]
	if exists {
		extraDNSNames := strings.Split(value, "\n")
		extraDNSNames = slices.DeleteFunc(extraDNSNames, func(name string) bool {
			return name == "" // Remove empty strings left after Split.
		})

		allDnsNames = append(allDnsNames, extraDNSNames...)

		setupLog.Info("Loaded cert-controller local DNS names (" + strconv.Itoa(len(extraDNSNames)) + "):")
		for i, v := range extraDNSNames {
			setupLog.Info("value" + strconv.Itoa(i) + " = '" + v + "'")
		}
	}
	return allDnsNames
}

func startCertController(_ string, mgr ctrl.Manager) error {
	setupFinished := make(chan struct{})

	setupLog.Info("Loading ExtraDNSNames...")
	extraDNSNames := loadExtraDnsNamesForCertificate(mgr)
	setupLog.Info("Loaded " + strconv.Itoa(len(extraDNSNames)) + " ExtraDNSNames.")

	if err := rotator.AddRotator(mgr, &rotator.CertRotator{
		Webhooks: []rotator.WebhookInfo{
			{Name: "greenhouse-validating-webhook-configuration", Type: rotator.Validating},
			{Name: "greenhouse-mutating-webhook-configuration", Type: rotator.Mutating},
			{Name: "teamrolebindings.greenhouse.sap", Type: rotator.CRDConversion},
		},
		IsReady: setupFinished,
		SecretKey: types.NamespacedName{
			Namespace: "greenhouse",
			Name:      "greenhouse-webhook-server-cert",
		},
		RequireLeaderElection: false,
		CertDir:               "/tmp/k8s-webhook-server/serving-certs",
		CAName:                "greenhouse-ca", // Used for CA certificate Subject.
		CAOrganization:        "greenhouse",    // Used for CA certificate Subject.
		DNSName:               "greenhouse-webhook-service.greenhouse.svc",
		ExtraDNSNames:         extraDNSNames, //[]string{"greenhouse-webhook-service.greenhouse.svc.cluster.local"}, //, "172.17.0.1"
		ExtKeyUsages:          &[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// RestartOnSecretRefresh: true,
		// Optional with default values:
		// CertName:               "tls.crt",
		// KeyName:                "tls.key",
		// CaCertDuration:         10 * 365 * 24 * time.Hour,
		// ServerCertDuration:     1 * 365 * 24 * time.Hour,
		// LookaheadInterval:      90 * 24 * time.Hour,
		// RotationCheckFrequency: 90 * 24 * time.Hour,
	}); err != nil {
		setupLog.Error(err, "unable to set up cert rotation")
		return err
	}
	// Block until the setup (certificate generation) finishes.
	// <-setupFinished
	return nil
}
